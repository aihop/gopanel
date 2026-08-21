package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeAuditMeta map[string]any

func recordCodeAudit(userID, projectID, sessionID uint, action, status, resource, detail, ip string, startedAt time.Time, meta codeAuditMeta) {
	if global.DB == nil {
		return
	}
	encoded, _ := json.Marshal(meta)
	event := &model.AICodeAuditEvent{
		UserID: userID, ProjectID: projectID, SessionID: sessionID,
		Action: action, Status: status, Resource: strings.TrimSpace(resource),
		Detail: truncateCodeAuditDetail(detail), IP: ip,
		DurationMS: time.Since(startedAt).Milliseconds(), Meta: string(encoded),
	}
	if err := global.DB.Create(event).Error; err != nil && global.LOG != nil {
		global.LOG.Errorf("create Code audit event failed: %v", err)
	}
}

func truncateCodeAuditDetail(detail string) string {
	runes := []rune(strings.TrimSpace(detail))
	if len(runes) > 500 {
		runes = runes[:500]
	}
	return string(runes)
}

func GetCodeAuditEvents(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrCodeSessionIDInvalid)))
	}
	if _, err := getAISessionWithPermission(uint(sessionID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	page, limit = normalizeCodePage(page, limit, 50)
	query := global.DB.Model(&model.AICodeAuditEvent{}).Where("session_id = ?", sessionID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	var events []model.AICodeAuditEvent
	if err := query.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&events).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": events, "total": total}))
}
