package api

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

type codeRepositoryDeliveryResult struct {
	RepositoryID    string     `json:"repositoryId"`
	RepositoryName  string     `json:"repositoryName"`
	Status          string     `json:"status"`
	Branch          string     `json:"branch"`
	TargetBranch    string     `json:"targetBranch"`
	Remote          string     `json:"remote,omitempty"`
	RemoteBranch    string     `json:"remoteBranch,omitempty"`
	Commit          string     `json:"commit,omitempty"`
	PushStatus      string     `json:"pushStatus"`
	PushedCommit    string     `json:"pushedCommit,omitempty"`
	SourceAppliedAt *time.Time `json:"sourceAppliedAt,omitempty"`
	ErrorMessage    string     `json:"errorMessage,omitempty"`
	ConflictFiles   []string   `json:"conflictFiles,omitempty"`
}

func validateCodeMultiWorktreeDeliverySession(session *model.AIDevSession, claims *token.CustomClaims) error {
	if session == nil || session.IsolationMode != codeIsolationMultiWorktree {
		return errors.New("当前会话未启用多仓库 Worktree 隔离")
	}
	if !isManagedAISessionWorkDir(session.WorkDir, session.UserID) || !isAISessionWorkspaceDirectory(session.WorkDir) {
		return errors.New("会话多仓库 Worktree 不在 GoPanel 管理目录中")
	}
	sourceDirs, err := getAISessionSourceDirs(session.ProjectID, claims)
	if err != nil {
		return err
	}
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return errors.New("会话多仓库 Worktree 元数据不完整")
	}
	for _, repository := range repositories {
		resolved, resolveErr := filepath.EvalSymlinks(filepath.Clean(repository.SourceDir))
		if resolveErr != nil {
			return errors.New("会话仓库源目录不可用")
		}
		if !repositoryWithinSourceDirs(resolved, sourceDirs) || !isPathInside(repository.WorktreeDir, session.WorkDir) {
			return errors.New("会话仓库与项目配置不一致")
		}
		if repository.Status != codeDeliveryCompleted {
			if err := validateCodeSessionRepositoryWorktree(session, &repository); err != nil {
				return err
			}
		}
		if err := validateAIProjectWorkDirForClaims(repository.SourceDir, claims); err != nil {
			return err
		}
	}
	return nil
}

func hasCodeMultiRepositoryDelivery(sessionID uint) bool {
	var count int64
	return global.DB.Model(&model.AIDevSessionRepository{}).Where("session_id = ?", sessionID).Count(&count).Error == nil && count > 0
}

func codeMultiRepositoryWorkspaceDir(session *model.AIDevSession, repositories []model.AIDevSessionRepository) (string, error) {
	if session == nil || len(repositories) == 0 {
		return "", errors.New("会话多仓库 Worktree 元数据不可用")
	}
	workspaceDir := filepath.Dir(filepath.Clean(repositories[0].WorktreeDir))
	if workspaceDir != filepath.Clean(aiSessionWorktreeDir(session.UserID, session.ID)) {
		return "", errors.New("会话多仓库 Worktree 目录与会话编号不一致")
	}
	for _, repository := range repositories {
		if filepath.Dir(filepath.Clean(repository.WorktreeDir)) != workspaceDir {
			return "", errors.New("会话仓库 Worktree 目录不一致")
		}
	}
	return workspaceDir, nil
}

func commitCodeSessionRepository(session *model.AIDevSession, repositoryID, message string) (codeGitDeliveryResult, error) {
	message, err := validateCodeGitCommitMessage(message)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	repository, err := codeSessionRepositoryByCodeID(session.ID, strings.TrimSpace(repositoryID))
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := validateCodeSessionRepositoryWorktree(session, repository); err != nil {
		return codeGitDeliveryResult{}, err
	}
	staged, err := runCodeGit(repository.WorktreeDir, "diff", "--cached", "--name-only")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if strings.TrimSpace(staged) == "" {
		return codeGitDeliveryResult{}, errors.New("当前仓库暂存区没有可提交的变更")
	}
	if err := validateCodeGitStagedChanges(repository.WorktreeDir); err != nil {
		return codeGitDeliveryResult{}, err
	}
	if _, err := runCodeGit(repository.WorktreeDir, "-c", "user.name=GoPanel Code", "-c", "user.email=code@gopanel.local", "commit", "-m", message); err != nil {
		return codeGitDeliveryResult{}, err
	}
	commit, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := global.DB.Model(repository).Updates(map[string]any{
		"status": "committed", "worktree_commit": commit, "error_message": "",
	}).Error; err != nil {
		return codeGitDeliveryResult{}, err
	}
	return codeGitDeliveryResult{
		Status: "committed", Commit: commit, Branch: repository.Branch,
		RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
	}, nil
}

func resumeCodeMultiRepositoryDelivery(session *model.AIDevSession, _ uint) (codeGitDeliveryResult, error) {
	return resumeCodeMultiRepositoryDeliveryWithProgress(session, 0, nil, nil)
}

func resumeCodeMultiRepositoryDeliveryWithProgress(
	session *model.AIDevSession,
	userID uint,
	lease *codeExecutionLease,
	report codeDeliveryProgressReporter,
) (codeGitDeliveryResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("会话多仓库 Worktree 元数据不可用")
	}
	if codeMultiRepositoryAllCompleted(repositories) {
		return publishCodeMultiRepositoryDeliveryWithProgress(session, report)
	}
	publishingStarted := false
	if codeMultiRepositoryDeliveryFrozen(repositories) {
		if err := restoreCodeMultiRepositoryIntegrationWorktrees(session, repositories); err != nil {
			return codeGitDeliveryResult{}, err
		}
		if publishingStarted, err = codeMultiRepositoryPublishStarted(session, repositories); err != nil {
			return codeGitDeliveryResult{}, err
		}
	} else {
		prepared, prepareErr := prepareCodeMultiRepositoryDeliveryWithProgress(session, report)
		if prepareErr != nil || prepared.Status == codeDeliveryJobConflict || prepared.Status == codeDeliveryJobPartial {
			return prepared, prepareErr
		}
	}
	runQualityGate := !publishingStarted
	if publishingStarted {
		runQualityGate, err = codeMultiRepositoryHasQualityChecks(session)
		if err != nil {
			return codeGitDeliveryResult{}, err
		}
	}
	if runQualityGate {
		err = runCodeDeliveryQualityGate(session, userID, lease, report)
	}
	if err != nil {
		stored, loadErr := loadCodeSessionRepositories(session.ID)
		if loadErr != nil {
			return codeGitDeliveryResult{}, loadErr
		}
		return codeMultiRepositoryFailure(codeStoredRepositoryDeliveryResults(stored), err), err
	}
	return publishCodeMultiRepositoryDeliveryWithProgress(session, report)
}

func codeMultiRepositoryAllCompleted(repositories []model.AIDevSessionRepository) bool {
	if len(repositories) == 0 {
		return false
	}
	for index := range repositories {
		if repositories[index].Status != codeDeliveryCompleted {
			return false
		}
	}
	return true
}

func codeMultiRepositoryDeliveryFrozen(repositories []model.AIDevSessionRepository) bool {
	if len(repositories) == 0 {
		return false
	}
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		if repository.Status != codeDeliveryMerged || strings.TrimSpace(repository.SourceCommit) == "" ||
			strings.TrimSpace(repository.MergeCommit) == "" {
			return false
		}
	}
	return true
}

func codeMultiRepositoryPublishStarted(
	session *model.AIDevSession,
	repositories []model.AIDevSessionRepository,
) (bool, error) {
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted {
			continue
		}
		if repository.SourceAppliedAt != nil || repository.PushStatus == codePushPushed {
			return true, nil
		}
		if strings.TrimSpace(repository.MergeCommit) == "" || strings.TrimSpace(repository.TargetBranch) == "" {
			continue
		}
		commit, err := runCodeGit(repository.SourceDir, "rev-parse", "refs/heads/"+repository.TargetBranch)
		if err == nil && strings.TrimSpace(commit) == strings.TrimSpace(repository.MergeCommit) {
			if err := markCodeMultiRepositorySourceApplied(repository); err != nil {
				return false, err
			}
			return true, nil
		}
		remoteBranch := deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)
		if !codeDeliveryHasRemote(repository.RemoteName, remoteBranch) {
			continue
		}
		if _, err := fetchCodeRepositoryWithCredential(
			repository.SourceDir, repository.RemoteName, codeProjectGitCredentialID(session.ProjectID),
		); err != nil {
			return false, err
		}
		remoteCommit, err := runCodeGit(
			repository.SourceDir, "rev-parse", "refs/remotes/"+repository.RemoteName+"/"+remoteBranch,
		)
		if err == nil && strings.TrimSpace(remoteCommit) == strings.TrimSpace(repository.MergeCommit) {
			return true, nil
		}
	}
	return false, nil
}

func codeStoredRepositoryDeliveryResult(repository *model.AIDevSessionRepository) codeRepositoryDeliveryResult {
	pushStatus := repository.PushStatus
	if !codeDeliveryHasRemote(repository.RemoteName, deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch)) {
		pushStatus = "local"
	}
	return codeRepositoryDeliveryResult{
		RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
		Status: repository.Status, Branch: repository.Branch, TargetBranch: repository.TargetBranch,
		Remote: repository.RemoteName, RemoteBranch: deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
		Commit: repository.MergeCommit, PushStatus: pushStatus, PushedCommit: repository.PushedCommit,
		SourceAppliedAt: repository.SourceAppliedAt, ErrorMessage: repository.PushError,
	}
}

func codeMultiRepositoryFailure(results []codeRepositoryDeliveryResult, err error) codeGitDeliveryResult {
	status := codeDeliveryJobFailed
	if hasCompletedCodeRepositoryResult(results) {
		status = codeDeliveryJobPartial
	}
	return codeGitDeliveryResult{
		Status: status, ResultType: codeMultiRepositoryResultType(results),
		ErrorMessage: err.Error(), Repositories: results,
	}
}

func hasCompletedCodeRepositoryResult(results []codeRepositoryDeliveryResult) bool {
	for _, result := range results {
		if (result.Status != codeDeliveryCompleted || result.SourceAppliedAt != nil) &&
			(result.SourceAppliedAt != nil || result.PushStatus == codePushPushed) {
			return true
		}
	}
	return false
}

func codeMultiRepositoryResultType(results []codeRepositoryDeliveryResult) string {
	hasLocal, hasRemote := false, false
	for _, result := range results {
		if result.Status == codeDeliveryCompleted && result.SourceAppliedAt == nil {
			continue
		}
		if result.PushStatus == codePushPushed {
			hasRemote = true
		} else if result.SourceAppliedAt != nil {
			hasLocal = true
		}
	}
	if hasLocal && hasRemote {
		return "mixed"
	}
	if hasRemote {
		return "remote_verified"
	}
	if hasLocal {
		return "local"
	}
	return ""
}
