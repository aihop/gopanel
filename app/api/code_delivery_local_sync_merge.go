package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func syncCodeDeliveryTargetOnDemand(sourceDir, targetBranch, deliveryCommit string) error {
	sourceDir, targetBranch, deliveryCommit = strings.TrimSpace(sourceDir), strings.TrimSpace(targetBranch), strings.TrimSpace(deliveryCommit)
	if sourceDir == "" || targetBranch == "" || deliveryCommit == "" {
		return errors.New("本地同步所需的仓库信息不完整")
	}
	if err := verifyCodeDeliveryCommitExists(sourceDir, deliveryCommit, "本次交付"); err != nil {
		return err
	}
	targetRef := "refs/heads/" + targetBranch
	targetCommit, err := runCodeGit(sourceDir, "rev-parse", targetRef)
	if err != nil {
		return errors.New("本地主仓目标分支不可用")
	}
	targetCommit = strings.TrimSpace(targetCommit)
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", deliveryCommit, targetCommit); err == nil {
		return nil
	}
	checkoutDir, err := codeTargetBranchCheckoutDir(sourceDir, targetRef)
	if err != nil {
		return err
	}
	if checkoutDir != "" {
		status, statusErr := runCodeGit(checkoutDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			return errors.New("本地主仓存在未提交改动，请处理后重试")
		}
	}
	resultCommit := deliveryCommit
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", targetCommit, deliveryCommit); err != nil {
		resultCommit, err = createCodeDeliveryLocalMergeCommit(sourceDir, targetCommit, deliveryCommit, targetBranch)
		if err != nil {
			return err
		}
	}
	if checkoutDir != "" {
		if _, err := runCodeGit(checkoutDir, "merge", "--ff-only", resultCommit); err != nil {
			return fmt.Errorf("本地主仓无法更新到合并结果：%w", err)
		}
	} else if _, err := runCodeGit(sourceDir, "update-ref", targetRef, resultCommit, targetCommit); err != nil {
		return errors.New("目标分支在操作期间发生变化，请刷新后重试")
	}
	updated, err := runCodeGit(sourceDir, "rev-parse", targetRef)
	if err != nil {
		return errors.New("本地主仓合并结果核验失败")
	}
	if _, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", deliveryCommit, strings.TrimSpace(updated)); err != nil {
		return errors.New("本地主仓合并结果核验失败")
	}
	return nil
}

func codeTargetBranchCheckoutDir(sourceDir, targetRef string) (string, error) {
	output, err := runCodeGit(sourceDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	workDir := ""
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			workDir = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		case strings.TrimSpace(line) == "branch "+targetRef:
			return workDir, nil
		}
	}
	return "", nil
}

func createCodeDeliveryLocalMergeCommit(sourceDir, targetCommit, deliveryCommit, targetBranch string) (string, error) {
	root, err := os.MkdirTemp("", "gopanel-local-sync-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(root)
	workDir := filepath.Join(root, "worktree")
	if _, err := runCodeGit(sourceDir, "worktree", "add", "--detach", workDir, targetCommit); err != nil {
		return "", err
	}
	defer func() {
		_, _ = runCodeGit(workDir, "merge", "--abort")
		_, _ = runCodeGit(sourceDir, "worktree", "remove", "--force", workDir)
	}()
	if _, err := runCodeGit(
		workDir, codeGitAuthoredArgs(
			"-c", "commit.gpgsign=false", "merge", "--no-edit", deliveryCommit,
		)...,
	); err != nil {
		conflicts := codeGitConflictFiles(workDir)
		if len(conflicts) > 0 {
			return "", fmt.Errorf("目标分支 %s 与交付提交存在冲突：%s", targetBranch, strings.Join(conflicts, ", "))
		}
		return "", err
	}
	commit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	return strings.TrimSpace(commit), err
}
