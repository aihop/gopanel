package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeAttentionAction struct {
	Type                 string `json:"type"`
	Label                string `json:"label"`
	Method               string `json:"method,omitempty"`
	Path                 string `json:"path,omitempty"`
	RequiresConfirmation bool   `json:"requiresConfirmation"`
}

type codeAttentionItem struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Severity   string                `json:"severity"`
	Title      string                `json:"title"`
	Summary    string                `json:"summary"`
	SessionID  uint                  `json:"sessionId"`
	TaskID     uint                  `json:"taskId,omitempty"`
	ApprovalID uint                  `json:"approvalId,omitempty"`
	UpdatedAt  time.Time             `json:"updatedAt"`
	Actions    []codeAttentionAction `json:"actions"`
}

func GetCodeAttention(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	_, limit = normalizeCodePage(1, limit, 100)
	items, err := loadCodeAttentionItems(claims.UserId, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if strings.Contains(c.Path(), "/mobile/app/") {
		useMobileCodeAttentionPaths(items)
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": len(items)}))
}

func loadCodeAttentionItems(userID uint, limit int) ([]codeAttentionItem, error) {
	if global.DB == nil {
		return nil, errors.New("数据库未初始化")
	}
	var sessions []model.AIDevSession
	if err := global.DB.Where("user_id = ?", userID).Order("updated_at desc").Limit(limit).Find(&sessions).Error; err != nil || len(sessions) == 0 {
		return []codeAttentionItem{}, err
	}
	sessionIDs := make([]uint, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
	}
	approvals, deliveries, runs, err := loadCodeAttentionSources(userID, sessionIDs)
	if err != nil {
		return nil, err
	}
	items := make([]codeAttentionItem, 0, len(sessions))
	for _, session := range sessions {
		if item := buildCodeAttentionItem(session, approvals[session.ID], deliveries[session.ID], runs[session.ID]); item != nil {
			items = append(items, *item)
		}
	}
	return items, nil
}

func loadCodeAttentionSources(userID uint, sessionIDs []uint) (map[uint]*model.AIApproval, map[uint]*model.AICodeDeliveryJob, map[uint]*model.AIExecutionRun, error) {
	approvals := make(map[uint]*model.AIApproval)
	deliveries := make(map[uint]*model.AICodeDeliveryJob)
	runs := make(map[uint]*model.AIExecutionRun)
	var approvalRows []model.AIApproval
	if err := global.DB.Where("request_user_id = ? AND status = ? AND session_id IN ?", userID, "pending", sessionIDs).Order("created_at desc").Find(&approvalRows).Error; err != nil {
		return nil, nil, nil, err
	}
	for index := range approvalRows {
		if approvals[approvalRows[index].SessionID] == nil {
			approvals[approvalRows[index].SessionID] = &approvalRows[index]
		}
	}
	var deliveryRows []model.AICodeDeliveryJob
	if err := global.DB.Where("user_id = ? AND session_id IN ?", userID, sessionIDs).Find(&deliveryRows).Error; err != nil {
		return nil, nil, nil, err
	}
	for index := range deliveryRows {
		deliveries[deliveryRows[index].SessionID] = &deliveryRows[index]
	}
	var runRows []model.AIExecutionRun
	rankedRuns := global.DB.Model(&model.AIExecutionRun{}).
		Select("ai_execution_runs.*, ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at DESC, id DESC) AS row_number").
		Where("session_id IN ?", sessionIDs)
	if err := global.DB.Table("(?) AS ranked_runs", rankedRuns).Where("row_number = 1").Find(&runRows).Error; err != nil {
		return nil, nil, nil, err
	}
	for index := range runRows {
		if runs[runRows[index].SessionID] == nil {
			runs[runRows[index].SessionID] = &runRows[index]
		}
	}
	return approvals, deliveries, runs, nil
}

func buildCodeAttentionItem(session model.AIDevSession, approval *model.AIApproval, delivery *model.AICodeDeliveryJob, run *model.AIExecutionRun) *codeAttentionItem {
	item := &codeAttentionItem{SessionID: session.ID, TaskID: session.LastTaskID, UpdatedAt: session.UpdatedAt}
	switch {
	case approval != nil:
		item.ID, item.Type, item.Severity = fmt.Sprintf("approval:%d", approval.ID), "approval", "warning"
		item.Title, item.Summary, item.ApprovalID = "等待你确认", approval.Content, approval.ID
		item.UpdatedAt = approval.UpdatedAt
		item.Actions = []codeAttentionAction{
			{Type: "approve", Label: "允许执行", Method: "POST", Path: fmt.Sprintf("/api/code/approvals/%d/approve", approval.ID), RequiresConfirmation: true},
			{Type: "reject", Label: "拒绝", Method: "POST", Path: fmt.Sprintf("/api/code/approvals/%d/reject", approval.ID), RequiresConfirmation: true},
		}
	case session.Status == codeSessionStatusFailed && session.CurrentStage == codeSessionStageInitializationFailed:
		item.ID, item.Type, item.Severity = fmt.Sprintf("initialization:%d", session.ID), "initialization_failed", "error"
		item.Title, item.Summary = "会话初始化失败", session.InitializationErr
		item.Actions = []codeAttentionAction{{Type: "retry_initialization", Label: "重新初始化", Method: "POST", Path: fmt.Sprintf("/api/code/sessions/%d/initialization/retry", session.ID), RequiresConfirmation: true}}
	case delivery != nil && (delivery.Status == codeDeliveryJobConflict || delivery.Status == codeDeliveryJobPartial || delivery.Status == codeDeliveryJobFailed):
		item.ID, item.Type, item.Severity = fmt.Sprintf("delivery:%d", delivery.ID), "delivery_failed", "error"
		item.Title, item.Summary = codeDeliveryAttentionText(delivery)
		item.UpdatedAt = delivery.UpdatedAt
		item.Actions = []codeAttentionAction{{Type: "open_session", Label: "查看并处理"}}
	case run != nil && run.Status == "failed" && (session.LastTaskID == 0 || run.TaskID == session.LastTaskID):
		item.ID, item.Type, item.Severity = fmt.Sprintf("run:%d", run.ID), "execution_failed", "error"
		item.Title, item.Summary = "开发任务执行失败", run.ErrorMessage
		item.TaskID, item.UpdatedAt = run.TaskID, run.UpdatedAt
		item.Actions = []codeAttentionAction{{Type: "open_session", Label: "查看详情"}}
	default:
		return nil
	}
	item.Summary = truncateCodeAttentionSummary(item.Summary)
	return item
}

func codeDeliveryAttentionText(job *model.AICodeDeliveryJob) (string, string) {
	if job.Status == codeDeliveryJobConflict {
		return "代码交付存在冲突", "请打开会话查看冲突文件并处理后继续交付。"
	}
	if job.Status == codeDeliveryJobPartial {
		return "部分仓库交付失败", job.ErrorMessage
	}
	return "代码交付失败", job.ErrorMessage
}

func truncateCodeAttentionSummary(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 300 {
		runes = runes[:300]
	}
	return string(runes)
}

func useMobileCodeAttentionPaths(items []codeAttentionItem) {
	for itemIndex := range items {
		for actionIndex := range items[itemIndex].Actions {
			items[itemIndex].Actions[actionIndex].Path = strings.Replace(
				items[itemIndex].Actions[actionIndex].Path, "/api/code/", "/api/mobile/app/", 1,
			)
		}
	}
}
