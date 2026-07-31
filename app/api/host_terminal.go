package api

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func CreateHostTerminalSession(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req createHostTerminalRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	record, err := hostTerminals.create(req, claims.UserId, c.IP())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(record))
}

func GetHostTerminalCapabilities(c fiber.Ctx) error {
	return c.JSON(e.Succ(getHostTerminalCapabilities()))
}

func ListHostTerminalSessions(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, limit = normalizeCodePage(page, limit, 20)
	query := global.DB.Model(&model.HostTerminalSession{}).Where("user_id = ?", claims.UserId)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	var sessions []model.HostTerminalSession
	if err := query.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&sessions).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	for index := range sessions {
		if (sessions[index].Status == "running" || sessions[index].Status == "starting") && hostTerminals.get(sessions[index].ID) == nil {
			markHostTerminalInterrupted(&sessions[index])
			sessions[index].Status = "interrupted"
			sessions[index].ErrorMessage = "服务重启或终端进程已丢失"
		}
	}
	return c.JSON(e.Succ(fiber.Map{"items": sessions, "total": total}))
}

func StopHostTerminalSession(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	record, err := loadHostTerminalRecord(c.Params("id"), claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if record.Status != "running" && record.Status != "starting" {
		return c.JSON(e.Fail(errors.New("终端会话已结束")))
	}
	if !hostTerminals.stop(record.ID) {
		markHostTerminalInterrupted(record)
		return c.JSON(e.Fail(errors.New("终端进程已不在当前服务中")))
	}
	recordHostTerminalAudit(record.ID, claims.UserId, "stop", "success", c.IP(), "用户停止终端会话")
	return c.JSON(e.Succ(nil))
}

func GetHostTerminalAuditEvents(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	record, err := loadHostTerminalRecord(c.Params("id"), claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var events []model.HostTerminalAuditEvent
	if err := global.DB.Where("session_id = ?", record.ID).Order("created_at desc").Limit(100).Find(&events).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(events))
}

func HostTerminalWebSocket(wsConn *websocket.Conn) {
	defer wsConn.Close()
	claims, ok := wsConn.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil {
		return
	}
	record, err := loadHostTerminalRecord(wsConn.Params("id"), claims.UserId)
	if err != nil {
		_ = wsConn.WriteJSON(hostTerminalEvent{Type: "error", Data: err.Error()})
		return
	}
	session := hostTerminals.get(record.ID)
	if session == nil {
		markHostTerminalInterrupted(record)
		_ = wsConn.WriteJSON(hostTerminalEvent{Type: "error", Data: "终端进程已结束或服务已重启"})
		return
	}
	readOnly := wsConn.Query("read_only") == "1"
	subscriber, baseline := session.subscribe(claims.UserId, wsConn.IP(), readOnly)
	recordHostTerminalAudit(record.ID, claims.UserId, "connect", "success", wsConn.IP(), mapReadOnlyDetail(readOnly))
	defer session.unsubscribe(subscriber)
	var writeMu sync.Mutex
	writeEvent := func(event hostTerminalEvent) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return wsConn.WriteJSON(event)
	}
	if err := writeEvent(baseline); err != nil {
		return
	}
	go func() {
		for event := range subscriber.Events {
			if writeEvent(event) != nil {
				_ = wsConn.Close()
				return
			}
			if event.Type == "closed" {
				_ = wsConn.Close()
				return
			}
		}
	}()
	for {
		_, payload, err := wsConn.ReadMessage()
		if err != nil {
			return
		}
		var message struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if json.Unmarshal(payload, &message) != nil {
			continue
		}
		switch message.Type {
		case "cmd":
			if err := session.write(subscriber.ID, []byte(message.Data)); err != nil {
				_ = writeEvent(hostTerminalEvent{Type: "control", Data: err.Error()})
			}
		case "resize":
			var size struct{ Cols, Rows uint16 }
			if json.Unmarshal([]byte(message.Data), &size) == nil && size.Cols > 0 && size.Rows > 0 {
				_ = session.resize(subscriber.ID, size.Cols, size.Rows)
			}
		case "take_control":
			granted, reason := session.takeControl(subscriber.ID)
			status := "success"
			if !granted {
				status = "denied"
				_ = writeEvent(hostTerminalEvent{Type: "control", Data: reason})
			}
			recordHostTerminalAudit(record.ID, claims.UserId, "take_control", status, wsConn.IP(), reason)
		case "release_control":
			session.releaseControl(subscriber.ID)
		case "resync":
			session.mu.Lock()
			event := hostTerminalEvent{Type: "baseline", Sequence: session.sequence, Data: string(session.history), HasControl: session.controllerID == subscriber.ID, Truncated: session.historyTruncated, LeaseExpiresAt: session.controlExpiresAt.UnixMilli()}
			session.mu.Unlock()
			_ = writeEvent(event)
		case "ping":
			_ = writeEvent(hostTerminalEvent{Type: "pong"})
		}
	}
}

func loadHostTerminalRecord(idValue string, userID uint) (*model.HostTerminalSession, error) {
	id, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil || id == 0 {
		return nil, errors.New("终端会话 ID 无效")
	}
	var record model.HostTerminalSession
	if err := global.DB.Where("id = ? AND user_id = ?", id, userID).First(&record).Error; err != nil {
		return nil, errors.New("终端会话不存在")
	}
	return &record, nil
}

func markHostTerminalInterrupted(record *model.HostTerminalSession) {
	if record == nil || (record.Status != "running" && record.Status != "starting") {
		return
	}
	now := time.Now()
	_ = global.DB.Model(record).Updates(map[string]any{"status": "interrupted", "ended_at": now, "error_message": "服务重启或终端进程已丢失"}).Error
}

func recordHostTerminalAudit(sessionID, userID uint, action, status, ip, detail string) {
	if global.DB == nil {
		return
	}
	_ = global.DB.Create(&model.HostTerminalAuditEvent{SessionID: sessionID, UserID: userID, Action: action, Status: status, IP: ip, Detail: truncateHostTerminalDetail(detail)}).Error
}

func truncateHostTerminalDetail(detail string) string {
	runes := []rune(strings.TrimSpace(detail))
	if len(runes) > 500 {
		runes = runes[:500]
	}
	return string(runes)
}

func mapReadOnlyDetail(readOnly bool) string {
	if readOnly {
		return "只读连接"
	}
	return "可控连接"
}
