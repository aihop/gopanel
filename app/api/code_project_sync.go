package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const codeProjectSyncInterval = time.Minute

type codeProjectRepositorySpec struct {
	Name         string
	Path         string
	ParentPath   string
	GitlinkPath  string
	Branch       string
	Remote       string
	RemoteBranch string
	RemoteRef    string
	LeaseKey     string
}

type codeProjectRepositorySync struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Branch       string `json:"branch"`
	Remote       string `json:"remote,omitempty"`
	RemoteBranch string `json:"remoteBranch,omitempty"`
	LocalCommit  string `json:"localCommit,omitempty"`
	RemoteCommit string `json:"remoteCommit,omitempty"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
}

type codeProjectSyncStatus struct {
	ProjectID    uint                        `json:"projectId"`
	Status       string                      `json:"status"`
	CanSync      bool                        `json:"canSync"`
	Updated      bool                        `json:"updated"`
	Repositories []codeProjectRepositorySync `json:"repositories"`
}

var codeProjectSyncOnce sync.Once

func codeProjectSourceDirs(project *model.AIProject) []string {
	sourceDirs := project.SourceDirs
	if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		sourceDirs = aiProjectWorkspaceSourceDirs(project.WorkDir)
		if len(sourceDirs) == 0 {
			sourceDirs = []string{project.WorkDir}
		}
	}
	return sourceDirs
}

func codeProjectRepositorySpecs(project *model.AIProject) ([]codeProjectRepositorySpec, error) {
	sourceDirs := codeProjectSourceDirs(project)
	policy, err := codeProjectDeliveryPolicy(project, sourceDirs)
	if err != nil {
		return nil, err
	}
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return nil, err
	}
	specs := make([]codeProjectRepositorySpec, 0, len(candidates))
	for _, candidate := range candidates {
		branch := ""
		if candidate.SourceDir == policy.PrimaryRepository {
			branch = policy.DeliveryBranch
		} else {
			currentBranch, branchErr := runCodeGit(candidate.SourceDir, "branch", "--show-current")
			if branchErr != nil {
				return nil, fmt.Errorf("读取仓库 %s 当前分支失败：%w", filepath.Base(candidate.SourceDir), branchErr)
			}
			branch = strings.TrimSpace(currentBranch)
		}
		remote, remoteRef := "", ""
		if branch != "" {
			remote, remoteRef = codeRepositoryRemoteTracking(candidate.SourceDir, branch)
			if remote != "" && remoteRef == "" {
				remoteRef = remote + "/" + branch
			}
		}
		specs = append(specs, codeProjectRepositorySpec{
			Name: filepath.Base(candidate.SourceDir), Path: candidate.SourceDir,
			ParentPath: candidate.ParentSourceDir, GitlinkPath: candidate.GitlinkPath, Branch: branch,
			Remote: remote, RemoteBranch: codeRemoteBranch(remote, remoteRef, branch), RemoteRef: remoteRef,
			LeaseKey: codeDeliveryRepositoryKey(candidate.SourceDir, remote, branch),
		})
	}
	if len(specs) == 0 {
		return nil, errors.New("项目目录中未发现 Git 仓库")
	}
	return specs, nil
}

func validateCodeProjectGitlinkTargets(specs []codeProjectRepositorySpec) error {
	byPath := make(map[string]codeProjectRepositorySpec, len(specs))
	for _, spec := range specs {
		byPath[spec.Path] = spec
	}
	for _, child := range specs {
		if child.ParentPath == "" || child.GitlinkPath == "" {
			continue
		}
		parent, exists := byPath[child.ParentPath]
		if !exists {
			return fmt.Errorf("Gitlink 子仓 %s 的父仓不在同步范围内", child.Name)
		}
		parentTarget := parent.Branch
		if parent.RemoteRef != "" {
			parentTarget = parent.RemoteRef
		}
		childTarget := child.Branch
		if child.RemoteRef != "" {
			childTarget = child.RemoteRef
		}
		if childTarget == "" {
			childTarget = "HEAD"
		}
		entry, err := runCodeGit(parent.Path, "ls-tree", parentTarget, "--", child.GitlinkPath)
		if err != nil || strings.TrimSpace(entry) == "" {
			return fmt.Errorf("父仓 %s 的目标分支未包含 Gitlink %s", parent.Name, child.GitlinkPath)
		}
		fields := strings.Fields(entry)
		childCommit, commitErr := runCodeGit(child.Path, "rev-parse", childTarget)
		if len(fields) < 3 || fields[0] != "160000" || commitErr != nil || fields[2] != childCommit {
			return fmt.Errorf("Gitlink %s 的父仓指针与子仓目标提交不一致，请先人工协调远端分支", child.GitlinkPath)
		}
	}
	return nil
}

func inspectCodeProjectRepositorySync(spec codeProjectRepositorySpec) codeProjectRepositorySync {
	result := codeProjectRepositorySync{
		Name: spec.Name, Path: spec.Path, Branch: spec.Branch, Remote: spec.Remote,
		RemoteBranch: spec.RemoteBranch, Status: "blocked",
	}
	currentBranch, err := runCodeGit(spec.Path, "branch", "--show-current")
	if err != nil || currentBranch != spec.Branch {
		result.Reason = "branch_mismatch"
		return result
	}
	dirty, err := runCodeGit(spec.Path, "status", "--porcelain")
	if err != nil {
		result.Reason = "repository_unavailable"
		return result
	}
	if strings.TrimSpace(dirty) != "" {
		result.Status, result.Reason = "dirty", "uncommitted_changes"
		return result
	}
	localTarget := spec.Branch
	if localTarget == "" {
		localTarget = "HEAD"
	}
	result.LocalCommit, err = runCodeGit(spec.Path, "rev-parse", localTarget)
	if err != nil {
		result.Reason = "local_branch_unavailable"
		return result
	}
	if spec.Remote == "" || spec.RemoteRef == "" {
		result.Status = "local"
		return result
	}
	result.RemoteCommit, err = runCodeGit(spec.Path, "rev-parse", spec.RemoteRef)
	if err != nil {
		result.Status, result.Reason = "offline", "remote_ref_unavailable"
		return result
	}
	counts, err := runCodeGit(spec.Path, "rev-list", "--left-right", "--count", localTarget+"..."+spec.RemoteRef)
	if err != nil || len(strings.Fields(counts)) != 2 {
		result.Reason = "comparison_failed"
		return result
	}
	fields := strings.Fields(counts)
	result.Ahead, _ = strconv.Atoi(fields[0])
	result.Behind, _ = strconv.Atoi(fields[1])
	switch {
	case result.Ahead == 0 && result.Behind == 0:
		result.Status = "synced"
	case result.Ahead == 0:
		result.Status = "behind"
	case result.Behind == 0:
		result.Status = "ahead"
	default:
		result.Status = "diverged"
	}
	return result
}

func codeProjectHasActiveTerminal(project *model.AIProject) bool {
	if global.DB == nil {
		return true
	}
	paths := append([]string(nil), codeProjectSourceDirs(project)...)
	if strings.TrimSpace(project.WorkDir) != "" {
		paths = append(paths, project.WorkDir)
	}
	var terminals []model.HostTerminalSession
	if err := global.DB.Where("status IN ?", []string{"starting", "running"}).Find(&terminals).Error; err != nil {
		return true
	}
	for _, terminal := range terminals {
		if hostTerminals.get(terminal.ID) == nil {
			markHostTerminalInterrupted(&terminal)
			continue
		}
		terminalPath := filepath.Clean(terminal.WorkDir)
		for _, projectPath := range paths {
			projectPath = filepath.Clean(projectPath)
			if terminalPath == projectPath || isPathInside(terminalPath, projectPath) || isPathInside(projectPath, terminalPath) {
				return true
			}
		}
	}
	return false
}

func createCodeSessionWorktreeWithLease(session *model.AIDevSession, project *model.AIProject, includeUncommitted bool) error {
	if !beginCodeRepositoryOperation(project) {
		return errors.New("项目主仓正在被项目终端使用，请关闭终端后再创建会话")
	}
	defer endCodeRepositoryOperation(project)
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.LeaseKey)
	}
	owner := newCodeRepositoryLeaseOwner("worktree-create")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return err
	}
	if !acquired {
		return errors.New("项目主仓正在同步或交付，请稍后重试")
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, keys) }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go heartbeatCodeRepositoryLeases(ctx, owner, keys)
	return createCodeSessionWorktree(session, project, includeUncommitted)
}

func inspectCodeProjectSync(project *model.AIProject) (codeProjectSyncStatus, error) {
	return inspectCodeProjectSyncIgnoringOwner(project, "")
}

func inspectCodeProjectSyncIgnoringOwner(project *model.AIProject, ignoredOwner string) (codeProjectSyncStatus, error) {
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		return codeProjectSyncStatus{}, err
	}
	result := codeProjectSyncStatus{ProjectID: project.ID, Status: "synced", Repositories: make([]codeProjectRepositorySync, 0, len(specs))}
	if codeProjectHasActiveTerminal(project) {
		result.Status = "blocked"
		for _, spec := range specs {
			repository := inspectCodeProjectRepositorySync(spec)
			repository.Status, repository.Reason = "blocked", "active_terminal"
			result.Repositories = append(result.Repositories, repository)
		}
		return result, nil
	}
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.LeaseKey)
	}
	if codeProjectRepositoriesBusy(keys, ignoredOwner) {
		result.Status = "blocked"
		for _, spec := range specs {
			repository := inspectCodeProjectRepositorySync(spec)
			repository.Status, repository.Reason = "blocked", "repository_busy"
			result.Repositories = append(result.Repositories, repository)
		}
		return result, nil
	}
	priority := map[string]int{"synced": 0, "local": 1, "behind": 2, "ahead": 3, "offline": 4, "dirty": 5, "diverged": 6, "blocked": 7}
	hasRemote := false
	for _, spec := range specs {
		hasRemote = hasRemote || spec.Remote != ""
		repository := inspectCodeProjectRepositorySync(spec)
		result.Repositories = append(result.Repositories, repository)
		if priority[repository.Status] > priority[result.Status] {
			result.Status = repository.Status
		}
	}
	result.CanSync = hasRemote && (result.Status == "synced" || result.Status == "behind" || result.Status == "offline")
	return result, nil
}

func codeProjectRepositoriesBusy(keys []string, ignoredOwner string) bool {
	if global.DB == nil || len(keys) == 0 {
		return true
	}
	query := global.DB.Model(&model.AICodeDeliveryLease{}).
		Where("repository_key IN ? AND lease_expires_at > ?", keys, time.Now())
	if ignoredOwner != "" {
		query = query.Where("lease_owner <> ?", ignoredOwner)
	}
	var count int64
	return query.Count(&count).Error != nil || count > 0
}

func syncCodeProject(project *model.AIProject, automatic bool) (codeProjectSyncStatus, error) {
	if !beginCodeRepositoryOperation(project) {
		return inspectCodeProjectSync(project)
	}
	defer endCodeRepositoryOperation(project)
	specs, err := codeProjectRepositorySpecs(project)
	if err != nil {
		return codeProjectSyncStatus{}, err
	}
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.LeaseKey)
	}
	owner := newCodeRepositoryLeaseOwner("project-sync")
	acquired, err := acquireCodeRepositoryLeases(owner, 0, keys)
	if err != nil {
		return codeProjectSyncStatus{}, err
	}
	if !acquired {
		result, inspectErr := inspectCodeProjectSync(project)
		if inspectErr == nil {
			result.Status, result.CanSync = "blocked", false
		}
		return result, inspectErr
	}
	defer func() { _ = releaseCodeRepositoryLeases(owner, keys) }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go heartbeatCodeRepositoryLeases(ctx, owner, keys)
	initial := make([]codeProjectRepositorySync, 0, len(specs))
	for _, spec := range specs {
		state := inspectCodeProjectRepositorySync(spec)
		initial = append(initial, state)
		if state.Status == "dirty" || state.Status == "blocked" {
			return codeProjectSyncStatus{ProjectID: project.ID, Status: state.Status, Repositories: initial}, nil
		}
	}
	for _, spec := range specs {
		if spec.Remote != "" {
			if _, fetchErr := fetchCodeRepository(spec.Path, spec.Remote); fetchErr != nil {
				result, inspectErr := inspectCodeProjectSyncIgnoringOwner(project, owner)
				if inspectErr == nil {
					result.Status, result.CanSync = "offline", false
					for index := range result.Repositories {
						if result.Repositories[index].Path == spec.Path {
							result.Repositories[index].Status = "offline"
							result.Repositories[index].Reason = "fetch_failed"
						}
					}
				}
				return result, inspectErr
			}
		}
	}
	for _, spec := range specs {
		state := inspectCodeProjectRepositorySync(spec)
		if state.Status != "synced" && state.Status != "local" && state.Status != "behind" {
			return inspectCodeProjectSyncIgnoringOwner(project, owner)
		}
	}
	if err := validateCodeProjectGitlinkTargets(specs); err != nil {
		if automatic {
			result, inspectErr := inspectCodeProjectSyncIgnoringOwner(project, owner)
			if inspectErr == nil {
				result.Status, result.CanSync = "blocked", false
			}
			return result, inspectErr
		}
		return codeProjectSyncStatus{}, err
	}
	updated := false
	for _, spec := range specs {
		state := inspectCodeProjectRepositorySync(spec)
		if state.Status == "behind" {
			if _, err := runCodeGit(spec.Path, "merge", "--ff-only", spec.RemoteRef); err != nil {
				return codeProjectSyncStatus{}, err
			}
			updated = true
		}
	}
	result, err := inspectCodeProjectSyncIgnoringOwner(project, owner)
	result.Updated = updated
	return result, err
}

func GetCodeProjectSync(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := getCodeProjectWithPermission(uint(projectID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := inspectCodeProjectSync(project)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func SyncCodeProject(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := c.Bind().JSON(&req); err != nil || !req.Confirm {
		return c.JSON(e.Fail(errors.New("同步主仓需要明确确认")))
	}
	project, err := getCodeProjectWithPermission(uint(projectID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	startedAt := time.Now()
	result, err := syncCodeProject(project, false)
	if err != nil {
		recordCodeAudit(claims.UserId, project.ID, 0, "project_git_sync", "failed", "project", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	if result.Status != "synced" && result.Status != "local" {
		err = fmt.Errorf("本地主仓当前状态为 %s，已停止同步且未修改仓库", result.Status)
		recordCodeAudit(claims.UserId, project.ID, 0, "project_git_sync", "blocked", "project", err.Error(), c.IP(), startedAt, nil)
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, project.ID, 0, "project_git_sync", "success", "project", result.Status, c.IP(), startedAt, codeAuditMeta{"automatic": false})
	return c.JSON(e.Succ(result))
}

func StartCodeProjectSync() {
	codeProjectSyncOnce.Do(func() {
		go func() {
			syncAllCodeProjects()
			ticker := time.NewTicker(codeProjectSyncInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					syncAllCodeProjects()
				case <-codeExecutions.stop:
					return
				}
			}
		}()
	})
}

func syncAllCodeProjects() {
	var projects []model.AIProject
	if global.DB == nil || global.DB.Find(&projects).Error != nil {
		return
	}
	for index := range projects {
		startedAt := time.Now()
		result, err := syncCodeProject(&projects[index], true)
		if err == nil && result.Status == "synced" && result.Updated {
			recordCodeAudit(projects[index].CreatorID, projects[index].ID, 0, "project_git_sync", "success", "project", "synced", "", startedAt, codeAuditMeta{"automatic": true})
		}
	}
}
