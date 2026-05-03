package api

import (
	"errors"
	"net/http"
	neturl "net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func GetAIGroups(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	groupRepo := repo.NewAIGroupRepo()
	groups, total, err := groupRepo.GetGroups(page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": groups,
		"total": total,
	}))
}

var aiPreviewProbeClient = &http.Client{Timeout: 2 * time.Second}

func refreshPreviewStatuses(sessionRepo repo.IAIDevSessionRepo, previews []*model.AIPreview) []*model.AIPreview {
	for _, preview := range previews {
		if preview == nil {
			continue
		}
		refreshed := refreshSinglePreviewStatus(sessionRepo, preview)
		if refreshed != nil {
			preview = refreshed
		}
	}
	return previews
}

func refreshSinglePreviewStatus(sessionRepo repo.IAIDevSessionRepo, preview *model.AIPreview) *model.AIPreview {
	if preview == nil {
		return nil
	}

	now := time.Now()
	if preview.LastCheckedAt != nil && now.Sub(*preview.LastCheckedAt) < 15*time.Second {
		return preview
	}

	parsed, err := neturl.Parse(preview.URL)
	if err != nil {
		preview.Status = "invalid"
		preview.LastCheckedAt = &now
		if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
			global.LOG.Warnf("update invalid preview status failed: %v", updateErr)
		}
		return preview
	}

	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0":
		preview.Status = "local"
		preview.LastCheckedAt = &now
		if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
			global.LOG.Warnf("update local preview status failed: %v", updateErr)
		}
		return preview
	}

	status := probePreviewStatus(preview.URL)
	preview.Status = status
	preview.LastCheckedAt = &now
	if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
		global.LOG.Warnf("update preview status failed: %v", updateErr)
	}
	return preview
}

func probePreviewStatus(previewURL string) string {
	req, err := http.NewRequest(http.MethodHead, previewURL, nil)
	if err == nil {
		resp, headErr := aiPreviewProbeClient.Do(req)
		if headErr == nil {
			defer resp.Body.Close()
			return previewStatusFromHTTP(resp.StatusCode)
		}
	}

	req, err = http.NewRequest(http.MethodGet, previewURL, nil)
	if err != nil {
		return "invalid"
	}

	resp, getErr := aiPreviewProbeClient.Do(req)
	if getErr != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	return previewStatusFromHTTP(resp.StatusCode)
}

func previewStatusFromHTTP(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return "ready"
	case statusCode >= 400 && statusCode < 500:
		return "ready"
	case statusCode >= 500:
		return "unreachable"
	default:
		return "checking"
	}
}

func CreateAIGroup(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}

	group := &model.AIGroup{
		Name:        req.Name,
		Description: req.Description,
		CreatorID:   claims.UserId,
	}

	groupRepo := repo.NewAIGroupRepo()
	if err := groupRepo.CreateGroup(group); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(group))
}

// === AI Task APIs ===

// 获取历史任务列表
func GetAITasks(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	projectID, _ := strconv.Atoi(c.Query("projectId", "0"))

	aiRepo := repo.NewAITaskRepo()
	var tasks []*model.AITask
	var total int64
	var err error

	if projectID > 0 {
		tasks, total, err = aiRepo.GetTasksByProjectID(uint(projectID), page, limit)
	} else {
		tasks, total, err = aiRepo.GetTasksByUserID(claims.UserId, page, limit)
	}

	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"items": tasks,
		"total": total,
	}))
}

// 获取某个任务的历史对话记录
func GetAITaskMessages(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))

	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 权限校验
	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(err))
	}

	messages, err := aiRepo.GetMessagesByTaskID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(messages))
}

// 重命名任务
func UpdateAITask(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Title string `json:"title"`
	}
	if bindErr := c.Bind().JSON(&req); bindErr != nil {
		return c.JSON(e.Fail(bindErr))
	}

	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(err))
	}

	task.Title = req.Title
	if err := aiRepo.UpdateTask(task); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ())
}

// 删除任务
func DeleteAITask(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	taskID, _ := strconv.Atoi(c.Params("id"))

	aiRepo := repo.NewAITaskRepo()
	task, err := aiRepo.GetTaskByID(uint(taskID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(err))
	}

	if err := aiRepo.DeleteTask(uint(taskID)); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ())
}

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
	task := &model.AITask{
		UserID:    claims.UserId,
		ProjectID: session.ProjectID,
		Title:     title,
		AgentName: session.AgentName,
		WorkDir:   session.WorkDir,
		Status:    "queued",
	}
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

// 获取开发会话列表
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

	return c.JSON(e.Succ(fiber.Map{
		"items": sessions,
		"total": total,
	}))
}

// 获取开发会话详情
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

	var currentTask *model.AITask
	if session.LastTaskID > 0 {
		currentTask, _ = repo.NewAITaskRepo().GetTaskByID(session.LastTaskID)
	}

	return c.JSON(e.Succ(fiber.Map{
		"session":           session,
		"currentTask":       currentTask,
		"latestInstruction": latestInstruction,
		"previews":          previews,
	}))
}

// 创建开发会话
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

	session := &model.AIDevSession{
		UserID:       claims.UserId,
		ProjectID:    req.ProjectID,
		Title:        title,
		AgentName:    strings.TrimSpace(req.AgentName),
		WorkDir:      workDir,
		Status:       "active",
		CurrentStage: "idle",
	}

	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.CreateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(session))
}

// 向开发会话发送一条开发指令
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
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
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

	instruction := &model.AIInstruction{
		SessionID:       session.ID,
		UserID:          claims.UserId,
		ProjectID:       session.ProjectID,
		TaskID:          task.ID,
		Content:         content,
		Status:          "queued",
		AllowCode:       allowCode,
		AutoPreview:     req.AutoPreview,
		RequireApproval: requireApproval,
		AnalysisOnly:    req.AnalysisOnly,
	}

	sessionRepo := repo.NewAIDevSessionRepo()
	if err := sessionRepo.CreateInstruction(instruction); err != nil {
		return c.JSON(e.Fail(err))
	}

	if err := repo.NewAITaskRepo().CreateMessage(&model.AIMessage{
		TaskID:  task.ID,
		Role:    "user",
		Content: content,
	}); err != nil {
		return c.JSON(e.Fail(err))
	}

	now := time.Now()
	session.Status = "active"
	session.CurrentStage = "instruction_queued"
	session.LastTaskID = task.ID
	session.LastInstructionAt = &now
	if strings.TrimSpace(session.Title) == "" {
		session.Title = buildDefaultSessionTitle(session.WorkDir, content)
	}
	if err := sessionRepo.UpdateSession(session); err != nil {
		return c.JSON(e.Fail(err))
	}

	// 第一阶段先让会话与现有 AITask 形成最小闭环，
	// 后续真正的调度器/WS 消费者再根据 instruction 接管执行。
	task.Status = "queued"
	if err := repo.NewAITaskRepo().UpdateTask(task); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fiber.Map{
		"session":     session,
		"instruction": instruction,
		"task":        task,
	}))
}

// 获取开发会话状态摘要
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

	var currentTask *model.AITask
	var recentOutput string
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
	}))
}

// 获取开发会话预览列表
func GetAISessionPreviews(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))

	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	previews, err := repo.NewAIDevSessionRepo().GetPreviewsBySessionID(session.ID, 50)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews = refreshPreviewStatuses(repo.NewAIDevSessionRepo(), previews)

	return c.JSON(e.Succ(fiber.Map{
		"items": previews,
		"total": len(previews),
	}))
}
