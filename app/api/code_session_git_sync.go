package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type codeSessionGitSyncRepository struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Branch       string `json:"branch"`
	Remote       string `json:"remote,omitempty"`
	RemoteBranch string `json:"remoteBranch,omitempty"`
	LocalCommit  string `json:"localCommit,omitempty"`
	RemoteCommit string `json:"remoteCommit,omitempty"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	CanSync      bool   `json:"canSync"`
	Updated      bool   `json:"updated"`
}

type codeSessionGitSyncStatus struct {
	SessionID    uint                           `json:"sessionId"`
	Status       string                         `json:"status"`
	CanSync      bool                           `json:"canSync"`
	Repositories []codeSessionGitSyncRepository `json:"repositories"`
}

type codeSessionGitSyncTarget struct {
	ID, Name, SourceDir, WorktreeDir, Branch string
	TargetBranch, Remote, RemoteBranch       string
	BaseCommit                               string
	RepositoryID                             uint
	HasGitlink                               bool
}

func codeSessionGitSyncTargets(session *model.AIDevSession) ([]codeSessionGitSyncTarget, error) {
	return codeSessionGitSyncTargetsWithDB(global.DB, session)
}

func codeSessionGitSyncTargetsWithDB(db *gorm.DB, session *model.AIDevSession) ([]codeSessionGitSyncTarget, error) {
	if session == nil {
		return nil, errors.New("开发会话不可用")
	}
	if session.IsolationMode == codeIsolationMultiWorktree {
		var repositories []model.AIDevSessionRepository
		err := db.Where("session_id = ?", session.ID).Order("link_name asc").Find(&repositories).Error
		if err != nil || len(repositories) == 0 {
			return nil, errors.New("会话多仓库 Worktree 元数据不完整")
		}
		gitlinkRepositories := make(map[string]bool)
		for _, repository := range repositories {
			if repository.ParentSourceDir != "" && repository.GitlinkPath != "" {
				gitlinkRepositories[repository.SourceDir] = true
				gitlinkRepositories[repository.ParentSourceDir] = true
			}
		}
		targets := make([]codeSessionGitSyncTarget, 0, len(repositories))
		for index := range repositories {
			repository := &repositories[index]
			if err := validateCodeSessionRepositoryWorktree(session, repository); err != nil {
				return nil, err
			}
			targets = append(targets, codeSessionGitSyncTarget{
				ID: codeSessionRepositoryID(repository.ID), Name: repository.LinkName,
				SourceDir: repository.SourceDir, WorktreeDir: repository.WorktreeDir, Branch: repository.Branch,
				TargetBranch: repository.TargetBranch, Remote: repository.RemoteName,
				RemoteBranch: repository.RemoteBranch, BaseCommit: repository.BaseCommit,
				RepositoryID: repository.ID,
				HasGitlink:   gitlinkRepositories[repository.SourceDir],
			})
		}
		return targets, nil
	}
	if err := validateCodeWorktreeDeliverySession(session); err != nil {
		return nil, err
	}
	return []codeSessionGitSyncTarget{{
		ID: "session", Name: filepath.Base(session.SourceWorkDir), SourceDir: session.SourceWorkDir,
		WorktreeDir: session.WorkDir, Branch: session.WorktreeBranch, TargetBranch: session.TargetBranch,
		Remote: session.RemoteName, RemoteBranch: session.RemoteBranch, BaseCommit: session.BaseCommit,
	}}, nil
}

func codeSessionGitRemoteRef(target codeSessionGitSyncTarget) string {
	if target.Remote == "" || target.RemoteBranch == "" {
		return ""
	}
	return "refs/remotes/" + target.Remote + "/" + target.RemoteBranch
}

func inspectCodeSessionGitSyncTarget(target codeSessionGitSyncTarget, fetchErr error) codeSessionGitSyncRepository {
	result := codeSessionGitSyncRepository{
		ID: target.ID, Name: target.Name, Branch: target.Branch,
		Remote: target.Remote, RemoteBranch: target.RemoteBranch, Status: "blocked",
	}
	currentBranch, err := runCodeGit(target.WorktreeDir, "branch", "--show-current")
	if err != nil || currentBranch != target.Branch {
		result.Reason = "branch_mismatch"
		return result
	}
	status, err := runCodeGit(target.WorktreeDir, "status", "--porcelain")
	if err != nil {
		result.Reason = "repository_unavailable"
		return result
	}
	if strings.TrimSpace(status) != "" {
		result.Status, result.Reason = "dirty", "uncommitted_changes"
		return result
	}
	result.LocalCommit, err = runCodeGit(target.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		result.Reason = "local_commit_unavailable"
		return result
	}
	if target.HasGitlink {
		result.Reason = "gitlink_coordination_required"
		return result
	}
	if target.Remote == "" || target.RemoteBranch == "" {
		result.Status = "local"
		return result
	}
	if fetchErr != nil {
		result.Status, result.Reason = "offline", "fetch_failed"
		return result
	}
	remoteRef := codeSessionGitRemoteRef(target)
	result.RemoteCommit, err = runCodeGit(target.WorktreeDir, "rev-parse", remoteRef)
	if err != nil {
		result.Status, result.Reason = "offline", "remote_ref_unavailable"
		return result
	}
	counts, err := runCodeGit(target.WorktreeDir, "rev-list", "--left-right", "--count", "HEAD..."+remoteRef)
	fields := strings.Fields(counts)
	if err != nil || len(fields) != 2 {
		result.Reason = "comparison_failed"
		return result
	}
	result.Ahead, _ = strconv.Atoi(fields[0])
	result.Behind, _ = strconv.Atoi(fields[1])
	switch {
	case result.Ahead == 0 && result.Behind == 0:
		result.Status = "synced"
	case result.Ahead > 0 && result.Behind > 0:
		result.Status, result.Reason = "diverged", "history_diverged"
	case result.LocalCommit != target.BaseCommit:
		result.Status, result.Reason = "local_ahead", "local_commits"
	case result.Ahead == 0 && result.Behind > 0:
		result.Status, result.CanSync = "behind", true
	default:
		result.Status, result.Reason = "remote_behind", "remote_history_rewritten"
	}
	return result
}

func inspectCodeSessionGitSyncTargets(sessionID uint, targets []codeSessionGitSyncTarget, fetchErrors map[string]error) codeSessionGitSyncStatus {
	result := codeSessionGitSyncStatus{SessionID: sessionID, Status: "synced", Repositories: make([]codeSessionGitSyncRepository, 0, len(targets))}
	priority := map[string]int{"synced": 0, "local": 1, "behind": 2, "offline": 3, "local_ahead": 4, "dirty": 5, "diverged": 6, "remote_behind": 7, "blocked": 8}
	hasDirtyRepository := false
	for _, target := range targets {
		repository := inspectCodeSessionGitSyncTarget(target, fetchErrors[target.ID])
		result.Repositories = append(result.Repositories, repository)
		result.CanSync = result.CanSync || repository.CanSync
		hasDirtyRepository = hasDirtyRepository || repository.Status == "dirty"
		if priority[repository.Status] > priority[result.Status] {
			result.Status = repository.Status
		}
	}
	if hasDirtyRepository {
		result.CanSync = false
		for index := range result.Repositories {
			result.Repositories[index].CanSync = false
		}
	}
	return result
}

func fetchCodeSessionGitTargets(targets []codeSessionGitSyncTarget) map[string]error {
	errorsByID := make(map[string]error)
	for _, target := range targets {
		if target.Remote == "" {
			continue
		}
		_, errorsByID[target.ID] = fetchCodeRepository(target.SourceDir, target.Remote)
	}
	return errorsByID
}

func validateCodeSessionGitSyncIdle(tx *gorm.DB, sessionID uint) error {
	var count int64
	err := tx.Model(&model.AIInstruction{}).Where("session_id = ? AND status IN ?", sessionID, []string{"queued", "running", "pending_approval"}).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("当前会话仍有待处理指令，完成或停止后再同步远端")
	}
	return nil
}

func persistCodeSessionGitBaseline(tx *gorm.DB, session *model.AIDevSession, target codeSessionGitSyncTarget, commit string) error {
	if target.RepositoryID == 0 {
		session.BaseCommit, session.RemoteCommit, session.RepositorySync = commit, commit, "synced"
		return tx.Model(session).Updates(map[string]any{"base_commit": commit, "remote_commit": commit, "repository_sync": "synced"}).Error
	}
	return tx.Model(&model.AIDevSessionRepository{}).Where("id = ? AND session_id = ?", target.RepositoryID, session.ID).
		Updates(map[string]any{"base_commit": commit, "remote_commit": commit, "sync_status": "synced"}).Error
}

func acquireCodeSessionGitSyncLeases(targets []codeSessionGitSyncTarget, purpose string) (func(), error) {
	keys := make([]string, 0, len(targets))
	for _, target := range targets {
		keys = append(keys, codeDeliveryRepositoryKey(target.SourceDir, target.Remote, target.TargetBranch))
	}
	owner := newCodeRepositoryLeaseOwner(purpose)
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, errors.New("项目主仓正在同步或交付，请稍后重试")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go heartbeatCodeRepositoryLeases(ctx, owner, keys)
	return func() {
		cancel()
		_ = releaseCodeRepositoryLeases(owner, keys)
	}, nil
}

func checkCodeSessionGitSync(c fiber.Ctx) (codeSessionGitSyncStatus, *model.AIDevSession, error) {
	session, _, err := getCodeGitSessionContext(c)
	if err != nil {
		return codeSessionGitSyncStatus{}, nil, err
	}
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		return codeSessionGitSyncStatus{}, session, err
	}
	for _, target := range targets {
		if err := validateAIProjectWorkDirForClaims(target.SourceDir, c.Locals(constant.AppAuthName).(*token.CustomClaims)); err != nil {
			return codeSessionGitSyncStatus{}, session, err
		}
	}
	release, err := acquireCodeSessionGitSyncLeases(targets, "session-sync-check")
	if err != nil {
		return codeSessionGitSyncStatus{}, session, err
	}
	defer release()
	fetchErrors := fetchCodeSessionGitTargets(targets)
	return inspectCodeSessionGitSyncTargets(session.ID, targets, fetchErrors), session, nil
}

func syncCodeSessionGitRepositoryOperation(c fiber.Ctx, syncRepositoryID string) (codeSessionGitSyncStatus, *model.AIDevSession, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	session, _, err := getCodeGitSessionContext(c)
	if err != nil {
		return codeSessionGitSyncStatus{}, nil, err
	}
	project, err := getCodeProjectWithPermission(session.ProjectID, claims)
	if err != nil {
		return codeSessionGitSyncStatus{}, session, err
	}
	if !beginCodeRepositoryOperation(project) {
		return codeSessionGitSyncStatus{}, session, errors.New("项目主仓正在被终端使用，请关闭终端后再同步会话")
	}
	defer endCodeRepositoryOperation(project)
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		return codeSessionGitSyncStatus{}, session, err
	}
	for _, target := range targets {
		if err := validateAIProjectWorkDirForClaims(target.SourceDir, claims); err != nil {
			return codeSessionGitSyncStatus{}, session, err
		}
	}
	release, err := acquireCodeSessionGitSyncLeases(targets, "session-sync")
	if err != nil {
		return codeSessionGitSyncStatus{}, session, err
	}
	defer release()
	var result codeSessionGitSyncStatus
	err = runCodeSessionWorkspaceMutationWithTx(session, func(tx *gorm.DB, current *model.AIDevSession) error {
		if err := validateCodeSessionGitSyncIdle(tx, current.ID); err != nil {
			return err
		}
		currentTargets, err := codeSessionGitSyncTargetsWithDB(tx, current)
		if err != nil {
			return err
		}
		fetchErrors := fetchCodeSessionGitTargets(currentTargets)
		result = inspectCodeSessionGitSyncTargets(current.ID, currentTargets, fetchErrors)
		var selected *codeSessionGitSyncTarget
		for index := range currentTargets {
			if currentTargets[index].ID == syncRepositoryID {
				selected = &currentTargets[index]
				break
			}
		}
		if selected == nil {
			return errors.New("Git 仓库不存在或不属于当前会话")
		}
		var state *codeSessionGitSyncRepository
		for index := range result.Repositories {
			if result.Repositories[index].ID == syncRepositoryID {
				state = &result.Repositories[index]
				break
			}
		}
		if state == nil {
			return errors.New("Git 仓库同步状态不可用")
		}
		for _, repository := range result.Repositories {
			if repository.Status == "dirty" {
				return fmt.Errorf("仓库 %s 存在未提交修改，请先提交或清理后再同步", repository.Name)
			}
		}
		if !state.CanSync {
			return fmt.Errorf("仓库当前状态为 %s，不能安全同步到会话", state.Status)
		}
		if _, err := runCodeGit(selected.WorktreeDir, "merge", "--ff-only", codeSessionGitRemoteRef(*selected)); err != nil {
			return err
		}
		previousCommit := state.LocalCommit
		commit, err := runCodeGit(selected.WorktreeDir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if err := persistCodeSessionGitBaseline(tx, current, *selected, commit); err != nil {
			_, _ = runCodeGit(selected.WorktreeDir, "reset", "--hard", previousCommit)
			return err
		}
		selected.BaseCommit = commit
		if current.IsolationMode == codeIsolationMultiWorktree {
			var repositories []model.AIDevSessionRepository
			err := tx.Where("session_id = ?", current.ID).Order("link_name asc").Find(&repositories).Error
			if err != nil {
				return err
			}
			for index := range repositories {
				if repositories[index].ID == selected.RepositoryID {
					repositories[index].BaseCommit, repositories[index].RemoteCommit, repositories[index].SyncStatus = commit, commit, "synced"
				}
			}
			if err := writeCodeSessionManifest(current.WorkDir, repositories); err != nil {
				_, _ = runCodeGit(selected.WorktreeDir, "reset", "--hard", previousCommit)
				return err
			}
		}
		result = inspectCodeSessionGitSyncTargets(current.ID, currentTargets, fetchErrors)
		for index := range result.Repositories {
			if result.Repositories[index].ID == syncRepositoryID {
				result.Repositories[index].Updated = true
			}
		}
		return nil
	})
	if errors.Is(err, errCodeSessionWorkspaceBusy) {
		err = errors.New("当前会话正在执行 AI 指令或终端操作，请完成或停止后再同步远端")
	}
	return result, session, err
}
