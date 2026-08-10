package api

import (
	"context"
	"errors"
	"path/filepath"
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

const (
	codePushPending = "pending"
	codePushPushed  = "pushed"
	codePushFailed  = "failed"
)

var errCodePushRemoteAdvanced = errors.New("远端分支已在交付后更新，请先同步并重新交付")
var errCodeDeliveryBranchAdvanced = errors.New("交付分支已被外部更新，请确认后重新交付")

type codeRepositoryPushResult struct {
	RepositoryID   string `json:"repositoryId"`
	RepositoryName string `json:"repositoryName"`
	Status         string `json:"status"`
	Remote         string `json:"remote,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Commit         string `json:"commit,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	// 本地主仓是否已同步到交付提交。未同步不影响推送，仅用于提示用户手动同步。
	LocalSynced      bool   `json:"localSynced"`
	LocalSyncError   string `json:"localSyncError,omitempty"`
	LocalSyncCommand string `json:"localSyncCommand,omitempty"`
	Ready            bool   `json:"ready"`
}

type codePushResult struct {
	Available    bool                       `json:"available"`
	Status       string                     `json:"status"`
	Repositories []codeRepositoryPushResult `json:"repositories"`
}

func GetCodeDeliveryPushStatus(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateCodeDeliveryPushSession(session, claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := loadCodeDeliveryPushStatus(session)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func PushCodeSessionDelivery(c fiber.Ctx) error {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if !req.Confirm {
		return c.JSON(e.Fail(errors.New("推送操作需要明确确认")))
	}
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, err := getCodeDeliverySessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validateCodeDeliveryPushSession(session, claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	lease, err := codeExecutions.acquireSession(context.Background(), session, codeExecutionDelivery, false)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer lease.Release()
	result, err := pushCodeSessionDelivery(session)
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "git_push", "failed", "delivery", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "git_push", "success", "delivery", result.Status, c.IP(), startedAt, codeAuditMeta{"repositories": len(result.Repositories)})
	return c.JSON(e.Succ(result))
}

func loadCodeDeliveryPushStatus(session *model.AIDevSession) (codePushResult, error) {
	if hasCodeMultiRepositoryDelivery(session.ID) {
		return loadCodeMultiRepositoryPushStatus(session)
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return codePushResult{Status: "unavailable", Repositories: []codeRepositoryPushResult{}}, nil
		}
		return codePushResult{}, err
	}
	result := codeRepositoryPushResult{
		RepositoryID: "session", RepositoryName: filepath.Base(delivery.SourceWorkDir),
		Status: delivery.PushStatus, Remote: delivery.RemoteName,
		Branch: codeDeliveryPushTarget(
			codeProjectDeliveryMode(delivery.ProjectID),
			delivery.SessionID, delivery.RemoteBranch, delivery.TargetBranch,
		),
		Commit: delivery.PushedCommit, ErrorMessage: delivery.PushError,
		LocalSynced: delivery.SourceAppliedAt != nil, LocalSyncError: delivery.LocalSyncError,
		LocalSyncCommand: codeDeliveryLocalSyncCommand(delivery.SourceWorkDir, delivery.MergeCommit),
		Ready:            strings.TrimSpace(delivery.MergeCommit) != "",
	}
	return summarizeCodePushResults([]codeRepositoryPushResult{result}), nil
}

func validateCodeDeliveryPushSession(session *model.AIDevSession, claims *token.CustomClaims) error {
	if !hasCodeMultiRepositoryDelivery(session.ID) {
		return nil
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil || len(sourceDirs) == 0 {
		return errors.New("项目源目录不可用")
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return errors.New("会话多仓库交付记录不可用")
	}
	for _, repository := range repositories {
		resolved, err := filepath.EvalSymlinks(filepath.Clean(repository.SourceDir))
		if err != nil || !repositoryWithinSourceDirs(resolved, sourceDirs) {
			return errors.New("会话仓库与项目配置不一致")
		}
		if err := validateAIProjectWorkDirForClaims(resolved, claims); err != nil {
			return err
		}
	}
	return nil
}

func deliveryRemoteBranch(remoteBranch, targetBranch string) string {
	if strings.TrimSpace(remoteBranch) != "" {
		return remoteBranch
	}
	return targetBranch
}

func summarizeCodePushResults(repositories []codeRepositoryPushResult) codePushResult {
	result := codePushResult{Status: codePushPending, Repositories: repositories}
	if len(repositories) == 0 {
		result.Status = "unavailable"
		return result
	}
	result.Available = true
	allPushed := true
	for _, repository := range repositories {
		if repository.Remote == "" || repository.Branch == "" || !repository.Ready {
			result.Available = false
		}
		if repository.Status == codePushFailed {
			result.Status = codePushFailed
		}
		if repository.Status != codePushPushed {
			allPushed = false
		}
	}
	if allPushed {
		result.Status = codePushPushed
	}
	return result
}

func pushCodeSessionDelivery(session *model.AIDevSession) (codePushResult, error) {
	if hasCodeMultiRepositoryDelivery(session.ID) {
		return pushCodeMultiRepositoryDelivery(session)
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return codePushResult{}, errors.New("会话尚未完成本地合并交付")
	}
	mode := codeProjectDeliveryMode(delivery.ProjectID)
	result, err := pushCodeDeliveryRepositoryWithCredential(codeDeliveryPushRequest{
		SourceDir:  delivery.SourceWorkDir,
		RemoteName: delivery.RemoteName,
		RemoteBranch: codeDeliveryPushTarget(
			mode, delivery.SessionID, delivery.RemoteBranch, delivery.TargetBranch,
		),
		MergeCommit: delivery.MergeCommit, PushStatus: delivery.PushStatus,
		Isolated: codeDeliveryPushIsolated(mode), LastPushedCommit: delivery.PushedCommit,
		CredentialID: codeProjectGitCredentialID(delivery.ProjectID),
	}, nil)
	result.RepositoryID, result.RepositoryName = "session", filepath.Base(delivery.SourceWorkDir)
	updates := map[string]any{"push_status": result.Status, "push_error": result.ErrorMessage}
	if result.Status == codePushPushed {
		updates["pushed_commit"], updates["pushed_at"] = result.Commit, time.Now()
	}
	if updateErr := global.DB.Model(&delivery).Updates(updates).Error; updateErr != nil {
		return codePushResult{}, updateErr
	}
	if err != nil {
		return summarizeCodePushResults([]codeRepositoryPushResult{result}), err
	}
	return summarizeCodePushResults([]codeRepositoryPushResult{result}), nil
}

// codeDeliveryPushRequest 描述一次单仓交付推送。
// Isolated 为真时推送目标是本会话独占的交付分支：远端不存在属正常，
// 且允许用 --force-with-lease 覆盖 LastPushedCommit（自己上次推的提交）。
type codeDeliveryPushRequest struct {
	SourceDir        string
	RemoteName       string
	RemoteBranch     string
	MergeCommit      string
	PushStatus       string
	Isolated         bool
	LastPushedCommit string
	CredentialID     uint
}

func pushCodeDeliveryRepository(sourceDir, remoteName, remoteBranch, mergeCommit, pushStatus string) (codeRepositoryPushResult, error) {
	return pushCodeDeliveryRepositoryWithCredential(codeDeliveryPushRequest{
		SourceDir: sourceDir, RemoteName: remoteName, RemoteBranch: remoteBranch,
		MergeCommit: mergeCommit, PushStatus: pushStatus,
	}, nil)
}

func pushCodeDeliveryRepositoryWithCredential(request codeDeliveryPushRequest, report codeDeliveryProgressReporter) (codeRepositoryPushResult, error) {
	sourceDir, remoteName := request.SourceDir, request.RemoteName
	remoteBranch, mergeCommit := request.RemoteBranch, request.MergeCommit
	credentialID := request.CredentialID
	result := codeRepositoryPushResult{Status: codePushPending, Remote: remoteName, Branch: remoteBranch, Commit: mergeCommit, Ready: mergeCommit != ""}
	if request.PushStatus == codePushPushed {
		result.Status = codePushPushed
		return result, nil
	}
	if remoteName == "" || remoteBranch == "" {
		return failedCodePushResult(result, errors.New("仓库没有可用的远端跟踪分支"))
	}
	// 推送的是交付提交本身（commit-ish），不要求本地目标分支已经指向它。
	if err := verifyCodeDeliveryCommitReachable(sourceDir, mergeCommit, "本次交付"); err != nil {
		return failedCodePushResult(result, err)
	}
	if _, err := fetchCodeRepositoryWithCredential(sourceDir, remoteName, credentialID); err != nil {
		return failedCodePushResult(result, err)
	}
	remoteRef := "refs/remotes/" + remoteName + "/" + remoteBranch
	currentRemoteCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
	if err == nil && currentRemoteCommit == mergeCommit {
		result.Status = codePushPushed
		return result, nil
	}
	forceLease := ""
	switch {
	case err != nil:
		// 独占的会话交付分支首次推送时远端还不存在，这是正常的。
		if !request.Isolated {
			return failedCodePushResult(result, errCodePushRemoteAdvanced)
		}
	case codeRemoteCommitCanFastForward(sourceDir, currentRemoteCommit, mergeCommit):
	case !request.Isolated:
		return failedCodePushResult(result, errCodePushRemoteAdvanced)
	case strings.TrimSpace(request.LastPushedCommit) != strings.TrimSpace(currentRemoteCommit):
		return failedCodePushResult(result, errCodeDeliveryBranchAdvanced)
	default:
		forceLease = strings.TrimSpace(currentRemoteCommit)
	}
	args := []string{"-c", "credential.interactive=never", "push"}
	if forceLease != "" {
		args = append(args, "--force-with-lease=refs/heads/"+remoteBranch+":"+forceLease)
	}
	args = append(args, "--", remoteName, mergeCommit+":refs/heads/"+remoteBranch)
	if _, err := runCodeGitWithCredential(sourceDir, codeGitFetchTimeout, credentialID, args...); err != nil {
		pushErr := err
		if _, fetchErr := fetchCodeRepositoryWithCredential(sourceDir, remoteName, credentialID); fetchErr == nil {
			latestRemoteCommit, resolveErr := runCodeGit(sourceDir, "rev-parse", remoteRef)
			// 推送命令报错但远端已经是交付提交：认定推送已生效，避免重复推送。
			if resolveErr == nil && strings.TrimSpace(latestRemoteCommit) == mergeCommit {
				result.Status, result.Commit = codePushPushed, mergeCommit
				return result, nil
			}
			// 独占分支本来就允许覆盖，无法快进不构成「远端已推进」。
			if resolveErr == nil && !request.Isolated &&
				!codeRemoteCommitCanFastForward(sourceDir, latestRemoteCommit, mergeCommit) {
				return failedCodePushResult(result, errCodePushRemoteAdvanced)
			}
		}
		return failedCodePushResult(result, pushErr)
	}
	if report != nil {
		report(codeDeliveryStageVerifying, 85)
	}
	if _, err := fetchCodeRepositoryWithCredential(sourceDir, remoteName, credentialID); err != nil {
		return failedCodePushResult(result, err)
	}
	pushedCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
	if err != nil || pushedCommit != mergeCommit {
		return failedCodePushResult(result, errors.New("推送后远端提交核验失败"))
	}
	result.Status, result.Commit = codePushPushed, mergeCommit
	return result, nil
}

func codeRemoteCommitCanFastForward(sourceDir, remoteCommit, mergeCommit string) bool {
	remoteCommit, mergeCommit = strings.TrimSpace(remoteCommit), strings.TrimSpace(mergeCommit)
	if remoteCommit == "" || mergeCommit == "" {
		return false
	}
	_, err := runCodeGit(sourceDir, "merge-base", "--is-ancestor", remoteCommit, mergeCommit)
	return err == nil
}

func failedCodePushResult(result codeRepositoryPushResult, err error) (codeRepositoryPushResult, error) {
	result.Status, result.ErrorMessage = codePushFailed, strings.TrimSpace(err.Error())
	return result, err
}
