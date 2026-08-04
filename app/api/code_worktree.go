package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

const codeWorktreeCommandTimeout = 15 * time.Second

type codeWorktreeCapability struct {
	Available         bool     `json:"available"`
	Reason            string   `json:"reason"`
	SourceDir         string   `json:"sourceDir,omitempty"`
	SourceDirs        []string `json:"sourceDirs,omitempty"`
	RepositoryCount   int      `json:"repositoryCount"`
	DirtyRepositories []string `json:"dirtyRepositories,omitempty"`
	SnapshotSupported bool     `json:"snapshotSupported"`
}

func aiProjectWorktreeRoot(userID uint) string {
	return filepath.Join(aiProjectUserRoot(userID), "worktrees")
}

func aiSessionWorktreeDir(userID, sessionID uint) string {
	return filepath.Join(aiProjectWorktreeRoot(userID), fmt.Sprintf("session_%d", sessionID))
}

func isManagedAISessionWorkDir(workDir string, userID uint) bool {
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if filepath.Dir(workDir) != aiProjectWorktreeRoot(userID) {
		return false
	}
	sessionID := strings.TrimPrefix(filepath.Base(workDir), "session_")
	if sessionID == filepath.Base(workDir) {
		return false
	}
	if _, err := strconv.ParseUint(sessionID, 10, 64); err != nil {
		return false
	}
	if info, err := os.Stat(filepath.Join(workDir, ".git")); err == nil && !info.IsDir() {
		return true
	}
	return isAISessionWorkspaceDirectory(workDir)
}

func GetCodeWorktreeCapability(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.Atoi(c.Params("id"))
	if err != nil || projectID <= 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if project.CreatorID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权访问该项目")))
	}
	return c.JSON(e.Succ(inspectCodeWorktreeCapability(project)))
}

func inspectCodeWorktreeCapability(project *model.AIProject) codeWorktreeCapability {
	sourceDirs := project.SourceDirs
	if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		sourceDirs = aiProjectWorkspaceSourceDirs(project.WorkDir)
		if len(sourceDirs) == 0 {
			sourceDirs = []string{project.WorkDir}
		}
	}
	if len(sourceDirs) == 0 {
		return codeWorktreeCapability{Reason: "source_unavailable"}
	}
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return codeWorktreeCapability{Reason: "source_unavailable"}
	}
	if len(candidates) == 0 {
		return codeWorktreeCapability{Reason: "not_git"}
	}
	resolvedDirs := make([]string, len(candidates))
	for index, candidate := range candidates {
		resolvedDirs[index] = candidate.SourceDir
	}
	result := codeWorktreeCapability{
		Available: true, SourceDirs: resolvedDirs, RepositoryCount: len(resolvedDirs),
		DirtyRepositories: codeCandidateNames(candidates), SnapshotSupported: true,
	}
	if len(candidates) == 1 {
		result.SourceDir = candidates[0].SourceDir
	}
	return result
}

func createCodeSessionWorktree(session *model.AIDevSession, project *model.AIProject, includeUncommitted ...bool) error {
	sourceDirs := project.SourceDirs
	if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		sourceDirs = aiProjectWorkspaceSourceDirs(project.WorkDir)
		if len(sourceDirs) == 0 {
			sourceDirs = []string{project.WorkDir}
		}
	}
	policy, err := codeProjectDeliveryPolicy(project, sourceDirs)
	if err != nil {
		return fmt.Errorf("项目交付策略无效：%w", err)
	}
	prepared, err := prepareDiscoveredCodeRepositoriesWithPolicy(sourceDirs, policy, includeUncommitted...)
	if err != nil {
		if errors.Is(err, errCodeGitAuthentication) {
			return err
		}
		return fmt.Errorf("当前项目不支持 Git Worktree 隔离：%w", err)
	}
	if len(prepared) > 1 {
		return createCodeSessionRepositoryWorktrees(session, project, prepared)
	}
	repository := prepared[0]
	worktreeDir := aiSessionWorktreeDir(session.UserID, session.ID)
	if _, err := os.Lstat(worktreeDir); !errors.Is(err, os.ErrNotExist) {
		return errors.New("会话 Worktree 目录已存在")
	}
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0750); err != nil {
		return fmt.Errorf("创建 Worktree 管理目录失败：%w", err)
	}
	branch := fmt.Sprintf("gopanel/code-%d-%d", session.ID, time.Now().Unix())
	if _, err := runCodeGit(repository.SourceDir, "worktree", "add", "-b", branch, worktreeDir, repository.BaseCommit); err != nil {
		return err
	}
	if err := installCodeManagedPushGuard(worktreeDir); err != nil {
		_, _ = runCodeGit(repository.SourceDir, "worktree", "remove", "--force", worktreeDir)
		_, _ = runCodeGit(repository.SourceDir, "branch", "-D", "--", branch)
		return err
	}
	if err := applyCodeRepositorySnapshot(repository, worktreeDir, prepared); err != nil {
		_, _ = runCodeGit(repository.SourceDir, "worktree", "remove", "--force", worktreeDir)
		_, _ = runCodeGit(repository.SourceDir, "branch", "-D", "--", branch)
		return err
	}
	session.SourceWorkDir = repository.SourceDir
	session.WorkDir = worktreeDir
	session.WorktreeBranch = branch
	session.TargetBranch = repository.TargetBranch
	session.BaseCommit = repository.BaseCommit
	session.RemoteName = repository.RemoteName
	session.RemoteBranch = repository.RemoteBranch
	session.RemoteCommit = repository.RemoteCommit
	session.RepositorySync = repository.SyncStatus
	session.IsolationMode = "single_worktree"
	return nil
}

func rollbackCodeSessionWorktree(session *model.AIDevSession) {
	if session != nil && session.IsolationMode == codeIsolationMultiWorktree {
		rollbackCodeSessionRepositoryWorktrees(session)
		return
	}
	if session == nil || session.SourceWorkDir == "" || session.WorktreeBranch == "" {
		return
	}
	if _, err := runCodeGit(session.SourceWorkDir, "worktree", "remove", "--force", session.WorkDir); err != nil {
		global.LOG.Errorf("Rollback Code worktree %d failed: %v", session.ID, err)
		return
	}
	if _, err := runCodeGit(session.SourceWorkDir, "branch", "-D", session.WorktreeBranch); err != nil {
		global.LOG.Errorf("Rollback Code worktree branch %s failed: %v", session.WorktreeBranch, err)
	}
}

func cleanupCodeSessionWorktree(session *model.AIDevSession) error {
	if session != nil && session.IsolationMode == codeIsolationMultiWorktree {
		return cleanupCodeSessionRepositoryWorktrees(session)
	}
	if session == nil || session.SourceWorkDir == "" || session.WorktreeBranch == "" {
		return nil
	}
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) {
		return errors.New("会话 Worktree 不在 GoPanel 管理目录中")
	}
	status, err := runCodeGit(session.WorkDir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) != "" {
		return errors.New("会话 Worktree 仍有未提交修改，已保留目录避免数据丢失")
	}
	if _, err := runCodeGit(session.SourceWorkDir, "merge-base", "--is-ancestor", session.WorktreeBranch, "HEAD"); err != nil {
		return errors.New("会话 Worktree 包含尚未合并的提交，已保留目录避免数据丢失")
	}
	if _, err := runCodeGit(session.SourceWorkDir, "worktree", "remove", session.WorkDir); err != nil {
		return err
	}
	if _, err := runCodeGit(session.SourceWorkDir, "branch", "-d", "--", session.WorktreeBranch); err != nil {
		return err
	}
	return nil
}

func cleanupDeliveredCodeSessionWorktrees(session *model.AIDevSession) error {
	if session == nil || session.ID == 0 {
		return nil
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err == nil {
		return cleanupCodeDeliveryWorktree(&delivery)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return err
	}
	workspaceDir, err := codeMultiRepositoryWorkspaceDir(session, repositories)
	if err != nil {
		return err
	}
	snapshot := *session
	snapshot.WorkDir = workspaceDir
	snapshot.IsolationMode = codeIsolationMultiWorktree
	return cleanupCodeSessionRepositoryWorktrees(&snapshot)
}

func runCodeGit(workDir string, args ...string) (string, error) {
	return runCodeGitWithTimeout(workDir, codeWorktreeCommandTimeout, args...)
}

func runCodeGitWithTimeout(workDir string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	commandArgs := append([]string{"-C", workDir}, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", errors.New("Git 操作超时")
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("Git 操作失败：%w", normalizeCodeGitCommandError(message))
	}
	return strings.TrimSpace(string(output)), nil
}
