package service

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
)

func buildWebsiteDeploySummary(item model.AppDeploy) *response.WebsiteDeploySummary {
	if item.ID == 0 {
		return nil
	}
	return &response.WebsiteDeploySummary{
		ID:               item.ID,
		Version:          item.Version,
		ReleaseID:        item.ReleaseID,
		PipelineRecordID: item.PipelineRecordID,
		SourceType:       item.SourceType,
		ImageTag:         item.ImageTag,
		Status:           item.Status,
		IsActive:         item.IsActive,
		CreatedAt:        item.CreatedAt,
	}
}

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
		if web != nil && web.ID > 0 {
			websiteIDs = append(websiteIDs, web.ID)
		}
	}
	appDeployRepo := repo.NewAppDeploy(global.DB)
	activeReleaseMap, _ := appDeployRepo.ActiveReleaseByWebsiteIDs(websiteIDs)
	latestPipelineSyncMap, _ := appDeployRepo.LatestPipelineSyncByWebsiteIDs(websiteIDs)
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
				dList = append(dList, d.Domain)
			}
			otherDomains = strings.Join(dList, ",")
		}

		websiteDTOs = append(websiteDTOs, &response.WebsiteRes{
			ID:                 web.ID,
			CreatedAt:          web.CreatedAt,
			UpdatedAt:          web.UpdatedAt,
			Protocol:           web.Protocol,
			PrimaryDomain:      web.PrimaryDomain,
			Type:               web.Type,
			Remark:             web.Remark,
			Status:             web.Status,
			CodeSource:         web.CodeSource,
			Alias:              web.Alias,
			AppName:            appName,
			ExpireDate:         web.ExpireDate,
			RuntimeName:        runtimeName,
			RuntimeDir:         web.RuntimeDir,
			SitePath:           sitePath,
			AccessLogPath:      accessLogPath,
			ErrorLogPath:       errorLogPath,
			AppInstallID:       appInstallID,
			PipelineID:         web.PipelineID,
			RuntimeType:        runtimeType,
			OtherDomains:       otherDomains,
			DefaultServer:      web.DefaultServer,
			Proxy:              web.Proxy,
			IPV6:               web.IPV6,
			Ipv6:               web.IPV6,
			AntiCrawler:        web.AntiCrawler,
			AntiLeech:          web.AntiLeech,
			RateLimitMode:      web.RateLimitMode,
			WafEnable:          web.WafEnable,
			BlockSensitive:     web.BlockSensitive,
			IPAllowlist:        web.IPAllowlist,
			IPBlocklist:        web.IPBlocklist,
			SecurityHeader:     web.SecurityHeader,
			HstsEnabled:        web.HstsEnabled,
			ActiveRelease:      buildWebsiteDeploySummary(activeReleaseMap[web.ID]),
			LatestPipelineSync: buildWebsiteDeploySummary(latestPipelineSyncMap[web.ID]),
		})
	}
	FillWebsiteRuntimeMeta(context.Background(), websiteDTOs)
	return websiteDTOs, nil
}

func (s *WebsiteService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}

type PipelineDeploySummary struct {
	Matched int
	Success int
	Failed  int
}

func (s *WebsiteService) DeployFromPipeline(ctx context.Context, pipelineID uint, pipelineRecordID uint, version string, artifactPath string, imageTag string) (*PipelineDeploySummary, error) {
	websites, err := s.repo.ListBy(s.repo.WithPipelineID(pipelineID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch websites for pipeline: %w", err)
	}

	summary := &PipelineDeploySummary{Matched: len(websites)}
	if len(websites) == 0 {
		global.LOG.Infof("No websites associated with pipeline %d. Skipping deployment.", pipelineID)
		return summary, nil
	}

	var failed []string
	for _, w := range websites {
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}
		global.LOG.Infof("Triggering pipeline result sync for website %s (ID: %d) from pipeline %d", w.Alias, w.ID, pipelineID)
		releaseDir := filepath.Join(global.CONF.System.BaseDir, "www", w.Alias, "releases", version)
		if _, err := ProcessAppDeployment(w, pipelineRecordID, version, artifactPath, releaseDir, "", imageTag, "pipeline_sync"); err != nil {
			summary.Failed++
			failed = append(failed, fmt.Sprintf("%s: %v", w.Alias, err))
			continue
		}
		summary.Success++
	}

	if len(failed) > 0 {
		return summary, fmt.Errorf("网站构建结果同步失败: %s", strings.Join(failed, " | "))
	}
	return summary, nil
}
