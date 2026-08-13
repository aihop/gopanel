package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/container"
)

const flowDeploymentSourceType = "flow_container_bind"

type flowDeploymentLoggerKey struct{}

var (
	flowHealthCheck               = checkContainerWebsiteEndpoint
	flowHealthPollInterval        = time.Second
	flowStabilizationPollInterval = 5 * time.Second
	flowStabilizationDuration     = func(minutes int) time.Duration { return time.Duration(minutes) * time.Minute }
	flowRetentionDuration         = func(minutes int) time.Duration { return time.Duration(minutes) * time.Minute }
	flowCleanupContainer          = cleanupPreviousContainer
	flowPreparePreviousContainer  = preparePreviousFlowContainer
	flowPrepareTargetContainer    = prepareFlowTargetContainer
)

type flowWebsiteDeploymentSnapshot struct {
	website        model.Website
	upstreams      []model.WebsiteUpstream
	activeDeployID uint
}

func withFlowDeploymentLogger(ctx context.Context, logger *PipelineLogger) context.Context {
	return context.WithValue(ctx, flowDeploymentLoggerKey{}, logger)
}

func flowDeploymentLogger(ctx context.Context) *PipelineLogger {
	logger, _ := ctx.Value(flowDeploymentLoggerKey{}).(*PipelineLogger)
	return logger
}

func waitForFlowWebsiteReady(ctx context.Context, environment model.FlowEnvironment, target containerWebsiteTarget) error {
	timeoutSeconds := environment.ExternalVerifyTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	successTarget := environment.HealthCheckSuccessCount
	if successTarget <= 0 {
		successTarget = 1
	}
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	consecutiveSuccesses := 0
	var lastErr error
	for {
		requestCtx, requestCancel := context.WithTimeout(checkCtx, 5*time.Second)
		lastErr = flowHealthCheck(requestCtx, target.Scheme, target.Address)
		requestCancel()
		if lastErr == nil {
			consecutiveSuccesses++
			if consecutiveSuccesses >= successTarget {
				return nil
			}
		} else {
			consecutiveSuccesses = 0
		}
		if err := waitFlowHealthInterval(checkCtx, flowHealthPollInterval); err != nil {
			if lastErr != nil {
				return fmt.Errorf("端口健康检查未在 %d 秒内通过: %w", timeoutSeconds, lastErr)
			}
			return fmt.Errorf("端口健康检查未在 %d 秒内连续成功 %d 次", timeoutSeconds, successTarget)
		}
	}
}

func monitorFlowWebsiteStabilization(ctx context.Context, logger *PipelineLogger, environment model.FlowEnvironment, target containerWebsiteTarget) error {
	if !environment.RuntimeMonitorEnabled || environment.StabilizationMinutes <= 0 {
		return nil
	}
	failureThreshold := environment.RuntimeFailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	recoveryThreshold := environment.RuntimeRecoveryThreshold
	if recoveryThreshold <= 0 {
		recoveryThreshold = 1
	}
	monitorCtx, cancel := context.WithTimeout(ctx, flowStabilizationDuration(environment.StabilizationMinutes))
	defer cancel()
	if logger != nil {
		logger.Info("网站已切换，开始稳定观察: %d 分钟，连续失败阈值=%d", environment.StabilizationMinutes, failureThreshold)
	}

	failures := 0
	recoveries := 0
	for {
		if monitorCtx.Err() != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if logger != nil {
				logger.Info("网站稳定观察通过")
			}
			return nil
		}
		requestCtx, requestCancel := context.WithTimeout(monitorCtx, 5*time.Second)
		checkErr := flowHealthCheck(requestCtx, target.Scheme, target.Address)
		requestCancel()
		if monitorCtx.Err() != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if logger != nil {
				logger.Info("网站稳定观察通过")
			}
			return nil
		}
		if checkErr != nil {
			failures++
			recoveries = 0
			if failures >= failureThreshold {
				return fmt.Errorf("稳定观察期端口健康检查连续失败 %d 次: %w", failures, checkErr)
			}
			if logger != nil {
				logger.Info("稳定观察期健康检查失败 (%d/%d): %v", failures, failureThreshold, checkErr)
			}
		} else if failures > 0 {
			recoveries++
			if recoveries >= recoveryThreshold {
				failures = 0
				recoveries = 0
			}
		}
		if err := waitFlowHealthInterval(monitorCtx, flowStabilizationPollInterval); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if logger != nil {
				logger.Info("网站稳定观察通过")
			}
			return nil
		}
	}
}

func waitFlowHealthInterval(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Millisecond
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func captureFlowWebsiteDeploymentSnapshot(websiteID uint) (flowWebsiteDeploymentSnapshot, error) {
	var snapshot flowWebsiteDeploymentSnapshot
	if err := global.DB.First(&snapshot.website, websiteID).Error; err != nil {
		return snapshot, err
	}
	if err := global.DB.Where("website_id = ?", websiteID).Order("sort asc, id asc").Find(&snapshot.upstreams).Error; err != nil {
		return snapshot, err
	}
	var active model.AppDeploy
	if err := global.DB.Where("website_id = ? AND is_active = ?", websiteID, true).Order("id desc").First(&active).Error; err == nil {
		snapshot.activeDeployID = active.ID
	}
	return snapshot, nil
}

func restoreFlowWebsiteDeployment(ctx context.Context, snapshot flowWebsiteDeploymentSnapshot) error {
	tx := global.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()
	txCtx := context.WithValue(ctx, constant.DB, tx)
	if err := tx.Save(&snapshot.website).Error; err != nil {
		return err
	}
	if err := repo.NewWebsiteUpstream().ReplaceByWebsiteID(txCtx, snapshot.website.ID, snapshot.upstreams); err != nil {
		return err
	}
	if err := tx.Model(&model.AppDeploy{}).Where("website_id = ? AND is_active = ?", snapshot.website.ID, true).Update("is_active", false).Error; err != nil {
		return err
	}
	if snapshot.activeDeployID > 0 {
		if err := tx.Model(&model.AppDeploy{}).Where("id = ?", snapshot.activeDeployID).Update("is_active", true).Error; err != nil {
			return err
		}
	}
	if err := applyContainerWebsiteCaddy(txCtx); err != nil {
		return fmt.Errorf("恢复旧网站反向代理失败: %w", err)
	}
	return tx.Commit().Error
}

func preparePreviousFlowContainer(ctx context.Context, oldContainerID, newContainerID string) (bool, error) {
	oldContainerID = strings.TrimSpace(oldContainerID)
	if oldContainerID == "" {
		return false, nil
	}
	cli, err := docker.NewDockerClient()
	if err != nil {
		return false, err
	}
	if cli == nil {
		return false, errors.New("当前容器运行时不可用")
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(ctx, oldContainerID)
	if err != nil {
		return false, fmt.Errorf("读取旧容器状态失败: %w", err)
	}
	if inspect.State != nil && inspect.State.Running {
		return false, nil
	}
	if newContainerID = strings.TrimSpace(newContainerID); newContainerID != "" {
		if err := cli.ContainerStop(ctx, newContainerID, container.StopOptions{}); err != nil {
			return false, fmt.Errorf("释放新容器端口失败: %w", err)
		}
	}
	if err := cli.ContainerStart(ctx, oldContainerID, container.StartOptions{}); err != nil {
		return true, fmt.Errorf("重新启动旧容器失败: %w", err)
	}
	return true, nil
}

func prepareFlowTargetContainer(ctx context.Context, oldContainerID, newContainerID string) error {
	newContainerID = strings.TrimSpace(newContainerID)
	if newContainerID == "" {
		return errors.New("Runner 容器为空")
	}
	cli, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	if cli == nil {
		return errors.New("当前容器运行时不可用")
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(ctx, newContainerID)
	if err != nil {
		return fmt.Errorf("读取 Runner 容器状态失败: %w", err)
	}
	if inspect.State != nil && inspect.State.Running {
		return nil
	}
	if err := cli.ContainerStart(ctx, newContainerID, container.StartOptions{}); err == nil {
		return nil
	} else if oldContainerID = strings.TrimSpace(oldContainerID); oldContainerID == "" || !strings.Contains(err.Error(), "port is already allocated") {
		return fmt.Errorf("重新启动 Runner 容器失败: %w", err)
	}
	if err := cli.ContainerStop(ctx, oldContainerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("停止旧容器释放固定端口失败: %w", err)
	}
	if err := cli.ContainerStart(ctx, newContainerID, container.StartOptions{}); err != nil {
		_ = cli.ContainerStart(context.Background(), oldContainerID, container.StartOptions{})
		return fmt.Errorf("切换固定端口到 Runner 容器失败: %w", err)
	}
	return nil
}

func retainPreviousFlowContainer(websiteID uint, oldContainerID, newContainerID string, retention time.Duration) {
	oldContainerID = strings.TrimSpace(oldContainerID)
	newContainerID = strings.TrimSpace(newContainerID)
	if oldContainerID == "" || oldContainerID == newContainerID {
		return
	}
	cleanup := func() {
		var website model.Website
		if err := global.DB.Select("container_id").First(&website, websiteID).Error; err != nil {
			global.LOG.Errorf("确认旧容器 %s 是否仍在使用失败: %v", oldContainerID, err)
			return
		}
		if strings.TrimSpace(website.ContainerID) == oldContainerID {
			return
		}
		if err := flowCleanupContainer(oldContainerID); err != nil {
			global.LOG.Errorf("清理 Flow 旧容器 %s 失败: %v", oldContainerID, err)
			return
		}
		global.LOG.Infof("Flow 稳定期结束后已清理旧容器 %s", oldContainerID)
	}
	if retention <= 0 {
		cleanup()
		return
	}
	time.AfterFunc(retention, cleanup)
}
