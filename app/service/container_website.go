package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	containertypes "github.com/docker/docker/api/types"
)

var applyContainerWebsiteCaddy = ApplyCaddyFromDB

type containerWebsiteTarget struct {
	ContainerID          string
	RuntimeHost          string
	WebsiteID            uint
	HostPort             int
	Scheme               string
	Address              string
	DeploymentSourceType string
}

func (u *ContainerService) BindWebsite(ctx context.Context, req *dto.ContainerWebsiteBind) error {
	if req == nil {
		return buserr.New(constant.ErrContainerWebsiteBindEmpty)
	}
	target, err := resolveContainerWebsiteTarget(ctx, req)
	if err != nil {
		return err
	}
	return bindContainerTargetToWebsite(ctx, target, "")
}

func resolveContainerWebsiteTarget(ctx context.Context, req *dto.ContainerWebsiteBind) (containerWebsiteTarget, error) {
	lookup, err := newContainerRuntimeLookup(ctx)
	if err != nil {
		return containerWebsiteTarget{}, buserr.WithDetail(constant.ErrContainerWebsiteRuntimeLoadFailed, err.Error())
	}
	defer lookup.Close()

	containerID := strings.TrimSpace(req.ContainerID)
	item, ok := lookup.containersByID[containerID]
	if !ok {
		return containerWebsiteTarget{}, buserr.New(constant.ErrContainerWebsiteNotFound)
	}
	if !strings.EqualFold(strings.TrimSpace(item.State), "running") {
		return containerWebsiteTarget{}, buserr.New(constant.ErrContainerWebsiteNotRunning)
	}

	runtimeHost := strings.TrimSpace(lookup.sourceByID[item.ID])
	if runtimeHost == "" {
		runtimeHost = strings.TrimSpace(lookup.defaultMeta.RuntimeHost)
	}
	requestedHost := strings.TrimSpace(req.RuntimeHost)
	if requestedHost != "" && runtimeHost != requestedHost {
		return containerWebsiteTarget{}, buserr.New(constant.ErrContainerWebsiteStale)
	}

	address, err := publishedContainerAddress(item.Ports, req.HostPort)
	if err != nil {
		return containerWebsiteTarget{}, err
	}
	scheme := strings.ToLower(strings.TrimSpace(req.Scheme))
	if scheme == "" {
		scheme = "http"
	}
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := checkContainerWebsiteEndpoint(checkCtx, scheme, address); err != nil {
		return containerWebsiteTarget{}, err
	}
	return containerWebsiteTarget{
		ContainerID: item.ID,
		RuntimeHost: runtimeHost,
		WebsiteID:   req.WebsiteID,
		HostPort:    req.HostPort,
		Scheme:      scheme,
		Address:     address,
	}, nil
}

func publishedContainerAddress(ports []containertypes.Port, hostPort int) (string, error) {
	var fallback string
	for _, item := range ports {
		if int(item.PublicPort) != hostPort || !strings.EqualFold(item.Type, "tcp") {
			continue
		}
		host := strings.TrimSpace(item.IP)
		switch host {
		case "", "0.0.0.0", "::":
			host = "127.0.0.1"
		}
		address := net.JoinHostPort(host, strconv.Itoa(hostPort))
		if !strings.Contains(host, ":") {
			return address, nil
		}
		fallback = address
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", buserr.New(constant.ErrContainerWebsitePortMismatch)
}

func checkContainerWebsiteEndpoint(ctx context.Context, scheme, address string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+address+"/", nil)
	if err != nil {
		return err
	}
	client := http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return buserr.WithMap(constant.ErrContainerWebsiteUpstreamUnreachable, map[string]interface{}{
			"scheme":  scheme,
			"address": address,
			"detail":  err.Error(),
		})
	}
	closeErr := response.Body.Close()
	if response.StatusCode >= http.StatusInternalServerError {
		return buserr.WithMap(constant.ErrContainerWebsiteUpstreamBadStatus, map[string]interface{}{
			"scheme":  scheme,
			"address": address,
			"status":  response.StatusCode,
		})
	}
	return closeErr
}

func bindContainerTargetToWebsite(ctx context.Context, target containerWebsiteTarget, version string) error {
	websiteRepo := repo.NewWebsite()
	website, err := websiteRepo.GetFirst(websiteRepo.WithID(target.WebsiteID))
	if err != nil {
		return buserr.New(constant.ErrContainerWebsiteMissing)
	}
	if website.Type != constant.Proxy {
		return buserr.New(constant.ErrContainerWebsiteNotReverseProxy)
	}
	if website.AppInstallID > 0 {
		return buserr.New(constant.ErrContainerWebsiteAppStoreForbidden)
	}

	upstream := model.WebsiteUpstream{
		WebsiteID: website.ID,
		Address:   target.Address,
		Scheme:    target.Scheme,
		Weight:    1,
		Enabled:   true,
		Transport: target.Scheme,
	}
	website.Proxy = buildWebsiteUpstreamDial(upstream)
	website.ContainerID = target.ContainerID
	website.CodeSource = "container"
	website.Status = constant.WebRunning

	tx := global.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()
	txCtx := context.WithValue(ctx, constant.DB, tx)
	if err := websiteRepo.Save(txCtx, &website); err != nil {
		return err
	}
	if err := repo.NewWebsiteUpstream().ReplaceByWebsiteID(txCtx, website.ID, []model.WebsiteUpstream{upstream}); err != nil {
		return err
	}
	sourceType := strings.TrimSpace(target.DeploymentSourceType)
	if sourceType == "" {
		sourceType = "container_bind"
	}
	deploy := &model.AppDeploy{
		WebsiteID:   website.ID,
		Version:     flowContainerDeploymentVersion(version),
		SourceType:  sourceType,
		SourceUrl:   website.Proxy,
		Status:      constant.WebRunning,
		LogText:     fmt.Sprintf("绑定容器 %s 的宿主端口 %d 到网站反向代理\n", target.ContainerID, target.HostPort),
		ContainerID: target.ContainerID,
		RuntimeHost: target.RuntimeHost,
		Port:        target.HostPort,
		IsActive:    true,
	}
	if err := repo.NewAppDeploy(global.DB).ReplaceActiveContainerBinding(tx, deploy); err != nil {
		return err
	}
	if err := applyContainerWebsiteCaddy(txCtx); err != nil {
		return buserr.WithDetail(constant.ErrContainerWebsiteApplyFailed, err.Error())
	}
	return tx.Commit().Error
}

func flowContainerDeploymentVersion(version string) string {
	version = strings.TrimSpace(version)
	if version != "" {
		return version
	}
	return fmt.Sprintf("container-%d", time.Now().Unix())
}
