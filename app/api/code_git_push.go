package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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

type codeRepositoryPushResult struct {
	RepositoryID   string `json:"repositoryId"`
	RepositoryName string `json:"repositoryName"`
	Status         string `json:"status"`
	Remote         string `json:"remote,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Commit         string `json:"commit,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	Ready          bool   `json:"ready"`
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
		repositories, err := loadCodeSessionRepositories(session.ID)
		if err != nil {
			return codePushResult{}, err
		}
		results := make([]codeRepositoryPushResult, 0, len(repositories))
		for _, repository := range repositories {
			remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
			results = append(results, codeRepositoryPushResult{
				RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
				Status: repository.PushStatus, Remote: repository.RemoteName,
				Branch: remoteBranch, Commit: repository.PushedCommit, ErrorMessage: repository.PushError,
				Ready: repository.MergeCommit != "" && repository.RemoteCommit != "",
			})
		}
		return summarizeCodePushResults(results), nil
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
		Branch: deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch),
		Commit: delivery.PushedCommit, ErrorMessage: delivery.PushError,
		Ready: delivery.MergeCommit != "" && delivery.RemoteCommit != "",
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
		return pushCodeMultiRepositoryDelivery(session.ID)
	}
	var delivery model.AICodeDelivery
	if err := global.DB.Where("session_id = ?", session.ID).First(&delivery).Error; err != nil {
		return codePushResult{}, errors.New("会话尚未完成本地合并交付")
	}
	result, err := pushCodeDeliveryRepository(
		delivery.SourceWorkDir, delivery.TargetBranch, delivery.RemoteName,
		deliveryRemoteBranch(delivery.RemoteBranch, delivery.TargetBranch),
		delivery.RemoteCommit, delivery.MergeCommit, delivery.PushStatus,
	)
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

func pushCodeMultiRepositoryDelivery(sessionID uint) (codePushResult, error) {
	repositories, err := loadCodeSessionRepositories(sessionID)
	if err != nil || len(repositories) == 0 {
		return codePushResult{}, errors.New("会话多仓库交付记录不可用")
	}
	sort.SliceStable(repositories, func(i, j int) bool { return repositories[i].LinkName < repositories[j].LinkName })
	results := make([]codeRepositoryPushResult, 0, len(repositories))
	for index := range repositories {
		repository := &repositories[index]
		result, pushErr := pushCodeDeliveryRepository(
			repository.SourceDir, repository.TargetBranch, repository.RemoteName,
			deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
			repository.RemoteCommit, repository.MergeCommit, repository.PushStatus,
		)
		result.RepositoryID, result.RepositoryName = codeSessionRepositoryID(repository.ID), repository.LinkName
		updates := map[string]any{"push_status": result.Status, "push_error": result.ErrorMessage}
		if result.Status == codePushPushed {
			updates["pushed_commit"], updates["pushed_at"] = result.Commit, time.Now()
		}
		if err := global.DB.Model(repository).Updates(updates).Error; err != nil {
			return codePushResult{Status: codePushFailed, Repositories: results}, err
		}
		results = append(results, result)
		if pushErr != nil {
			return summarizeCodePushResults(results), fmt.Errorf("仓库 %s 推送失败：%w", repository.LinkName, pushErr)
		}
	}
	return summarizeCodePushResults(results), nil
}

func pushCodeDeliveryRepository(sourceDir, targetBranch, remoteName, remoteBranch, remoteCommit, mergeCommit, pushStatus string) (codeRepositoryPushResult, error) {
	result := codeRepositoryPushResult{Status: codePushPending, Remote: remoteName, Branch: remoteBranch, Commit: mergeCommit, Ready: mergeCommit != ""}
	if pushStatus == codePushPushed {
		result.Status = codePushPushed
		return result, nil
	}
	if remoteName == "" || remoteBranch == "" {
		return failedCodePushResult(result, errors.New("仓库没有可用的远端跟踪分支"))
	}
	if mergeCommit == "" {
		return failedCodePushResult(result, errors.New("仓库尚未完成本地合并交付"))
	}
	localCommit, err := runCodeGit(sourceDir, "rev-parse", "refs/heads/"+targetBranch)
	if err != nil || localCommit != mergeCommit {
		return failedCodePushResult(result, errors.New("目标分支已在交付后发生变化，请重新创建会话交付"))
	}
	if _, err := fetchCodeRepository(sourceDir, remoteName); err != nil {
		return failedCodePushResult(result, err)
	}
	remoteRef := "refs/remotes/" + remoteName + "/" + remoteBranch
	currentRemoteCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
	if err == nil && currentRemoteCommit == mergeCommit {
		result.Status = codePushPushed
		return result, nil
	}
	if err != nil || currentRemoteCommit != remoteCommit {
		return failedCodePushResult(result, errors.New("远端分支已在交付后更新，请先同步并重新交付"))
	}
	if _, err := runCodeGitWithTimeout(
		sourceDir, codeGitFetchTimeout,
		"-c", "credential.interactive=never", "push", "--", remoteName,
		mergeCommit+":refs/heads/"+remoteBranch,
	); err != nil {
		return failedCodePushResult(result, err)
	}
	if _, err := fetchCodeRepository(sourceDir, remoteName); err != nil {
		return failedCodePushResult(result, err)
	}
	pushedCommit, err := runCodeGit(sourceDir, "rev-parse", remoteRef)
	if err != nil || pushedCommit != mergeCommit {
		return failedCodePushResult(result, errors.New("推送后远端提交核验失败"))
	}
	result.Status, result.Commit = codePushPushed, mergeCommit
	return result, nil
}

func failedCodePushResult(result codeRepositoryPushResult, err error) (codeRepositoryPushResult, error) {
	result.Status, result.ErrorMessage = codePushFailed, strings.TrimSpace(err.Error())
	return result, err
}
