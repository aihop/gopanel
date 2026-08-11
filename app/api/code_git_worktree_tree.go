package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func codeGitWorktreeMatchesCommit(workDir, commit string) (bool, error) {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return false, nil
	}
	localCommit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		return false, err
	}
	if _, err := runCodeGit(workDir, "merge-base", "--is-ancestor", localCommit, commit); err != nil {
		return false, nil
	}
	indexTree, err := runCodeGit(workDir, "write-tree")
	if err != nil {
		return false, nil
	}
	localTree, err := runCodeGit(workDir, "rev-parse", localCommit+"^{tree}")
	if err != nil {
		return false, err
	}
	targetTree, err := runCodeGit(workDir, "rev-parse", commit+"^{tree}")
	if err != nil {
		return false, err
	}
	if indexTree != localTree && indexTree != targetTree {
		return false, nil
	}
	index, err := os.CreateTemp("", "gopanel-worktree-index-")
	if err != nil {
		return false, err
	}
	indexPath := index.Name()
	if err := index.Close(); err != nil {
		_ = os.Remove(indexPath)
		return false, err
	}
	if err := os.Remove(indexPath); err != nil {
		return false, err
	}
	defer os.Remove(indexPath)
	if _, err := runCodeGitWithIndex(workDir, indexPath, "read-tree", "HEAD"); err != nil {
		return false, err
	}
	if _, err := runCodeGitWithIndex(workDir, indexPath, "add", "-A", "--", "."); err != nil {
		return false, err
	}
	worktreeTree, err := runCodeGitWithIndex(workDir, indexPath, "write-tree")
	if err != nil {
		return false, err
	}
	return worktreeTree == targetTree, nil
}

func runCodeGitWithIndex(workDir, indexPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), codeWorktreeCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...)
	command.Env = upsertEnvironment(codeGitEnvironment(), "GIT_INDEX_FILE", indexPath)
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
