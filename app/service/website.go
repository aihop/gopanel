package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
)

func NewWebsite() *WebsiteService {
	return &WebsiteService{
		repo: repo.NewWebsite(),
	}
}

type WebsiteService struct {
	repo *repo.WebsiteRepo
}

func (s WebsiteService) Create(req *request.WebsiteCreate, mode model.DatabaseMode) (err error) {
	alias := req.Alias
	if alias == "default" {
		return buserr.New("ErrDefaultAlias")
	}
	if common.ContainsChinese(alias) {
		alias, err = common.PunycodeEncode(alias)
		if err != nil {
			return
		}
	}
	websiteRepo := repo.NewWebsite()
	if exist, _ := websiteRepo.GetBy(websiteRepo.WithAlias(alias)); len(exist) > 0 {
		return errors.New("网站目录、别名已存在")
	}
	defaultHttpPort := 80
	var (
		otherDomains []model.WebsiteDomain
		domains      []model.WebsiteDomain
	)
	if strings.HasPrefix(req.PrimaryDomain, "http://") {
		req.Protocol = constant.ProtocolHTTP
	}

	req.PrimaryDomain = strings.TrimPrefix(req.PrimaryDomain, "https://")
	req.PrimaryDomain = strings.TrimPrefix(req.PrimaryDomain, "http://")

	if isIP(req.PrimaryDomain) && req.Protocol == "" {
		req.Protocol = constant.ProtocolHTTP
	}

	domains, _, _, err = getWebsiteDomains(req.PrimaryDomain, defaultHttpPort, 0)
	if err != nil {
		return errors.New("primary domain error: " + err.Error())
	}
	otherDomains, _, _, err = getWebsiteDomains(req.OtherDomains, defaultHttpPort, 0)
	if err != nil {
		return errors.New("other domains error: " + err.Error())
	}
	domains = append(domains, otherDomains...)
	if req.Protocol == "" {
		req.Protocol = constant.ProtocolHTTPS
	}

	fmt.Println(req.Protocol, "req.Protocol")

	defaultDate, _ := time.Parse(constant.DateLayout, constant.DefaultDate)
	website := &model.Website{
		PrimaryDomain: req.PrimaryDomain,
		Type:          req.Type,
		Alias:         alias,
		Remark:        req.Remark,
		Status:        constant.WebRunning,
		ExpireDate:    defaultDate,
		Protocol:      req.Protocol,
		Proxy:         req.Proxy,
		SiteDir:       "/",
		CodeSource:    req.CodeSource,
		AccessLog:     true,
		ErrorLog:      true,
		IPV6:          req.IPV6,
		PipelineID:    req.PipelineId,
	}

	var appInstall *model.AppInstall
	defer func() {
		if err != nil && website.AppInstallID > 0 {
			deleteReq := request.AppInstalledOperate{
				InstallId:   website.AppInstallID,
				Operate:     constant.Delete,
				ForceDelete: true,
			}
			if deleteErr := NewAppInstall().Operate(deleteReq); deleteErr != nil {
				global.LOG.Errorf(deleteErr.Error())
			}
		}
	}()

	caddyDomain := BuildWebsiteCaddyDomain(req.PrimaryDomain, req.Protocol)
	staticRoot := resolveWebsiteStaticRoot(alias)

	if req.CodeSource == "app_store" {
		req.Type = constant.Proxy
		var install model.AppInstall
		install, err = appInstallRepo.GetFirst(commonRepo.WithByID(req.AppInstallID))
		if err != nil {
			return err
		}
		appInstall = &install
		website.AppInstallID = appInstall.ID
		if req.Proxy != "" {
			website.Proxy = req.Proxy
		} else {
			website.Proxy = fmt.Sprintf("127.0.0.1:%d", appInstall.HttpPort)
		}
		website.Type = constant.Proxy
	}

	switch req.Type {
	case constant.WebApp:
		codeDir := req.CodeDir
		if codeDir == "" {
			codeDir = filepath.Join(global.CONF.System.BaseDir, "wwwroot", alias)
		}

		if req.CodeSource == "pipeline" {
			website.Status = "Pending"
			website.ContainerID = ""
			website.EngineEnv = ""
			if req.Proxy != "" {
				website.Proxy = req.Proxy
			} else {
				website.Proxy = "http://127.0.0.1:80"
			}
			global.LOG.Infof("Website created with pipeline source. Waiting for pipeline execution.")
		} else {
			hostPort, containerID, runtimeDir, err := DeployWebsiteEngine(context.Background(), alias, req, nil)
			if err != nil {
				return fmt.Errorf("failed to deploy container: %w", err)
			}
			website.Proxy = fmt.Sprintf("127.0.0.1:%d", hostPort)
			website.ContainerID = containerID
			website.EngineEnv = req.GitRepo
			website.RuntimeDir = runtimeDir
			website.Status = "Running"
			global.LOG.Infof("Deployed custom container %s on port %d", containerID, hostPort)
		}
	}

	if req.Type == constant.Proxy || req.Type == constant.WebApp {
		caddyProxy := website.Proxy
		if caddyProxy == "" {
			caddyProxy = req.Proxy
		}
		if strings.HasPrefix(alias, "/") {
			_, err = NewCaddy().AddServerPathBlock(caddyDomain, alias, caddyProxy, req.OtherDomains, req.Protocol)
		} else {
			_, err = NewCaddy().AddServerBlock(caddyDomain, caddyProxy, req.OtherDomains, req.Protocol)
		}
		if err != nil {
			return err
		}
	}
	if req.Type == constant.Static {
		website.SiteDir = staticRoot
		if err = ensureStaticWebsiteIndex(staticRoot); err != nil {
			return err
		}
		_, err = NewCaddy().AddStaticServerBlock(caddyDomain, staticRoot, req.OtherDomains, req.Protocol)
		if err != nil {
			return err
		}
	}
	tx := global.DB.Begin()
	defer tx.Rollback()
	if err = websiteRepo.Create(context.Background(), website); err != nil {
		return err
	}
	for i := range domains {
		domains[i].WebsiteID = website.ID
	}
	websiteDomainRepo := repo.NewWebsiteDomain()
	if err = websiteDomainRepo.BatchCreate(context.Background(), domains); err != nil {
		return err
	}

	// 如果是 git (自定义镜像部署)，初始部署成功后，生成一条发布记录
	if req.CodeSource == "git" && website.Type == constant.WebApp && website.ContainerID != "" {
		version := fmt.Sprintf("v%d", time.Now().Unix())
		deploy := model.WebsiteDeploy{
			WebsiteID:   website.ID,
			Version:     version,
			SourceType:  "git",
			ImageTag:    req.GitRepo,
			Status:      "Running",
			LogText:     "初始化部署自定义镜像: " + req.GitRepo + "\n",
			ContainerID: website.ContainerID,
			Port:        0, // 这里虽然没存 proxy 的具体端口数值，但可以通过网站本身拿到
			IsActive:    true,
			RuntimeDir:  website.RuntimeDir,
		}
		if err := global.DB.Create(&deploy).Error; err != nil {
			global.LOG.Errorf("Failed to create initial website deploy record for git image: %v", err)
		}
	}

	tx.Commit()
	return
}
