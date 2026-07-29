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
)

const codeWorktreeCommandTimeout = 15 * time.Second

type codeWorktreeCapability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
	SourceDir string `json:"sourceDir,omitempty"`
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
	info, err := os.Stat(filepath.Join(workDir, ".git"))
	return err == nil && !info.IsDir()
}

func GetCodeWorktreeCapability(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.Atoi(c.Params("id"))
	if err != nil || projectID <= 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := repo.NewAIGroupRepo().GetGroupByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if project.CreatorID != claims.UserId && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("无权访问该项目")))
	}
	return c.JSON(e.Succ(inspectCodeWorktreeCapability(project)))
}

func inspectCodeWorktreeCapability(project *model.AIGroup) codeWorktreeCapability {
	sourceDirs := project.SourceDirs
	if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		sourceDirs = aiProjectWorkspaceSourceDirs(project.WorkDir)
	}
	if len(sourceDirs) != 1 {
		return codeWorktreeCapability{Reason: "multi_source"}
	}
	sourceDir, err := filepath.EvalSymlinks(filepath.Clean(sourceDirs[0]))
	if err != nil {
		return codeWorktreeCapability{Reason: "source_unavailable"}
	}
	root, err := runCodeGit(sourceDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return codeWorktreeCapability{Reason: "not_git"}
	}
	root, err = filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(root)))
	if err != nil || root != sourceDir {
		return codeWorktreeCapability{Reason: "not_git_root"}
	}
	return codeWorktreeCapability{Available: true, SourceDir: sourceDir}
}

func createCodeSessionWorktree(session *model.AIDevSession, project *model.AIGroup) error {
	capability := inspectCodeWorktreeCapability(project)
	if !capability.Available {
		return fmt.Errorf("当前项目不支持 Git Worktree 隔离：%s", capability.Reason)
	}
	worktreeDir := aiSessionWorktreeDir(session.UserID, session.ID)
	if _, err := os.Lstat(worktreeDir); !errors.Is(err, os.ErrNotExist) {
		return errors.New("会话 Worktree 目录已存在")
	}
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0750); err != nil {
		return fmt.Errorf("创建 Worktree 管理目录失败：%w", err)
	}
	branch := fmt.Sprintf("gopanel/code-%d-%d", session.ID, time.Now().Unix())
	if _, err := runCodeGit(capability.SourceDir, "worktree", "add", "-b", branch, worktreeDir, "HEAD"); err != nil {
		return err
	}
	session.SourceWorkDir = capability.SourceDir
	session.WorkDir = worktreeDir
	session.WorktreeBranch = branch
	return nil
}

func rollbackCodeSessionWorktree(session *model.AIDevSession) {
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

func runCodeGit(workDir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codeWorktreeCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"-C", workDir}, args...)
	output, err := exec.CommandContext(ctx, "git", commandArgs...).CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", errors.New("Git 操作超时")
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("Git 操作失败：%s", message)
	}
	return strings.TrimSpace(string(output)), nil
}
