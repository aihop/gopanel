package api

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeGitDeliveryResult struct {
	Status         string                         `json:"status"`
	ResultType     string                         `json:"resultType,omitempty"`
	ErrorMessage   string                         `json:"errorMessage,omitempty"`
	Commit         string                         `json:"commit,omitempty"`
	Branch         string                         `json:"branch,omitempty"`
	RepositoryID   string                         `json:"repositoryId,omitempty"`
	RepositoryName string                         `json:"repositoryName,omitempty"`
	ConflictFiles  []string                       `json:"conflictFiles,omitempty"`
	Repositories   []codeRepositoryDeliveryResult `json:"repositories,omitempty"`
}

const codeDeliveryQueueTimeout = 2 * time.Minute

func validateCodeGitCommitMessage(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" || len([]rune(message)) > 500 {
		return "", errors.New("提交说明必须为 1 到 500 个字符")
	}
	if strings.ContainsRune(message, '\x00') {
		return "", errors.New("提交说明包含无效字符")
	}
	return message, nil
}

func validateCodeWorktreeDeliverySession(session *model.AIDevSession) error {
	if session == nil || strings.TrimSpace(session.SourceWorkDir) == "" || strings.TrimSpace(session.WorktreeBranch) == "" {
		return errors.New("当前会话未启用 Git Worktree 隔离")
	}
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) {
		return errors.New("会话 Worktree 不在 GoPanel 管理目录中")
	}
	return nil
}

func commitCodeSessionWorktree(session *model.AIDevSession, message string) (codeGitDeliveryResult, error) {
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return codeGitDeliveryResult{}, err
	}
	message, err := validateCodeGitCommitMessage(message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	staged, err := runCodeGit(session.WorkDir, "diff", "--cached", "--name-only")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if strings.TrimSpace(staged) == "" {
		return codeGitDeliveryResult{}, errors.New("暂存区没有可提交的变更")
	}
	if err := validateCodeGitStagedChanges(session.WorkDir); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if _, err := runCodeGit(session.WorkDir, "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local", "commit", "-m", message); err != nil {
		return codeGitDeliveryResult{}, err
	}
	commit, err := runCodeGit(session.WorkDir, "rev-parse", "HEAD")
	return codeGitDeliveryResult{Status: "committed", Commit: commit, Branch: session.WorktreeBranch}, err
}

func mergeCodeSessionWorktree(session *model.AIDevSession) (codeGitDeliveryResult, error) {
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return codeGitDeliveryResult{}, err
	}
	worktreeStatus, err := runCodeGit(session.WorkDir, "status", "--porcelain")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if strings.TrimSpace(worktreeStatus) != "" {
		return codeGitDeliveryResult{}, errors.New("Worktree 仍有未提交变更，请先提交")
	}
	targetBranch := session.TargetBranch
	if targetBranch == "" {
		targetBranch, _ = runCodeGit(session.SourceWorkDir, "branch", "--show-current")
	}
	if _, err := refreshCodeRepositoryTargetWithCredential(
		session.SourceWorkDir, targetBranch, session.RemoteName, codeProjectGitCredentialID(session.ProjectID),
	); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := syncCodeWorktreeWithTarget(session.WorkDir, targetBranch); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if _, err := runCodeGit(session.SourceWorkDir, "merge", "--no-ff", "--no-edit", session.WorktreeBranch); err != nil {
		conflicts := codeGitConflictFiles(session.SourceWorkDir)
		_, _ = runCodeGit(session.SourceWorkDir, "merge", "--abort")
		if len(conflicts) > 0 {
			return codeGitDeliveryResult{Status: "conflict", Branch: session.WorktreeBranch, ConflictFiles: conflicts}, nil
		}
		return codeGitDeliveryResult{}, err
	}
	commit, err := runCodeGit(session.SourceWorkDir, "rev-parse", "HEAD")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	return codeGitDeliveryResult{Status: "merged", Commit: commit, Branch: session.WorktreeBranch}, nil
}

func codeGitConflictFiles(workDir string) []string {
	output, err := runCodeGit(workDir, "diff", "--name-only", "--diff-filter=U")
	if err != nil || strings.TrimSpace(output) == "" {
		return nil
	}
	files := strings.Split(output, "\n")
	if len(files) > 200 {
		files = files[:200]
	}
	return files
}

func runCodeGitDelivery(c fiber.Ctx, action string, operation func(*model.AIDevSession) (codeGitDeliveryResult, error)) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, _, err := getCodeGitSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var result codeGitDeliveryResult
	err = runCodeSessionGitMutation(session, func(current *model.AIDevSession) error {
		var operationErr error
		result, operationErr = operation(current)
		return operationErr
	})
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, action, "failed", session.WorktreeBranch, err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	status := "success"
	if result.Status == "conflict" {
		status = "conflict"
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, action, status, result.Branch, result.Status, c.IP(), startedAt, codeAuditMeta{"commit": result.Commit, "conflictFiles": result.ConflictFiles})
	return c.JSON(e.Succ(result))
}

func getCodeDeliverySessionContext(c fiber.Ctx) (*model.AIDevSession, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return nil, err
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		if err := validateCodeMultiWorktreeDeliverySession(session, claims); err != nil {
			return nil, err
		}
		return session, nil
	}
	if hasCodeMultiRepositoryDelivery(session.ID) {
		return session, nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
				return nil, err
			}
			return session, nil
		}
		return nil, err
	}
	if delivery.UserID != session.UserID || delivery.ProjectID != session.ProjectID {
		return nil, errors.New("交付记录与当前会话不匹配")
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil || len(sourceDirs) == 0 {
		return nil, errors.New("交付源目录不可用")
	}
	storedSource, storedErr := filepath.EvalSymlinks(filepath.Clean(delivery.SourceWorkDir))
	if storedErr != nil || !repositoryWithinSourceDirs(storedSource, sourceDirs) {
		return nil, errors.New("交付源目录与项目配置不一致")
	}
	if err := validateAIProjectWorkDirForClaims(storedSource, claims); err != nil {
		return nil, err
	}
	return session, nil
}

func runCodeWorktreeMerge(c fiber.Ctx, operation func(*model.AIDevSession) (codeGitDeliveryResult, error)) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionDelivery, false)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer lease.Release()
	result, err := operation(session)
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "worktree_merge", "failed", session.WorktreeBranch, err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	status := "success"
	if result.Status == "conflict" {
		status = "conflict"
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "worktree_merge", status, result.Branch, result.Status, c.IP(), startedAt, codeAuditMeta{"commit": result.Commit, "conflictFiles": result.ConflictFiles})
	return c.JSON(e.Succ(result))
}

func CommitCodeGitChanges(c fiber.Ctx) error {
	var req struct {
		Message      string `json:"message"`
		RepositoryID string `json:"repositoryId"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return runCodeGitDelivery(c, "git_commit", func(session *model.AIDevSession) (codeGitDeliveryResult, error) {
		if session.IsolationMode == codeIsolationMultiWorktree {
			return commitCodeSessionRepository(session, req.RepositoryID, req.Message)
		}
		return commitCodeSessionWorktree(session, req.Message)
	})
}

func MergeCodeSessionWorktree(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("合并操作需要明确确认")))
	}
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	job, err := enqueueCodeDeliveryJob(session, claims.UserId, c.IP())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	view, err := loadCodeDeliveryJobView(job.SessionID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(view))
}

func GetCodeDeliveryJob(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if session.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权访问该交付任务")))
	}
	view, err := loadCodeDeliveryJobView(session.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.JSON(e.Succ(nil))
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(view))
}
