package api

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func GetSystemDiagnosticState(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("只有管理员可以使用系统诊断中心")))
	}
	var accounts []model.AIProviderAccount
	if err := global.DB.Where("user_id = ? AND enabled = ?", claims.UserId, true).
		Order("priority asc, name asc, id asc").Find(&accounts).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	var session model.AIDevSession
	err := global.DB.Where("user_id = ? AND agent_name = ?", claims.UserId, "system_diagnostic").Order("updated_at desc").First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	messages := make([]model.AIMessage, 0)
	if session.ID > 0 {
		_ = global.DB.Where("session_id = ?", session.ID).Order("created_at desc").Limit(100).Find(&messages).Error
		for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
			messages[left], messages[right] = messages[right], messages[left]
		}
	}
	return c.JSON(e.Succ(fiber.Map{
		"session": session, "messages": messages, "accounts": aiProviderAccountViews(accounts),
		"snapshot": buildSystemDiagnosticSnapshot(),
	}))
}

func ChatSystemDiagnostic(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("只有管理员可以使用系统诊断中心")))
	}
	var request struct {
		Content   string `json:"content"`
		AccountID uint   `json:"accountId"`
	}
	if err := c.Bind().JSON(&request); err != nil {
		return c.JSON(e.Fail(err))
	}
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" || len([]rune(request.Content)) > 4000 || request.AccountID == 0 {
		return c.JSON(e.Fail(errors.New("请选择 AI 并输入不超过 4000 字的诊断问题")))
	}
	request.Content = sanitizeSystemDiagnosticText(request.Content)
	var account model.AIProviderAccount
	if err := global.DB.Where("id = ? AND user_id = ? AND enabled = ?", request.AccountID, claims.UserId, true).First(&account).Error; err != nil {
		return c.JSON(e.Fail(errors.New("所选 AI 账号不存在或已停用")))
	}
	apiKey, err := encrypt.StringDecrypt(account.APIKey)
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return c.JSON(e.Fail(errors.New("所选 AI 账号密钥不可用")))
	}
	session, task, err := ensureSystemDiagnosticSession(claims.UserId, account)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var history []model.AIMessage
	if err := global.DB.Where("session_id = ?", session.ID).Order("created_at desc").Limit(20).Find(&history).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	for left, right := 0, len(history)-1; left < right; left, right = left+1, right-1 {
		history[left], history[right] = history[right], history[left]
	}
	userMessage := model.AIMessage{SessionID: session.ID, TaskID: task.ID, Role: "user", Content: request.Content}
	if err := global.DB.Create(&userMessage).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	llmMessages := []systemDiagnosticLLMMessage{{Role: "system", Content: systemDiagnosticPrompt}}
	for _, message := range history {
		role := "assistant"
		if message.Role == "user" {
			role = "user"
		}
		llmMessages = append(llmMessages, systemDiagnosticLLMMessage{Role: role, Content: message.Content})
	}
	llmMessages = append(llmMessages, systemDiagnosticLLMMessage{Role: "user", Content: request.Content})
	startedAt := time.Now()
	run := model.AIExecutionRun{SessionID: session.ID, TaskID: task.ID, ExecutorID: "system_diagnostic", Model: account.Model, Prompt: request.Content, Status: "running", StartedAt: startedAt}
	if err := global.DB.Create(&run).Error; err != nil {
		return c.JSON(e.Fail(err))
	}
	userID, clientIP := claims.UserId, c.IP()
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	c.Status(fiber.StatusOK)
	c.RequestCtx().SetBodyStreamWriter(func(writer *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := writeSystemDiagnosticSSE(writer, "start", fiber.Map{"session": session, "userMessage": userMessage, "run": run}); err != nil {
			return
		}
		answer, rawOutput, usage, toolAudits, runErr := runSystemDiagnosticLLM(ctx, &account, apiKey, llmMessages, func(delta string) error {
			if err := writeSystemDiagnosticSSE(writer, "delta", fiber.Map{"content": delta}); err != nil {
				cancel()
				return err
			}
			return nil
		})
		now := time.Now()
		run.DurationMS, run.CompletedAt = time.Since(startedAt).Milliseconds(), &now
		run.InputTokens, run.OutputTokens, run.TotalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
		run.RawOutput = rawOutput
		if runErr != nil {
			finishSystemDiagnosticRunFailure(&run, runErr, userID, session.ID, clientIP, startedAt, account.Model, toolAudits)
			_ = writeSystemDiagnosticSSE(writer, "error", fiber.Map{"message": runErr.Error()})
			return
		}
		assistantMessage, saveErr := finishSystemDiagnosticRunSuccess(&run, session, task, answer, account.Model, now)
		if saveErr != nil {
			finishSystemDiagnosticRunFailure(&run, saveErr, userID, session.ID, clientIP, startedAt, account.Model, toolAudits)
			_ = writeSystemDiagnosticSSE(writer, "error", fiber.Map{"message": saveErr.Error()})
			return
		}
		recordCodeAudit(userID, 0, session.ID, "system_diagnostic", "success", "gopanel", "只读诊断完成", clientIP, startedAt, codeAuditMeta{"model": account.Model, "runId": run.ID, "tools": toolAudits})
		_ = writeSystemDiagnosticSSE(writer, "done", fiber.Map{"session": session, "userMessage": userMessage, "assistantMessage": assistantMessage, "run": run})
	})
	return nil
}

func finishSystemDiagnosticRunFailure(run *model.AIExecutionRun, runErr error, userID, sessionID uint, clientIP string, startedAt time.Time, modelName string, toolAudits []systemDiagnosticToolAudit) {
	run.Status, run.ErrorMessage = "failed", runErr.Error()
	_ = global.DB.Save(run).Error
	recordCodeAudit(userID, 0, sessionID, "system_diagnostic", "failed", "gopanel", runErr.Error(), clientIP, startedAt, codeAuditMeta{"model": modelName, "runId": run.ID, "tools": toolAudits})
}

func finishSystemDiagnosticRunSuccess(run *model.AIExecutionRun, session *model.AIDevSession, task *model.AITask, answer, modelName string, completedAt time.Time) (*model.AIMessage, error) {
	run.Status, run.Output = "completed", answer
	assistantMessage := &model.AIMessage{SessionID: session.ID, TaskID: task.ID, RunID: run.ID, Role: "agent", Content: answer}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(run).Error; err != nil {
			return err
		}
		if err := tx.Create(assistantMessage).Error; err != nil {
			return err
		}
		return tx.Model(&model.AIDevSession{}).Where("id = ?", session.ID).Updates(map[string]any{"updated_at": completedAt, "provider_model": modelName}).Error
	})
	return assistantMessage, err
}

func ensureSystemDiagnosticSession(userID uint, account model.AIProviderAccount) (*model.AIDevSession, *model.AITask, error) {
	var session model.AIDevSession
	if err := global.DB.Where("user_id = ? AND agent_name = ?", userID, "system_diagnostic").Order("updated_at desc").First(&session).Error; err == nil {
		var task model.AITask
		if taskErr := global.DB.Where("id = ? AND user_id = ?", session.LastTaskID, userID).First(&task).Error; taskErr == nil {
			return &session, &task, nil
		}
	}
	session = model.AIDevSession{UserID: userID, Title: "GoPanel 系统诊断", AgentName: "system_diagnostic", WorkDir: ".", Status: "active", CurrentStage: "idle", ApprovalPolicy: "manual", ProviderModel: account.Model}
	task := model.AITask{UserID: userID, Title: session.Title, AgentName: session.AgentName, WorkDir: session.WorkDir, Status: "active"}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		task.SessionID = session.ID
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		session.LastTaskID = task.ID
		return tx.Save(&session).Error
	})
	return &session, &task, err
}
