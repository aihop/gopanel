package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"gorm.io/gorm"
)

type codeRepositoryDeliveryResult struct {
	RepositoryID   string   `json:"repositoryId"`
	RepositoryName string   `json:"repositoryName"`
	Status         string   `json:"status"`
	Branch         string   `json:"branch"`
	TargetBranch   string   `json:"targetBranch"`
	Remote         string   `json:"remote,omitempty"`
	RemoteBranch   string   `json:"remoteBranch,omitempty"`
	Commit         string   `json:"commit,omitempty"`
	PushStatus     string   `json:"pushStatus"`
	PushedCommit   string   `json:"pushedCommit,omitempty"`
	ErrorMessage   string   `json:"errorMessage,omitempty"`
	ConflictFiles  []string `json:"conflictFiles,omitempty"`
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

func prepareCodeMultiRepositoryDelivery(session *model.AIDevSession, repositories []model.AIDevSessionRepository) error {
	return prepareCodeMultiRepositoryDeliveryWithProgress(session, repositories, nil)
}

func prepareCodeMultiRepositoryDeliveryWithProgress(session *model.AIDevSession, repositories []model.AIDevSessionRepository, report codeDeliveryProgressReporter) error {
	if report != nil {
		report(codeDeliveryStageSyncing, 20)
	}
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted || repository.Status == codeDeliveryMerged {
			continue
		}
		status, err := runCodeGit(repository.WorktreeDir, "status", "--porcelain")
		if err != nil || strings.TrimSpace(status) != "" {
			return fmt.Errorf("仓库 %s 仍有未提交变更，请先提交", repository.LinkName)
		}
		targetBranch := repository.TargetBranch
		if targetBranch == "" {
			targetBranch, _ = runCodeGit(repository.SourceDir, "branch", "--show-current")
		}
		targetCommit, err := refreshCodeRepositoryTarget(repository.SourceDir, targetBranch, repository.RemoteName)
		if err != nil {
			return fmt.Errorf("仓库 %s 同步失败：%w", repository.LinkName, err)
		}
		if err := syncCodeWorktreeWithTarget(repository.WorktreeDir, targetBranch); err != nil {
			return fmt.Errorf("仓库 %s 同步失败：%w", repository.LinkName, err)
		}
		commit, err := runCodeGit(repository.WorktreeDir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		repository.TargetBranch, repository.RemoteCommit = targetBranch, targetCommit
		repository.WorktreeCommit = commit
	}
	if report != nil {
		report(codeDeliveryStageQualityCheck, 40)
	}
	if err := validateCodeQualityGate(session); err != nil {
		return err
	}
	for index := range repositories {
		repository := &repositories[index]
		if repository.Status == codeDeliveryCompleted || repository.Status == codeDeliveryMerged {
			continue
		}
		if err := global.DB.Model(repository).Updates(map[string]any{
			"status": codeDeliveryPrepared, "target_branch": repository.TargetBranch,
			"remote_commit": repository.RemoteCommit, "worktree_commit": repository.WorktreeCommit,
			"error_message": "",
		}).Error; err != nil {
			return err
		}
		repository.Status = codeDeliveryPrepared
	}
	return nil
}

func mergeCodeSessionRepository(repository *model.AIDevSessionRepository, repositories []model.AIDevSessionRepository) (codeRepositoryDeliveryResult, error) {
	result := codeRepositoryDeliveryResult{
		RepositoryID: codeSessionRepositoryID(repository.ID), RepositoryName: repository.LinkName,
		Status: repository.Status, Branch: repository.Branch, TargetBranch: repository.TargetBranch,
		Remote: repository.RemoteName, RemoteBranch: deliveryRemoteBranch(repository.RemoteBranch, repository.TargetBranch),
		Commit: repository.MergeCommit, PushStatus: repository.PushStatus,
		PushedCommit: repository.PushedCommit, ErrorMessage: repository.PushError,
	}
	if repository.Status == codeDeliveryMerged || repository.Status == codeDeliveryCompleted {
		return result, nil
	}
	if _, err := runCodeGit(repository.SourceDir, "merge-base", "--is-ancestor", repository.WorktreeCommit, "HEAD"); err != nil {
		if statusErr := validateCodeRepositoryMergeStatus(repository, repositories); statusErr != nil {
			return result, statusErr
		}
		if _, err := runCodeGit(repository.SourceDir, "merge", "--no-ff", "--no-edit", repository.Branch); err != nil {
			conflicts := codeGitConflictFiles(repository.SourceDir)
			_, _ = runCodeGit(repository.SourceDir, "merge", "--abort")
			if len(conflicts) > 0 {
				result.Status, result.ConflictFiles = "conflict", conflicts
				_ = global.DB.Model(repository).Updates(map[string]any{
					"status": "conflict", "error_message": strings.Join(conflicts, ", "),
				}).Error
				return result, nil
			}
			return result, err
		}
	}
	commit, err := runCodeGit(repository.SourceDir, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	now := time.Now()
	if err := global.DB.Model(repository).Updates(map[string]any{
		"status": codeDeliveryMerged, "merge_commit": commit, "merged_at": now, "error_message": "",
	}).Error; err != nil {
		return result, err
	}
	repository.Status, repository.MergeCommit, repository.MergedAt = codeDeliveryMerged, commit, &now
	result.Status, result.Commit = codeDeliveryMerged, commit
	return result, nil
}

func cleanupMergedCodeSessionRepository(repository *model.AIDevSessionRepository) error {
	if repository.Status == codeDeliveryCompleted {
		return nil
	}
	if _, err := os.Stat(repository.WorktreeDir); err == nil {
		if _, err := runCodeGit(repository.SourceDir, "worktree", "remove", repository.WorktreeDir); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if branches, err := runCodeGit(repository.SourceDir, "branch", "--list", repository.Branch); err == nil && strings.TrimSpace(branches) != "" {
		if _, err := runCodeGit(repository.SourceDir, "branch", "-d", "--", repository.Branch); err != nil {
			return err
		}
	}
	now := time.Now()
	if err := global.DB.Model(repository).Updates(map[string]any{
		"status": codeDeliveryCompleted, "completed_at": now, "error_message": "",
	}).Error; err != nil {
		return err
	}
	repository.Status, repository.CompletedAt = codeDeliveryCompleted, &now
	return nil
}

func completeCodeMultiRepositorySession(session *model.AIDevSession) error {
	project, err := repo.NewAIProjectRepo().GetProjectByID(session.ProjectID)
	if err != nil || strings.TrimSpace(project.WorkDir) == "" {
		return errors.New("项目聚合工作区不可用")
	}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIDevSession{}).Where("id = ?", session.ID).Updates(map[string]any{
			"work_dir": project.WorkDir, "isolation_mode": "", "source_work_dir": "", "worktree_branch": "",
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.AITask{}).Where("session_id = ?", session.ID).Update("work_dir", project.WorkDir).Error
	})
}

func resumeCodeMultiRepositoryDelivery(session *model.AIDevSession, _ uint) (codeGitDeliveryResult, error) {
	return resumeCodeMultiRepositoryDeliveryWithProgress(session, 0, nil)
}

func resumeCodeMultiRepositoryDeliveryWithProgress(session *model.AIDevSession, _ uint, report codeDeliveryProgressReporter) (codeGitDeliveryResult, error) {
	repositories, err := loadCodeSessionRepositories(session.ID)
	if err != nil || len(repositories) == 0 {
		return codeGitDeliveryResult{}, errors.New("会话多仓库 Worktree 元数据不可用")
	}
	sort.SliceStable(repositories, func(i, j int) bool {
		leftDepth := strings.Count(filepath.Clean(repositories[i].SourceDir), string(filepath.Separator))
		rightDepth := strings.Count(filepath.Clean(repositories[j].SourceDir), string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return repositories[i].LinkName < repositories[j].LinkName
	})
	workspaceDir, err := codeMultiRepositoryWorkspaceDir(session, repositories)
	if err != nil {
		return codeGitDeliveryResult{}, err
	}
	if err := prepareCodeMultiRepositoryDeliveryWithProgress(session, repositories, report); err != nil {
		return codeGitDeliveryResult{}, err
	}
	results := make([]codeRepositoryDeliveryResult, 0, len(repositories))
	for index := range repositories {
		if repositories[index].Status == codeDeliveryCompleted {
			results = append(results, codeStoredRepositoryDeliveryResult(&repositories[index]))
			continue
		}
		if err := syncCodeSessionRepositoryGitlinks(repositories); err != nil {
			return codeMultiRepositoryFailure(results, err), err
		}
		if err := commitCodeRepositoryGitlinkUpdates(&repositories[index], repositories); err != nil {
			return codeMultiRepositoryFailure(results, err), err
		}
		if repositories[index].WorktreeCommit != "" {
			if err := global.DB.Model(&repositories[index]).Updates(map[string]any{
				"worktree_commit": repositories[index].WorktreeCommit, "error_message": "",
			}).Error; err != nil {
				return codeMultiRepositoryFailure(results, err), err
			}
		}
		result, mergeErr := integrateAndPushCodeRepositoryWithProgress(session, &repositories[index], repositories, report)
		results = append(results, result)
		if mergeErr != nil {
			return codeMultiRepositoryFailure(results, mergeErr), mergeErr
		}
		if result.Status == "conflict" {
			conflictResult := codeGitDeliveryResult{
				Status: "conflict", RepositoryID: result.RepositoryID, RepositoryName: result.RepositoryName,
				Branch: result.Branch, ConflictFiles: result.ConflictFiles, Repositories: results,
			}
			if hasCompletedCodeRepositoryResult(results[:len(results)-1]) {
				conflictResult.Status = codeDeliveryJobPartial
			}
			conflictResult.ResultType = codeMultiRepositoryResultType(results)
			return conflictResult, nil
		}
		if report != nil {
			report(codeDeliveryStageCleaning, 90+index*5/max(1, len(repositories)))
		}
		if err := cleanupMergedCodeSessionRepository(&repositories[index]); err != nil {
			return codeMultiRepositoryFailure(results, err), err
		}
	}
	if err := completeCodeMultiRepositorySession(session); err != nil {
		return codeMultiRepositoryFailure(results, err), err
	}
	if err := os.RemoveAll(workspaceDir); err != nil {
		return codeMultiRepositoryFailure(results, err), err
	}
	return codeGitDeliveryResult{Status: "merged", ResultType: codeMultiRepositoryResultType(results), Repositories: results}, nil
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
		ErrorMessage: repository.PushError,
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
		if result.Status == codeDeliveryMerged || result.Status == codeDeliveryCompleted || result.PushStatus == codePushPushed {
			return true
		}
	}
	return false
}

func codeMultiRepositoryResultType(results []codeRepositoryDeliveryResult) string {
	hasLocal, hasRemote := false, false
	for _, result := range results {
		if result.PushStatus == codePushPushed {
			hasRemote = true
		} else if result.Status == codeDeliveryMerged || result.Status == codeDeliveryCompleted || result.PushStatus == "local" {
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
