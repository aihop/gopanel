package api

import (
	"errors"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getAISessionWithPermission(sessionID uint, claims *token.CustomClaims) (*model.AIDevSession, error) {
	sessionRepo := repo.NewAIDevSessionRepo()
	session, err := sessionRepo.GetSessionByID(sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return nil, errors.New("无权访问该开发会话")
	}
	return session, nil
}
func buildDefaultSessionTitle(workDir, content string) string {
	workDir = strings.TrimSpace(workDir)
	if workDir != "" && workDir != "/" && workDir != "." {
		base := filepath.Base(workDir)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "新开发会话"
	}
	runes := []rune(content)
	if len(runes) > 24 {
		runes = runes[:24]
	}
	return strings.TrimSpace(string(runes))
}
func ensureSessionTask(session *model.AIDevSession, claims *token.CustomClaims, hint string) (*model.AITask, error) {
	aiRepo := repo.NewAITaskRepo()
	if session.LastTaskID > 0 {
		task, err := aiRepo.GetTaskByID(session.LastTaskID)
		if err == nil {
			return task, nil
		}
	}
	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = buildDefaultSessionTitle(session.WorkDir, hint)
	}
	task := &model.AITask{UserID: claims.UserId, ProjectID: session.ProjectID, Title: title, AgentName: session.AgentName, WorkDir: session.WorkDir, Status: "queued"}
	if err := aiRepo.CreateTask(task); err != nil {
		return nil, err
	}
	session.LastTaskID = task.ID
	session.Status = "active"
	session.CurrentStage = "task_ready"
	if err := repo.NewAIDevSessionRepo().UpdateSession(session); err != nil {
		return nil, err
	}
	return task, nil
}
func GetAISessions(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	projectID, _ := strconv.Atoi(c.Query("projectId", "0"))
	sessionRepo := repo.NewAIDevSessionRepo()
	sessions, total, err := sessionRepo.GetSessionsByUserID(claims.UserId, uint(projectID), page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": sessions, "total": total}))
}
func GetAISession(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	sessionRepo := repo.NewAIDevSessionRepo()
	latestInstruction, err := sessionRepo.GetLatestInstructionBySessionID(session.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		latestInstruction = nil
		err = nil
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews, err := sessionRepo.GetPreviewsBySessionID(session.ID, 20)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews = refreshPreviewStatuses(sessionRepo, previews)
	pendingApproval := getPendingApprovalForSession(sessionRepo, session.ID)
	var currentTask *model.AITask
	if session.LastTaskID > 0 {
		currentTask, _ = repo.NewAITaskRepo().GetTaskByID(session.LastTaskID)
	}
	return c.JSON(e.Succ(fiber.Map{"session": session, "currentTask": currentTask, "latestInstruction": latestInstruction, "previews": previews, "pendingApproval": pendingApproval}))
}
func CreateAISession(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Title     string `json:"title"`
		WorkDir   string `json:"workDir"`
		ProjectID uint   `json:"projectId"`
		AgentName string `json:"agentName"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	workDir := strings.TrimSpace(req.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildDefaultSessionTitle(workDir, "")
	}
	session := &model.AIDevSession{UserID: claims.UserId, ProjectID: req.ProjectID, Title: title, AgentName: strings.TrimSpace(req.AgentName), WorkDir: workDir, Status: "active", CurrentStage: "idle"}
	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.CreateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(session))
}
func CreateAISessionInstruction(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var req struct {
		Content         string `json:"content"`
		AllowCode       *bool  `json:"allowCode"`
		AutoPreview     bool   `json:"autoPreview"`
		RequireApproval *bool  `json:"requireApproval"`
		AnalysisOnly    bool   `json:"analysisOnly"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return c.JSON(e.Fail(errors.New("开发指令不能为空")))
	}
	task, err := ensureSessionTask(session, claims, content)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	allowCode := true
	if req.AllowCode != nil {
		allowCode = *req.AllowCode
	}
	requireApproval := true
	if req.RequireApproval != nil {
		requireApproval = *req.RequireApproval
	}
	if req.RequireApproval == nil {
		requireApproval = false
	}
	instructionStatus := "queued"
	needsApproval := shouldRequireAIApproval(content, requireApproval)
	if needsApproval {
		instructionStatus = "pending_approval"
	}
	instruction := &model.AIInstruction{SessionID: session.ID, UserID: claims.UserId, ProjectID: session.ProjectID, TaskID: task.ID, Content: content, Status: instructionStatus, AllowCode: allowCode, AutoPreview: req.AutoPreview, RequireApproval: requireApproval, AnalysisOnly: req.AnalysisOnly}
	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.CreateInstruction(instruction); err != nil {
		return c.JSON(e.Fail(err))
	}
	createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
		SessionID:     session.ID,
		TaskID:        task.ID,
		InstructionID: instruction.ID,
		EventType:     "instruction_queued",
		Stage:         "instruction_queued",
		Title:         "收到开发指令",
		Content:       buildTimelineContent(content),
		Status:        "info",
	})
	var approval *model.AIApproval
	if needsApproval {
		approval = &model.AIApproval{
			SessionID:     session.ID,
			TaskID:        task.ID,
			InstructionID: instruction.ID,
			RequestUserID: claims.UserId,
			Title:         buildApprovalTitle(content),
			Content:       content,
			RiskLevel:     "high",
			Status:        "pending",
		}
		if err := sessionRepo.CreateApproval(approval); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	if err := repo.NewAITaskRepo().CreateMessage(&model.AIMessage{TaskID: task.ID, Role: "user", Content: content}); err != nil {
		return c.JSON(e.Fail(err))
	}
	now := time.Now()
	session.Status = "active"
	if needsApproval {
		session.CurrentStage = "awaiting_approval"
	} else {
		session.CurrentStage = "instruction_queued"
	}
	session.LastTaskID = task.ID
	session.LastInstructionAt = &now
	if strings.TrimSpace(session.Title) == "" {
		session.Title = buildDefaultSessionTitle(session.WorkDir, content)
	}
	if err := sessionRepo.UpdateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	if needsApproval {
		task.Status = "pending_approval"
	} else {
		task.Status = "queued"
	}
	if err := repo.NewAITaskRepo().UpdateTask(task); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"session": session, "instruction": instruction, "task": task, "approval": approval}))
}
func GetAISessionState(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	sessionRepo := repo.NewAIDevSessionRepo()
	latestInstruction, err := sessionRepo.GetLatestInstructionBySessionID(session.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		latestInstruction = nil
		err = nil
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews, err := sessionRepo.GetPreviewsBySessionID(session.ID, 20)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews = refreshPreviewStatuses(sessionRepo, previews)
	pendingApproval := getPendingApprovalForSession(sessionRepo, session.ID)
	timelineEvents, err := sessionRepo.GetTimelineEventsBySessionID(session.ID, 20)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var currentTask *model.AITask
	var recentOutput string
	var errorSummary string
	var changedFiles []string
	var recentMessages []*model.AIMessage
	if session.LastTaskID > 0 {
		currentTask, _ = repo.NewAITaskRepo().GetTaskByID(session.LastTaskID)
		messages, msgErr := repo.NewAITaskRepo().GetMessagesByTaskID(session.LastTaskID)
		if msgErr != nil {
			return c.JSON(e.Fail(msgErr))
		}
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "agent" {
				recentOutput = messages[i].Content
				break
			}
		}
		if recentOutput == "" && len(messages) > 0 {
			recentOutput = messages[len(messages)-1].Content
		}
		errorSummary = extractAIErrorSummary(recentOutput)
		changedFiles = extractAIChangedFiles(recentOutput)
		recentOutput = summarizeAIRecentOutput(recentOutput)
		if len(messages) > 10 {
			recentMessages = messages[len(messages)-10:]
		} else {
			recentMessages = messages
		}
	}
	return c.JSON(e.Succ(fiber.Map{
		"session":           session,
		"currentTask":       currentTask,
		"latestInstruction": latestInstruction,
		"currentStage":      session.CurrentStage,
		"recentOutput":      recentOutput,
		"recentMessages":    recentMessages,
		"previews":          previews,
		"pendingApproval":   pendingApproval,
		"timelineEvents":    timelineEvents,
		"errorSummary":      errorSummary,
		"changedFiles":      changedFiles,
	}))
}
