package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func notifySecurityEvent(event *model.SecurityEvent, resolved bool) {
	if event == nil {
		return
	}
	cfg, err := repo.NewNotify().GetConfig()
	if err != nil || !cfg.Enabled || !cfg.EnableSecurity {
		return
	}
	if !resolved && securityLevelRank(event.Level) < securityLevelRank("high") && !cfg.EnableSecurityLowMedium {
		return
	}
	subject, body := buildSecurityEventMail(event, resolved)
	now := time.Now()
	if err := SendNotifyMail(cfg, subject, body); err != nil {
		event.NotifyStatus, event.NotifyError = "failed", err.Error()
		global.LOG.Errorf("[Security] 风险事件 %d 邮件发送失败: %v", event.ID, err)
	} else {
		event.NotifyStatus, event.NotifyError, event.LastNotifiedAt = "sent", "", &now
	}
	if saveErr := repo.NewSecurityMonitoring().SaveEvent(event); saveErr != nil {
		global.LOG.Errorf("[Security] 风险事件 %d 通知状态保存失败: %v", event.ID, saveErr)
	}
}

func notifySecurityAIUpdate(event *model.SecurityEvent, previousLevel, previousConclusion string) {
	if event == nil || event.AnalysisStatus != model.SecurityAnalysisCompleted {
		return
	}
	if securityLevelRank(event.Level) <= securityLevelRank(previousLevel) && strings.TrimSpace(event.AIConclusion) == strings.TrimSpace(previousConclusion) {
		return
	}
	cfg, err := repo.NewNotify().GetConfig()
	if err != nil || !cfg.Enabled || !cfg.EnableSecurity {
		return
	}
	if securityLevelRank(event.Level) < securityLevelRank("high") && !cfg.EnableSecurityLowMedium {
		return
	}
	subject := fmt.Sprintf("[%s 安全] AI 研判更新：%s", constant.AppBrand, event.SourceName)
	_, body := buildSecurityEventMail(event, false)
	body = "AI 已完成风险研判，结论如下：\n\n" + body
	now := time.Now()
	if err := SendNotifyMail(cfg, subject, body); err != nil {
		event.NotifyError = err.Error()
		global.LOG.Errorf("[Security] 风险事件 %d AI 补充邮件失败: %v", event.ID, err)
	} else {
		event.LastAINotifiedAt, event.NotifyError = &now, ""
	}
	_ = repo.NewSecurityMonitoring().SaveEvent(event)
}

func RetrySecurityNotifications() {
	cfg, err := repo.NewNotify().GetConfig()
	if err != nil || !cfg.Enabled || !cfg.EnableSecurity {
		return
	}
	silenceHours := cfg.SilenceHours
	if silenceHours <= 0 {
		return
	}
	events, err := repo.NewSecurityMonitoring().EventsNeedingNotification(time.Now().Add(-time.Duration(silenceHours)*time.Hour), 20)
	if err != nil {
		global.LOG.Errorf("[Security] 加载待提醒风险失败: %v", err)
		return
	}
	for index := range events {
		notifySecurityEvent(&events[index], false)
	}
}

func buildSecurityEventMail(event *model.SecurityEvent, resolved bool) (string, string) {
	state := "风险告警"
	if resolved {
		state = "风险已恢复"
	}
	subject := fmt.Sprintf("[%s 安全] %s：%s", constant.AppBrand, state, event.SourceName)
	var body strings.Builder
	body.WriteString(fmt.Sprintf("%s 安全监测通知\n\n", constant.AppBrand))
	body.WriteString(fmt.Sprintf("状态：%s\n风险等级：%s\n对象：%s\n类型：%s\n", state, event.Level, event.SourceName, event.EventType))
	body.WriteString(fmt.Sprintf("首次发现：%s\n最近发现：%s\n命中次数：%d\n", event.FirstSeenAt.Format("2006-01-02 15:04:05"), event.LastSeenAt.Format("2006-01-02 15:04:05"), event.HitCount))
	body.WriteString("\n摘要：\n" + strings.TrimSpace(event.Summary) + "\n")
	if event.AnalysisStatus == model.SecurityAnalysisCompleted && strings.TrimSpace(event.AIConclusion) != "" {
		body.WriteString(fmt.Sprintf("\nAI 研判（置信度 %d%%）：\n%s\n", event.Confidence, strings.TrimSpace(event.AIConclusion)))
		var actions []map[string]any
		if json.Unmarshal([]byte(event.SuggestedActions), &actions) == nil && len(actions) > 0 {
			body.WriteString("\n建议动作：\n")
			for _, action := range actions {
				body.WriteString(fmt.Sprintf("- %v（需要审批：%v）\n", action["action"], action["requiresApproval"]))
			}
		}
	}
	body.WriteString("\n请打开 GoPanel 安全风险中心查看脱敏证据与详情。\n")
	return subject, body.String()
}
