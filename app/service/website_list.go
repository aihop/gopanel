package service

import (
	"context"
	"path"
	"strings"

	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/gormx"
)

func (s *WebsiteService) List(ctx *gormx.Contextx) (websiteDTOs []*response.WebsiteRes, err error) {
	_ = s.SyncFromCaddyfile()
	res, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return []*response.WebsiteRes{}, nil
	}
	websiteIDs := make([]uint, 0, len(res))
	for _, web := range res {
		websiteIDs = append(websiteIDs, web.ID)
	}
	diagnosticSummaries, err := loadWebsiteDiagnosticSummaries(websiteIDs)
	if err != nil {
		return nil, err
	}
	for _, web := range res {
		var (
			appName      string
			runtimeName  string
			runtimeType  string
			appInstallID uint
		)
		switch web.Type {
		case constant.Proxy:
			if web.AppInstallID > 0 {
				appInstall, err := appInstallRepo.GetFirst(commonRepo.WithByID(web.AppInstallID))
				if err == nil {
					appName = appInstall.Name
					appInstallID = appInstall.ID
				}
			}
		case constant.WebApp:
			runtimeName = web.EngineEnv
			runtimeType = "engine"
			appInstallID = 0
		}
		sitePath := path.Join(constant.AppInstallDir, "www", "sites", web.Alias)
		accessLogPath := websiteAccessLogPath(web.Alias)
		errorLogPath := websiteErrorLogPath(web.Alias)

		var otherDomains string
		if len(web.Domains) > 0 {
			var dList []string
			for _, d := range web.Domains {
				if d.Domain == "" || normalizeWebsiteDomainForCompare(d.Domain) == normalizeWebsiteDomainForCompare(web.PrimaryDomain) {
					continue
				}
				dList = append(dList, d.Domain)
			}
			otherDomains = strings.Join(dList, ",")
		}

		websiteDTOs = append(websiteDTOs, &response.WebsiteRes{
			ID:                       web.ID,
			CreatedAt:                web.CreatedAt,
			UpdatedAt:                web.UpdatedAt,
			Protocol:                 web.Protocol,
			PrimaryDomain:            web.PrimaryDomain,
			Type:                     web.Type,
			Remark:                   web.Remark,
			Status:                   web.Status,
			CodeSource:               web.CodeSource,
			Alias:                    web.Alias,
			AppName:                  appName,
			ExpireDate:               web.ExpireDate,
			RuntimeName:              runtimeName,
			RuntimeDir:               web.RuntimeDir,
			SitePath:                 sitePath,
			AccessLogPath:            accessLogPath,
			ErrorLogPath:             errorLogPath,
			AppInstallID:             appInstallID,
			ContainerID:              web.ContainerID,
			RuntimeType:              runtimeType,
			OtherDomains:             otherDomains,
			DefaultServer:            web.DefaultServer,
			Proxy:                    web.Proxy,
			IPV6:                     web.IPV6,
			Ipv6:                     web.IPV6,
			AntiCrawler:              web.AntiCrawler,
			AntiLeech:                web.AntiLeech,
			RateLimitMode:            web.RateLimitMode,
			WafEnable:                web.WafEnable,
			BlockSensitive:           web.BlockSensitive,
			IPAllowlist:              web.IPAllowlist,
			IPBlocklist:              web.IPBlocklist,
			SecurityHeader:           web.SecurityHeader,
			HstsEnabled:              web.HstsEnabled,
			HttpConfig:               web.HttpConfig,
			RedirectCode:             web.RedirectCode,
			RedirectDomainsToPrimary: web.RedirectDomainsToPrimary,
			Diagnostic:               diagnosticSummaries[web.ID],
			Upstreams:                responseWebsiteUpstreams(web.Upstreams, web.Proxy),
		})
	}
	FillWebsiteRuntimeMeta(context.Background(), websiteDTOs)
	return websiteDTOs, nil
}

func (s *WebsiteService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}
