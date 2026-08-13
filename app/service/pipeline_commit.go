package service

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

var pipelineCommitPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

type pipelineGitRunner func(cmd *exec.Cmd, action string) error

func normalizePipelineExpectedCommit(value string) (string, error) {
	commit := strings.ToLower(strings.TrimSpace(value))
	if commit == "" {
		return "", nil
	}
	if !pipelineCommitPattern.MatchString(commit) {
		return "", buserr.New(constant.ErrPipelineExpectedCommitInvalid)
	}
	return commit, nil
}

func fetchAndCheckoutPipelineCommit(ctx context.Context, logger *PipelineLogger, workspace, branch, expectedCommit string, runGit pipelineGitRunner) error {
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--prune", "origin", branch)
	fetchCmd.Dir = workspace
	if err := runGit(fetchCmd, "Git fetch"); err != nil {
		return err
	}
	if !pipelineCommitExists(ctx, workspace, expectedCommit) {
		logger.Info("目标提交不在当前缓存中，正在补全分支历史...")
		args := []string{"fetch", "origin", branch}
		if pipelineRepositoryIsShallow(ctx, workspace) {
			args = []string{"fetch", "--unshallow", "origin", branch}
		}
		historyCmd := exec.CommandContext(ctx, "git", args...)
		historyCmd.Dir = workspace
		if err := runGit(historyCmd, "Git fetch history"); err != nil {
			return err
		}
	}
	return checkoutPipelineCommitFromRef(ctx, logger, workspace, expectedCommit, "FETCH_HEAD", runGit)
}

func checkoutPipelineCommitFromRef(ctx context.Context, logger *PipelineLogger, workspace, expectedCommit, branchRef string, runGit pipelineGitRunner) error {
	if !pipelineCommitExists(ctx, workspace, expectedCommit) {
		return fmt.Errorf("expected commit %s is not reachable from the configured branch", expectedCommit)
	}
	if !pipelineCommitIsAncestor(ctx, workspace, expectedCommit, branchRef) {
		return fmt.Errorf("expected commit %s is not reachable from the configured branch", expectedCommit)
	}
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "--detach", expectedCommit)
	checkoutCmd.Dir = workspace
	if err := runGit(checkoutCmd, "Git checkout expected commit"); err != nil {
		return err
	}
	if err := verifyPipelineExpectedCommit(ctx, workspace, expectedCommit); err != nil {
		return err
	}
	logger.Info("已检出锁定提交: %s", expectedCommit)
	return nil
}

func pipelineCommitIsAncestor(ctx context.Context, workspace, commit, branchRef string) bool {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", commit, branchRef)
	cmd.Dir = workspace
	return cmd.Run() == nil
}

func verifyPipelineExpectedCommit(ctx context.Context, workspace, expectedCommit string) error {
	if expectedCommit == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = workspace
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot verify pipeline commit: %w", err)
	}
	actualCommit := strings.TrimSpace(string(output))
	if !pipelineCommitEqual(actualCommit, expectedCommit) {
		return fmt.Errorf("pipeline commit changed: expected %s, got %s", expectedCommit, actualCommit)
	}
	return nil
}

func pipelineCommitExists(ctx context.Context, workspace, commit string) bool {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-e", commit+"^{commit}")
	cmd.Dir = workspace
	return cmd.Run() == nil
}

func pipelineRepositoryIsShallow(ctx context.Context, workspace string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-shallow-repository")
	cmd.Dir = workspace
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func pipelineCommitEqual(actual, expected string) bool {
	return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected))
}
