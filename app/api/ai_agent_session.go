package api

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
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
	return ensureSessionTaskWithDB(global.DB, session, claims, hint)
}

func ensureSessionTaskWithDB(db *gorm.DB, session *model.AIDevSession, claims *token.CustomClaims, hint string) (*model.AITask, error) {
	if session.LastTaskID > 0 {
		var task model.AITask
		err := db.Where("id = ?", session.LastTaskID).First(&task).Error
		if err == nil {
			changed := false
			if task.SessionID == 0 {
				task.SessionID = session.ID
				changed = true
			}
			if task.NativeSessionID == "" && session.NativeSessionID != "" {
				task.NativeSessionID = session.NativeSessionID
				changed = true
			}
			if changed {
				if err := db.Save(&task).Error; err != nil {
					return nil, err
				}
			}
			return &task, nil
		}
	}
	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = buildDefaultSessionTitle(session.WorkDir, hint)
	}
	task := &model.AITask{UserID: claims.UserId, SessionID: session.ID, ProjectID: session.ProjectID, Title: title, AgentName: session.AgentName, NativeSessionID: session.NativeSessionID, WorkDir: session.WorkDir, Status: "queued"}
	if err := db.Create(task).Error; err != nil {
		return nil, err
	}
	session.LastTaskID = task.ID
	session.Status = "active"
	session.CurrentStage = "task_ready"
	if err := db.Save(session).Error; err != nil {
		return nil, err
	}
	return task, nil
}
func GetAISessions(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	page, limit = normalizeCodePage(page, limit, 20)
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
		Title              string               `json:"title"`
		WorkDir            string               `json:"workDir"`
		ProjectID          uint                 `json:"projectId"`
		ExecutorID         string               `json:"executorId"`
		ApprovalPolicy     string               `json:"approvalPolicy"`
		Isolated           bool                 `json:"isolated"`
		IncludeUncommitted bool                 `json:"includeUncommitted"`
		Provider           *codeProviderRequest `json:"provider"`
		CodexProvider      *codeProviderRequest `json:"codexProvider"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}
	workDir := strings.TrimSpace(req.WorkDir)
	var project *model.AIProject
	var err error
	if req.ProjectID > 0 {
		project, err = repo.NewAIProjectRepo().GetProjectByID(req.ProjectID)
		if err != nil {
			return c.JSON(e.Fail(errors.New("项目不存在")))
		}
		if project.CreatorID != claims.UserId && claims.Role != constant.UserRoleSuper {
			return c.JSON(e.Fail(errors.New("无权访问该项目")))
		}
		if strings.TrimSpace(project.WorkDir) != "" || len(project.SourceDirs) == 1 {
			workDir, err = aiProjectSessionWorkDir(project, claims)
			if err != nil {
				return c.JSON(e.Fail(err))
			}
		}
	}
	if workDir == "" {
		if claims.Role == constant.UserRoleSubAdmin {
			workDir = strings.TrimSpace(claims.FileBaseDir)
		} else {
			workDir = "."
		}
	}
	if claims.Role == constant.UserRoleSubAdmin {
		if err := validateAIProjectWorkDirForClaims(workDir, claims); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	executorID, err := validateCodeExecutorAvailable(req.ExecutorID, claims.Role)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	providerRequest := req.Provider
	if providerRequest == nil {
		providerRequest = req.CodexProvider
	}
	provider, err := normalizeCodeProviderRequest(executorID, providerRequest)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	approvalPolicy, err := normalizeCodeApprovalPolicy(req.ApprovalPolicy)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateCodeExecutorApprovalPolicy(executorID, approvalPolicy); err != nil {
		return c.JSON(e.Fail(err))
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = buildDefaultSessionTitle(workDir, "")
	}
	session := &model.AIDevSession{UserID: claims.UserId, ProjectID: req.ProjectID, Title: title, AgentName: executorID, WorkDir: workDir, Status: "active", CurrentStage: "idle", ApprovalPolicy: approvalPolicy}
	if err := setCodeProviderOnSession(session, provider); err != nil {
		return c.JSON(e.Fail(err))
	}
	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.CreateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Isolated {
		if project == nil {
			_ = sessionRepo.DeleteSession(session.ID)
			return c.JSON(e.Fail(errors.New("Git Worktree 隔离仅支持项目会话")))
		}
		if err := createCodeSessionWorktree(session, project, req.IncludeUncommitted); err != nil {
			_ = sessionRepo.DeleteSession(session.ID)
			return c.JSON(e.Fail(err))
		}
		if err := sessionRepo.UpdateSession(session); err != nil {
			rollbackCodeSessionWorktree(session)
			_ = sessionRepo.DeleteSession(session.ID)
			return c.JSON(e.Fail(err))
		}
	}
	task := &model.AITask{
		UserID: claims.UserId, SessionID: session.ID, ProjectID: session.ProjectID,
		Title: session.Title, AgentName: session.AgentName, WorkDir: session.WorkDir, Status: "active",
	}
	if err := repo.NewAITaskRepo().CreateTask(task); err != nil {
		rollbackCodeSessionWorktree(session)
		_ = sessionRepo.DeleteSession(session.ID)
		return c.JSON(e.Fail(err))
	}
	session.LastTaskID = task.ID
	if err := sessionRepo.UpdateSession(session); err != nil {
		_ = repo.NewAITaskRepo().DeleteTask(task.ID)
		rollbackCodeSessionWorktree(session)
		_ = sessionRepo.DeleteSession(session.ID)
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
	allowCode := true
	if req.AllowCode != nil {
		allowCode = *req.AllowCode
	}
	if req.AnalysisOnly || !allowCode {
		return c.JSON(e.Fail(errors.New("当前执行器尚不支持可强制保证的只读分析模式")))
	}
	if session.AgentName == "terminal" {
		return c.JSON(e.Fail(errors.New("纯终端会话不支持后台对话执行，请切换到高级终端")))
	}
	if err := validateCodeTokenBudget(session); err != nil {
		return c.JSON(e.Fail(err))
	}
	requireApproval := codeSessionRequiresRiskApproval(session)
	needsApproval := shouldRequireAIApproval(content, requireApproval)
	instructionStatus := "queued"
	if needsApproval {
		instructionStatus = "pending_approval"
	}
	var task *model.AITask
	var instruction *model.AIInstruction
	var approval *model.AIApproval
	if err := global.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		task, txErr = ensureSessionTaskWithDB(tx, session, claims, content)
		if txErr != nil {
			return txErr
		}
		instruction = &model.AIInstruction{SessionID: session.ID, UserID: claims.UserId, ProjectID: session.ProjectID, TaskID: task.ID, Content: content, Status: instructionStatus, AllowCode: allowCode, AutoPreview: req.AutoPreview, RequireApproval: requireApproval, AnalysisOnly: req.AnalysisOnly}
		if txErr = tx.Create(instruction).Error; txErr != nil {
			return txErr
		}
		if txErr = tx.Create(&model.AITimelineEvent{
			SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
			EventType: "instruction_queued", Stage: "instruction_queued", Title: "收到开发指令",
			Content: buildTimelineContent(content), Status: "info",
		}).Error; txErr != nil {
			return txErr
		}
		if needsApproval {
			approval = &model.AIApproval{
				SessionID: session.ID, TaskID: task.ID, InstructionID: instruction.ID,
				RequestUserID: claims.UserId, Title: buildApprovalTitle(content), Content: content,
				RiskLevel: "high", Status: "pending",
			}
			if txErr = tx.Create(approval).Error; txErr != nil {
				return txErr
			}
		}
		if txErr = tx.Create(&model.AIMessage{SessionID: session.ID, TaskID: task.ID, Role: "user", Content: content}).Error; txErr != nil {
			return txErr
		}
		now := time.Now()
		session.Status = "active"
		session.LastInstructionAt = &now
		if strings.TrimSpace(session.Title) == "" {
			session.Title = buildDefaultSessionTitle(session.WorkDir, content)
		}
		if txErr = tx.Model(&model.AIDevSession{}).Where("id = ?", session.ID).
			Updates(map[string]any{"status": "active", "last_instruction_at": now, "title": session.Title}).Error; txErr != nil {
			return txErr
		}
		return reconcileCodeTaskState(tx, session, task, instructionStatus, map[bool]string{true: "awaiting_approval", false: "instruction_queued"}[needsApproval])
	}); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !needsApproval {
		enqueueCodeInstruction(instruction.ID)
	} else {
		go service.NotifyCodeSession(session, task, service.CodeNotifyApproval, buildTimelineContent(content))
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
	latestRun, err := sessionRepo.GetLatestExecutionRunBySessionID(session.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		latestRun = nil
		err = nil
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if latestRun != nil {
		latestRun.RawOutput = ""
	}
	tokenUsage, err := loadCodeTokenUsage(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if session.LastTaskID > 0 {
		taskRepo := repo.NewAITaskRepo()
		currentTask, _ = taskRepo.GetTaskByID(session.LastTaskID)
		messages, msgErr := taskRepo.GetMessagesByTaskID(session.LastTaskID)
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
		"codexRuntime":      getCodexRuntimeState(session),
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
		"latestRun":         latestRun,
		"tokenUsage":        tokenUsage,
	}))
}
