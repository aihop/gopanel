package service

import (
	"context"
	"errors"
	"fmt"
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

func normalizeWebsiteProtocol(protocol string) string {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case constant.ProtocolHTTP, constant.Http:
		return constant.ProtocolHTTP
	case constant.ProtocolHTTPS:
		return constant.ProtocolHTTPS
	default:
		return ""
	}
}

func (s WebsiteService) Create(ctx context.Context, req *request.WebsiteCreate, mode model.DatabaseMode) (err error) {
	if req.CodeSource == "pipeline" {
		return errors.New("网站不再直接关联流水线，请先启动容器，再从容器列表发布到网站")
	}
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
		upstreams    []model.WebsiteUpstream
	)
	req.Protocol = normalizeWebsiteProtocol(req.Protocol)
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
	if req.Type == constant.Proxy {
		upstreams, err = ensureWebsiteProxyUpstreams(req.Upstreams, req.Proxy)
		if err != nil {
			return err
		}
		req.Proxy = websiteProxyFromUpstreams(upstreams, req.Proxy)
	}

	defaultDate, _ := time.Parse(constant.DateLayout, constant.DefaultDate)
	website := &model.Website{
		PrimaryDomain:            req.PrimaryDomain,
		Type:                     req.Type,
		Alias:                    alias,
		Remark:                   req.Remark,
		Status:                   constant.WebRunning,
		ExpireDate:               defaultDate,
		Protocol:                 req.Protocol,
		Proxy:                    req.Proxy,
		SiteDir:                  "/",
		CodeSource:               req.CodeSource,
		AccessLog:                true,
		ErrorLog:                 true,
		IPV6:                     req.IPV6,
		AntiCrawler:              req.AntiCrawler,
		AntiLeech:                req.AntiLeech,
		RateLimitMode:            req.RateLimitMode,
		WafEnable:                req.WafEnable,
		BlockSensitive:           req.BlockSensitive,
		IPAllowlist:              strings.TrimSpace(req.IPAllowlist),
		IPBlocklist:              strings.TrimSpace(req.IPBlocklist),
		SecurityHeader:           req.SecurityHeader,
		HstsEnabled:              req.HstsEnabled,
		HttpConfig:               req.HttpConfig,
		RedirectCode:             req.RedirectCode,
		RedirectDomainsToPrimary: req.RedirectDomainsToPrimary,
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
		if len(req.Upstreams) > 0 {
			upstreams, err = ensureWebsiteProxyUpstreams(req.Upstreams, website.Proxy)
			if err != nil {
				return err
			}
			website.Proxy = websiteProxyFromUpstreams(upstreams, website.Proxy)
		}
	}

	switch req.Type {
	case constant.WebApp:
		if err = validateWebsiteImageSource(req.CodeSource); err != nil {
			return err
		}
		deployRequest := websiteImageDeployRequest{Alias: alias, Image: req.GitRepo}
		hostPort, containerID, deployErr := deployWebsiteImage(context.Background(), deployRequest, nil)
		if deployErr != nil {
			return fmt.Errorf("failed to deploy container: %w", deployErr)
		}
		website.Proxy = fmt.Sprintf("127.0.0.1:%d", hostPort)
		website.ContainerID = containerID
		website.EngineEnv = req.GitRepo
		website.RuntimeDir = ""
		website.Status = "Running"
		global.LOG.Infof("Deployed custom container %s on port %d", containerID, hostPort)
	}

	if req.Type == constant.Static {
		website.SiteDir = staticRoot
		if err = ensureStaticWebsiteIndex(staticRoot); err != nil {
			return err
		}
	}
	if _, err = ensureWebsiteTrackingDirs(website.Alias); err != nil {
		return err
	}
	tx := global.DB.Begin()
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()
	txCtx := context.WithValue(ctx, constant.DB, tx)
	if err = websiteRepo.Create(txCtx, website); err != nil {
		return err
	}
	for i := range domains {
		domains[i].WebsiteID = website.ID
	}
	websiteDomainRepo := repo.NewWebsiteDomain()
	if err = websiteDomainRepo.BatchCreate(txCtx, domains); err != nil {
		return err
	}
	if website.Type == constant.Proxy {
		for i := range upstreams {
			upstreams[i].WebsiteID = website.ID
		}
		if err = repo.NewWebsiteUpstream().BatchCreate(txCtx, upstreams); err != nil {
			return err
		}
	}

	// 如果是 git (自定义镜像部署)，初始部署成功后，生成一条发布记录
	if req.CodeSource == "git" && website.Type == constant.WebApp && website.ContainerID != "" {
		version := fmt.Sprintf("v%d", time.Now().Unix())
		deploy := model.AppDeploy{
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
		if createDeployErr := tx.Create(&deploy).Error; createDeployErr != nil {
			global.LOG.Errorf("Failed to create initial website deploy record for git image: %v", createDeployErr)
		}
	}

	if err = ApplyCaddyFromDB(txCtx); err != nil {
		return err
	}
	if err = tx.Commit().Error; err != nil {
		return err
	}
	tx = nil
	return nil
}
