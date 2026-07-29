package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

const (
	CodeNotifyApproval  = "needsInput"
	CodeNotifyCompleted = "completed"
	CodeNotifyFailed    = "failed"
)

func NotifyCodeSession(session *model.AIDevSession, task *model.AITask, state, summary string) {
	if session == nil {
		return
	}
	cfg, err := repo.NewNotify().GetConfig()
	if err != nil || !cfg.Enabled || !cfg.EnableCode {
		return
	}
	subject, body := buildCodeNotifyMessage(session, task, state, summary)
	if err := SendNotifyMail(cfg, subject, body); err != nil {
		global.LOG.Errorf("[Code] 会话 %d 通知发送失败: %v", session.ID, err)
		return
	}
	global.LOG.Infof("[Code] 会话 %d 已发送 %s 通知", session.ID, state)
}

func buildCodeNotifyMessage(session *model.AIDevSession, task *model.AITask, state, summary string) (string, string) {
	stateLabel := codeNotifyStateLabel(state)
	title := strings.TrimSpace(session.Title)
	if task != nil && strings.TrimSpace(task.Title) != "" {
		title = strings.TrimSpace(task.Title)
	}
	if title == "" {
		title = fmt.Sprintf("会话 #%d", session.ID)
	}
	subject := fmt.Sprintf("[%s Code] %s：%s", constant.AppBrand, stateLabel, title)
	var body strings.Builder
	body.WriteString(fmt.Sprintf("%s Code 任务通知\n\n", constant.AppBrand))
	body.WriteString(fmt.Sprintf("状态：%s\n", stateLabel))
	body.WriteString(fmt.Sprintf("任务：%s\n", title))
	body.WriteString(fmt.Sprintf("执行器：%s\n", strings.TrimSpace(session.AgentName)))
	body.WriteString(fmt.Sprintf("工作目录：%s\n", strings.TrimSpace(session.WorkDir)))
	if summary = strings.TrimSpace(summary); summary != "" {
		body.WriteString(fmt.Sprintf("\n摘要：\n%s\n", summary))
	}
	if state == CodeNotifyApproval {
		body.WriteString("\n该任务正在等待人工确认，请打开 GoPanel Code 处理。\n")
	} else {
		body.WriteString("\n请打开 GoPanel Code 查看完整对话、日志和预览。\n")
	}
	body.WriteString(fmt.Sprintf("会话 ID：%d\n发送时间：%s\n", session.ID, time.Now().Format("2006-01-02 15:04:05")))
	return subject, body.String()
}

func codeNotifyStateLabel(state string) string {
	switch state {
	case CodeNotifyApproval:
		return "等待确认"
	case CodeNotifyFailed:
		return "执行失败"
	default:
		return "执行完成"
	}
}
