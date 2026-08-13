package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

func (s *PipelineService) stepClone(ctx context.Context, logger *PipelineLogger, pipeline *model.Pipeline, workspace, sinceCommit, expectedCommit string) (string, string, error) {
	logger.Info("准备代码拉取目录...")
	_ = os.MkdirAll(workspace, 0755)
	repoURL := buildPipelineRepoURL(pipeline.RepoUrl, pipeline.AuthType, pipeline.AuthData)
	runGitCommand := func(cmd *exec.Cmd, action string) error {
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new")
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(&outBuf, newLogWriter(logger, false))
		cmd.Stderr = io.MultiWriter(&errBuf, newLogWriter(logger, true))
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("%s 失败: %v", action, err)
			return err
		}
		return nil
	}
	gitDir := filepath.Join(workspace, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		logger.Info("检测到本地缓存，正在更新远端代码 (分支: %s)...", pipeline.Branch)
		remoteCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", repoURL)
		remoteCmd.Dir = workspace
		if err := runGitCommand(remoteCmd, "Git remote"); err != nil {
			return "", "", err
		}
		if expectedCommit == "" {
			checkoutCmd := exec.CommandContext(ctx, "git", "checkout", pipeline.Branch)
			checkoutCmd.Dir = workspace
			if err := runGitCommand(checkoutCmd, "Git checkout"); err != nil {
				return "", "", err
			}
			pullCmd := exec.CommandContext(ctx, "git", "pull", "origin", pipeline.Branch)
			pullCmd.Dir = workspace
			if err := runGitCommand(pullCmd, "Git pull"); err != nil {
				return "", "", err
			}
		} else if err := fetchAndCheckoutPipelineCommit(ctx, logger, workspace, pipeline.Branch, expectedCommit, runGitCommand); err != nil {
			return "", "", err
		}
	} else {
		logger.Info("首次执行或缓存丢失，正在执行 git clone (分支: %s)...", pipeline.Branch)
		cloneArgs := []string{"clone", "-b", pipeline.Branch, "--single-branch"}
		if expectedCommit == "" {
			cloneArgs = append(cloneArgs, "--depth", "1")
		}
		cloneArgs = append(cloneArgs, repoURL, workspace)
		cloneCmd := exec.CommandContext(ctx, "git", cloneArgs...)
		if err := runGitCommand(cloneCmd, "Git clone"); err != nil {
			return "", "", err
		}
		if expectedCommit != "" {
			if err := checkoutPipelineCommitFromRef(ctx, logger, workspace, expectedCommit, "origin/"+pipeline.Branch, runGitCommand); err != nil {
				return "", "", err
			}
		}
	}
	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	hashCmd.Dir = workspace
	hashBytes, err := hashCmd.Output()
	if err != nil {
		logger.Error("无法读取检出后的 Commit: %v", err)
		return "", "", err
	}
	commitHash := strings.TrimSpace(string(hashBytes))
	if expectedCommit != "" && !pipelineCommitEqual(commitHash, expectedCommit) {
		return "", "", fmt.Errorf("checked out commit %s does not match expected commit %s", commitHash, expectedCommit)
	}
	logger.Info("代码拉取成功, Commit Hash: %s", commitHash)
	changelog := collectPipelineChangelog(ctx, logger, workspace, sinceCommit)
	return commitHash, changelog, nil
}
