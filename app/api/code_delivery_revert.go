package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

const (
	codeRevertStatusReverted = "reverted"
	codeRevertStatusConflict = "conflict"
	codeRevertStatusSkipped  = "skipped"
)

// 撤销一律用反向提交，不用 reset。
//
// 默认交付模式是 direct 直推目标分支，交付提交大概率已经在远端，
// 别人可能已经拉下去了；reset 改写历史会让所有下游仓库炸掉。
// 反向提交的代价是历史里留下两条记录，但那是可审计的事实而不是噪音。
type codeDeliveryRevertRequest struct {
	SourceDir    string
	TargetBranch string
	MergeCommit  string
	RemoteName   string
	RemoteBranch string
	PushStatus   string
	CredentialID uint
	Label        string
}

type codeRepositoryRevertResult struct {
	RepositoryName string   `json:"repositoryName,omitempty"`
	Status         string   `json:"status"`
	RevertCommit   string   `json:"revertCommit,omitempty"`
	ConflictFiles  []string `json:"conflictFiles,omitempty"`
	ErrorMessage   string   `json:"errorMessage,omitempty"`
	Pushed         bool     `json:"pushed"`
}

// revertCodeDeliveryInRepository 在单个仓库里撤销一次交付。
//
// 全程在临时 worktree 里做，不碰用户可能正开着的工作区：
// 撤销失败时用户的当前分支和未提交改动必须原封不动。
func revertCodeDeliveryInRepository(
	request codeDeliveryRevertRequest,
	session *model.AIDevSession,
) (codeRepositoryRevertResult, error) {
	result := codeRepositoryRevertResult{RepositoryName: request.Label}
	if err := verifyCodeDeliveryCommitExists(request.SourceDir, request.MergeCommit, "本次交付"); err != nil {
		return result, err
	}
	targetBranch := strings.TrimSpace(request.TargetBranch)
	if targetBranch == "" {
		return result, errors.New("交付目标分支不可用")
	}
	targetRef := "refs/heads/" + targetBranch
	targetCommit, err := runCodeGit(request.SourceDir, "rev-parse", targetRef)
	if err != nil {
		return result, errors.New("本地目标分支 " + targetBranch + " 不可用")
	}
	targetCommit = strings.TrimSpace(targetCommit)

	// 交付提交不在目标分支上，说明这次交付根本没落到本地（或分支已被改写），
	// 此时撤销无从谈起，也不该凭空造一个反向提交。
	if _, err := runCodeGit(request.SourceDir, "merge-base", "--is-ancestor", request.MergeCommit, targetCommit); err != nil {
		result.Status = codeRevertStatusSkipped
		result.ErrorMessage = "该交付不在目标分支 " + targetBranch + " 上，无需撤销"
		return result, nil
	}
	return applyCodeDeliveryRevert(request, session, targetRef, targetCommit, result)
}

func applyCodeDeliveryRevert(
	request codeDeliveryRevertRequest,
	session *model.AIDevSession,
	targetRef, targetCommit string,
	result codeRepositoryRevertResult,
) (codeRepositoryRevertResult, error) {
	root, err := os.MkdirTemp("", "gopanel-delivery-revert-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(root)
	workDir := filepath.Join(root, "worktree")
	if _, err := runCodeGit(request.SourceDir, "worktree", "add", "--detach", workDir, targetCommit); err != nil {
		return result, err
	}
	defer func() {
		_, _ = runCodeGit(workDir, "revert", "--abort")
		_, _ = runCodeGit(request.SourceDir, "worktree", "remove", "--force", workDir)
	}()

	revertArgs, err := codeRevertArgsForCommit(workDir, request.MergeCommit)
	if err != nil {
		return result, err
	}
	if _, err := runCodeGit(workDir, revertArgs...); err != nil {
		if conflicts := codeGitConflictFiles(workDir); len(conflicts) > 0 {
			result.Status, result.ConflictFiles = codeRevertStatusConflict, conflicts
			result.ErrorMessage = "撤销与目标分支上的后续改动冲突，需要人工处理"
			return result, nil
		}
		return result, fmt.Errorf("撤销交付失败：%w", err)
	}
	staged, err := runCodeGit(workDir, "diff", "--cached", "--name-only")
	if err != nil {
		return result, err
	}
	// 反向提交产不出任何改动，说明这次交付的内容已经被撤掉了。
	// 这时再提交一个空提交毫无意义，更糟的是会掩盖「已撤销」这个事实。
	if strings.TrimSpace(staged) == "" {
		result.Status = codeRevertStatusSkipped
		result.ErrorMessage = "该交付的改动已经不在目标分支上，无需撤销"
		return result, nil
	}
	message := codeDeliveryRevertMessage(session, request.Label, request.MergeCommit)
	if _, err := runCodeGit(workDir, codeGitAuthoredArgs(
		"-c", "commit.gpgsign=false", "commit", "-m", message,
	)...); err != nil {
		return result, err
	}
	revertCommit, err := runCodeGit(workDir, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	revertCommit = strings.TrimSpace(revertCommit)
	if err := advanceCodeTargetBranch(request.SourceDir, targetRef, targetCommit, revertCommit); err != nil {
		return result, err
	}
	result.Status, result.RevertCommit = codeRevertStatusReverted, revertCommit
	return result, nil
}

// codeRevertArgsForCommit 决定要不要带 -m 1。
//
// 交付提交通常是 --no-ff 产生的合并提交，撤销合并必须指明主线；
// 但目标分支能快进时交付提交就是普通提交，这时带 -m 会被 Git 直接拒绝。
func codeRevertArgsForCommit(workDir, commit string) ([]string, error) {
	parents, err := runCodeGit(workDir, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return nil, err
	}
	// 输出形如 "<commit> <parent1> [<parent2> ...]"，减去自身即父提交数。
	if len(strings.Fields(strings.TrimSpace(parents)))-1 > 1 {
		return []string{"revert", "--no-commit", "-m", "1", commit}, nil
	}
	return []string{"revert", "--no-commit", commit}, nil
}

// advanceCodeTargetBranch 把目标分支推进到撤销提交。
//
// 分支可能正被某个 worktree 检出，直接 update-ref 会让那个工作区
// 的索引和 HEAD 对不上；检出时走 --ff-only 让 Git 自己更新索引。
// 没被检出才用 update-ref，并带上期望旧值做 CAS——期间分支被别人
// 推进的话必须失败而不是覆盖。
func advanceCodeTargetBranch(sourceDir, targetRef, expectedCommit, newCommit string) error {
	checkoutDir, err := codeTargetBranchCheckoutDir(sourceDir, targetRef)
	if err != nil {
		return err
	}
	if checkoutDir != "" {
		status, statusErr := runCodeGit(checkoutDir, "status", "--porcelain")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			return errors.New("目标分支所在工作区有未提交改动，请处理后重试")
		}
		if _, err := runCodeGit(checkoutDir, "merge", "--ff-only", newCommit); err != nil {
			return fmt.Errorf("目标分支无法更新到撤销提交：%w", err)
		}
		return nil
	}
	if _, err := runCodeGit(sourceDir, "update-ref", targetRef, newCommit, expectedCommit); err != nil {
		return errors.New("目标分支在操作期间发生变化，请刷新后重试")
	}
	return nil
}

// codeDeliveryRevertMessage 生成撤销提交的信息。
// 「This reverts commit <sha>.」是 Git 自己的格式，托管平台和
// git log --grep 都认它，所以照写而不是自创一套。
func codeDeliveryRevertMessage(session *model.AIDevSession, repositoryName, mergeCommit string) string {
	subject := "revert: " + codeDeliverySubject(session)
	if session != nil {
		scope := fmt.Sprintf("session #%d", session.ID)
		if name := codeTrailerValue(repositoryName); name != "" {
			scope += ", " + name
		}
		subject += " (" + scope + ")"
	}
	body := subject + "\n\nThis reverts commit " + mergeCommit + "."
	return codeAppendCommitTrailers(body, session)
}

// pushCodeDeliveryRevert 把撤销提交推到远端。
// 只有原交付确实推送过才需要——没推过的交付撤销后远端本就没有它。
func pushCodeDeliveryRevert(request codeDeliveryRevertRequest, revertCommit string) (bool, error) {
	if request.PushStatus != codePushPushed {
		return false, nil
	}
	pushResult, err := pushCodeDeliveryRepositoryWithCredential(codeDeliveryPushRequest{
		SourceDir:    request.SourceDir,
		RemoteName:   request.RemoteName,
		RemoteBranch: deliveryRemoteBranch(request.RemoteBranch, request.TargetBranch),
		MergeCommit:  revertCommit,
		CredentialID: request.CredentialID,
	}, nil)
	if err != nil {
		return false, err
	}
	if pushResult.Status != codePushPushed {
		return false, errors.New(pushResult.ErrorMessage)
	}
	return true, nil
}
