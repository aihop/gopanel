package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/gofiber/fiber/v3"
)

type codeGitDeliveryResult struct {
	Status        string   `json:"status"`
	Commit        string   `json:"commit,omitempty"`
	Branch        string   `json:"branch,omitempty"`
	ConflictFiles []string `json:"conflictFiles,omitempty"`
}

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
	sourceStatus, err := runCodeGit(session.SourceWorkDir, "status", "--porcelain")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if strings.TrimSpace(sourceStatus) != "" {
		return codeGitDeliveryResult{}, errors.New("源仓库存在未提交变更，无法安全合并")
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
	if err := cleanupCodeSessionWorktree(session); err != nil {
		return codeGitDeliveryResult{}, fmt.Errorf("合并成功，但清理 Worktree 失败：%w", err)
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

func runCodeGitDelivery(c fiber.Ctx, operation func(*model.AIDevSession) (codeGitDeliveryResult, error)) error {
	session, _, err := getCodeGitSessionContext(c)
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
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func CommitCodeGitChanges(c fiber.Ctx) error {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return runCodeGitDelivery(c, func(session *model.AIDevSession) (codeGitDeliveryResult, error) {
		return commitCodeSessionWorktree(session, req.Message)
	})
}

func MergeCodeSessionWorktree(c fiber.Ctx) error {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("合并操作需要明确确认")))
	}
	return runCodeGitDelivery(c, func(session *model.AIDevSession) (codeGitDeliveryResult, error) {
		result, err := mergeCodeSessionWorktree(session)
		if err != nil || result.Status != "merged" {
			return result, err
		}
		session.WorkDir = session.SourceWorkDir
		session.SourceWorkDir = ""
		session.WorktreeBranch = ""
		if err := repo.NewAIDevSessionRepo().UpdateSession(session); err != nil {
			return codeGitDeliveryResult{}, fmt.Errorf("Git 合并已完成，但会话元数据更新失败：%w", err)
		}
		return result, nil
	})
}
