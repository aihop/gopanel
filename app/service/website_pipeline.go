package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/global"
)

type PipelineDeploySummary struct {
	Matched int
	Success int
	Failed  int
}

// DeployFromPipeline 流水线部署
func (s *WebsiteService) DeployFromPipeline(ctx context.Context, pipelineID uint, pipelineRecordID uint, version string, artifactPath string, imageTag string, exposePort int) (*PipelineDeploySummary, error) {
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
		if _, err := ProcessAppDeployment(w, pipelineRecordID, version, artifactPath, releaseDir, "", imageTag, "pipeline_sync", exposePort); err != nil {
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
