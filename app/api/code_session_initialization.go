package api

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
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

const (
	codeSessionStageSyncingBase          = "syncing_base"
	codeSessionStageInitializationFailed = "initialization_failed"
	codeSessionInitializationWorkers     = 2
)

type codeSessionInitializationRunner struct {
	once   sync.Once
	mu     sync.Mutex
	queued map[uint]struct{}
	queue  chan uint
}

var backgroundCodeSessionInitialization = &codeSessionInitializationRunner{
	queued: make(map[uint]struct{}),
	queue:  make(chan uint, 128),
}

func StartCodeSessionInitialization() {
	backgroundCodeSessionInitialization.once.Do(func() {
		for range codeSessionInitializationWorkers {
			go backgroundCodeSessionInitialization.worker()
		}
		enqueuePersistedCodeSessionInitializations()
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					enqueuePersistedCodeSessionInitializations()
				case <-codeExecutions.stop:
					return
				}
			}
		}()
	})
}

func enqueuePersistedCodeSessionInitializations() {
	if global.DB == nil {
		return
	}
	var sessionIDs []uint
	if err := global.DB.Model(&model.AIDevSession{}).
		Where("status = ?", codeSessionStatusInitializing).
		Order("created_at ASC").Limit(500).Pluck("id", &sessionIDs).Error; err != nil {
		global.LOG.Errorf("Load initializing Code sessions failed: %v", err)
		return
	}
	for _, sessionID := range sessionIDs {
		enqueueCodeSessionInitialization(sessionID)
	}
}

func enqueueCodeSessionInitialization(sessionID uint) {
	if sessionID == 0 || codeExecutions.isStopping() {
		return
	}
	runner := backgroundCodeSessionInitialization
	runner.mu.Lock()
	if _, exists := runner.queued[sessionID]; exists {
		runner.mu.Unlock()
		return
	}
	runner.queued[sessionID] = struct{}{}
	runner.mu.Unlock()
	select {
	case runner.queue <- sessionID:
	default:
		runner.mu.Lock()
		delete(runner.queued, sessionID)
		runner.mu.Unlock()
	}
}

func (runner *codeSessionInitializationRunner) worker() {
	for {
		select {
		case sessionID := <-runner.queue:
			if err := initializeCodeSession(sessionID); err != nil && global.LOG != nil {
				global.LOG.Errorf("Initialize Code session %d failed: %v", sessionID, err)
			}
			runner.mu.Lock()
			delete(runner.queued, sessionID)
			runner.mu.Unlock()
			var status string
			if err := global.DB.Model(&model.AIDevSession{}).Where("id = ?", sessionID).Pluck("status", &status).Error; err == nil && status == codeSessionStatusInitializing {
				enqueueCodeSessionInitialization(sessionID)
			}
		case <-codeExecutions.stop:
			return
		}
	}
}

func initializeCodeSession(sessionID uint) error {
	session, err := repo.NewAIDevSessionRepo().GetSessionByID(sessionID)
	if err != nil || session.Status != codeSessionStatusInitializing {
		return err
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil {
		return failCodeSessionInitialization(session, errors.New("项目不存在"))
	}
	cleanupInterruptedCodeSessionInitialization(session, project)
	if err = createCodeSessionWorktreeWithLease(session, project, codeSessionIncludesUncommitted(session)); err != nil {
		return failCodeSessionInitialization(session, err)
	}
	task := &model.AITask{
		UserID: session.UserID, SessionID: session.ID, ProjectID: session.ProjectID,
		Title: session.Title, AgentName: session.AgentName, WorkDir: session.WorkDir, Status: codeSessionStatusActive,
	}
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIInstruction{}).
			Where("session_id = ? AND task_id = 0", session.ID).
			Update("task_id", task.ID).Error; err != nil {
			return err
		}
		result := tx.Model(&model.AIDevSession{}).
			Where("id = ? AND status = ?", session.ID, codeSessionStatusInitializing).
			Updates(map[string]any{
				"work_dir": session.WorkDir, "source_work_dir": session.SourceWorkDir,
				"worktree_branch": session.WorktreeBranch, "target_branch": session.TargetBranch,
				"base_commit": session.BaseCommit, "remote_name": session.RemoteName,
				"remote_branch": session.RemoteBranch, "remote_commit": session.RemoteCommit,
				"repository_sync": session.RepositorySync, "isolation_mode": session.IsolationMode,
				"last_task_id": task.ID, "status": codeSessionStatusActive,
				"current_stage": "idle", "initialization_error": "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("会话初始化状态已变化")
		}
		return nil
	})
	if err != nil {
		rollbackCodeSessionWorktree(session)
		return failCodeSessionInitialization(session, err)
	}
	_ = reconcileWebsiteIssueCodeTasks()
	return nil
}

func codeSessionIncludesUncommitted(session *model.AIDevSession) bool {
	return session == nil || session.IncludeUncommitted == nil || *session.IncludeUncommitted
}

func cleanupInterruptedCodeSessionInitialization(session *model.AIDevSession, project *model.AIProject) {
	if session == nil || session.ID == 0 || session.IsolationMode != "" || session.WorktreeBranch != "" {
		return
	}
	var repositoryCount int64
	if err := global.DB.Model(&model.AIDevSessionRepository{}).Where("session_id = ?", session.ID).Count(&repositoryCount).Error; err == nil && repositoryCount > 0 {
		snapshot := *session
		snapshot.WorkDir = aiSessionWorktreeDir(session.UserID, session.ID)
		snapshot.IsolationMode = codeIsolationMultiWorktree
		rollbackCodeSessionRepositoryWorktrees(&snapshot)
		return
	}
	worktreeDir := aiSessionWorktreeDir(session.UserID, session.ID)
	if _, err := os.Lstat(worktreeDir); err != nil {
		return
	}
	if !isPathInside(worktreeDir, aiProjectWorktreeRoot(session.UserID)) {
		return
	}
	for _, sourceDir := range codeProjectSourceDirs(project) {
		_, _ = runCodeGit(sourceDir, "worktree", "remove", "--force", worktreeDir)
		_ = os.RemoveAll(worktreeDir)
		_, _ = runCodeGit(sourceDir, "worktree", "prune")
		branches, _ := runCodeGit(sourceDir, "for-each-ref", "--format=%(refname:short)", "refs/heads/gopanel/code-"+strconv.FormatUint(uint64(session.ID), 10)+"-")
		for _, branch := range strings.Fields(branches) {
			_, _ = runCodeGit(sourceDir, "branch", "-D", "--", branch)
		}
	}
}

func failCodeSessionInitialization(session *model.AIDevSession, initializationErr error) error {
	message := truncateCodeAuditDetail(initializationErr.Error())
	if session != nil && session.ID > 0 {
		_ = global.DB.Model(&model.AIDevSession{}).Where("id = ?", session.ID).Updates(map[string]any{
			"status": codeSessionStatusFailed, "current_stage": codeSessionStageInitializationFailed,
			"initialization_error": strings.TrimSpace(message),
		}).Error
	}
	return initializationErr
}

func GetCodeSessionInitialization(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"id": session.ID, "status": session.Status, "currentStage": session.CurrentStage,
		"initializationError": session.InitializationErr,
	}))
}

func RetryCodeSessionInitialization(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result := global.DB.Model(&model.AIDevSession{}).
		Where("id = ? AND status = ? AND current_stage = ?", session.ID, codeSessionStatusFailed, codeSessionStageInitializationFailed).
		Updates(map[string]any{
			"status": codeSessionStatusInitializing, "current_stage": codeSessionStageSyncingBase,
			"initialization_error": "",
		})
	if result.Error != nil {
		return c.JSON(e.Fail(result.Error))
	}
	if result.RowsAffected != 1 {
		return c.JSON(e.Fail(errors.New("当前会话不需要重新初始化")))
	}
	enqueueCodeSessionInitialization(session.ID)
	return c.JSON(e.Succ(fiber.Map{
		"id": session.ID, "status": codeSessionStatusInitializing, "currentStage": codeSessionStageSyncingBase,
	}))
}
