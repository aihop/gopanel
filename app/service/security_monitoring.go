package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

var securityMonitoringMutex sync.Mutex

func EvaluateSecurityRisks() {
	if !securityMonitoringMutex.TryLock() {
		return
	}
	defer securityMonitoringMutex.Unlock()
	repository := repo.NewSecurityMonitoring()
	config, err := repository.GetConfig()
	if err != nil || !config.Enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	findings := make([]securityFinding, 0)
	if config.WebsiteEnabled {
		websiteFindings, collectErr := collectWebsiteSecurityFindings(config)
		if collectErr != nil {
			global.LOG.Errorf("[Security] 网站日志采集失败: %v", collectErr)
		} else {
			findings = append(findings, websiteFindings...)
		}
	}
	if config.SSHEnabled {
		sshFindings, collectErr := collectSSHSecurityFindings(ctx, config)
		if collectErr != nil {
			global.LOG.Errorf("[Security] SSH 日志采集失败: %v", collectErr)
		} else {
			findings = append(findings, sshFindings...)
		}
	}
	if config.PanelEnabled {
		panelFindings, collectErr := collectPanelSecurityFindings(config)
		if collectErr != nil {
			global.LOG.Errorf("[Security] 面板登录日志采集失败: %v", collectErr)
		} else {
			findings = append(findings, panelFindings...)
		}
	}
	for _, finding := range findings {
		event, fired, saveErr := repository.UpsertEvent(finding.event(), config.DebounceTimes)
		if saveErr != nil {
			global.LOG.Errorf("[Security] 风险事件保存失败: %v", saveErr)
			continue
		}
		if fired {
			notifySecurityEvent(event, false)
		}
	}
	resolveMinutes := config.ResolveAfterMinutes
	if resolveMinutes < 1 {
		resolveMinutes = 10
	}
	resolved, err := repository.ResolveStale(time.Now().Add(-time.Duration(resolveMinutes) * time.Minute))
	if err != nil {
		global.LOG.Errorf("[Security] 风险恢复评估失败: %v", err)
		return
	}
	for index := range resolved {
		notifySecurityEvent(&resolved[index], true)
	}
}

func GetSecurityMonitoringConfig() (model.SecurityMonitoringConfig, error) {
	return repo.NewSecurityMonitoring().GetConfig()
}

func SaveSecurityMonitoringConfig(config *model.SecurityMonitoringConfig) error {
	if config == nil {
		return errors.New("安全监测配置不能为空")
	}
	if config.AIIntervalMinutes < 5 || config.AIIntervalMinutes > 1440 {
		return errors.New("AI 巡检周期必须在 5 到 1440 分钟之间")
	}
	if config.MaxBatchBytes < 4096 || config.MaxBatchBytes > 16<<20 {
		return errors.New("单批日志大小必须在 4KB 到 16MB 之间")
	}
	if config.MaxBatchLines < 100 || config.MaxBatchLines > 50000 {
		return errors.New("单批日志行数必须在 100 到 50000 之间")
	}
	if config.DebounceTimes < 1 || config.DebounceTimes > 10 {
		return errors.New("风险去抖次数必须在 1 到 10 之间")
	}
	if config.ResolveAfterMinutes < 1 || config.ResolveAfterMinutes > 1440 {
		return errors.New("恢复判定时间必须在 1 到 1440 分钟之间")
	}
	return repo.NewSecurityMonitoring().SaveConfig(config)
}
