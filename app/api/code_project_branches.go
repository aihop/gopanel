package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

const maxCodeProjectBranchRepositories = 50

var codeProjectBranchScanExcludedDirs = map[string]struct{}{
	".git": {}, ".cache": {}, ".next": {}, ".nuxt": {}, ".output": {},
	".pnpm-store": {}, ".turbo": {}, ".venv": {}, "build": {}, "coverage": {},
	"dist": {}, "node_modules": {}, "target": {}, "vendor": {},
}

type codeProjectBranch struct {
	Name              string `json:"name"`
	Ref               string `json:"ref"`
	Scope             string `json:"scope"`
	Current           bool   `json:"current"`
	Upstream          string `json:"upstream,omitempty"`
	Commit            string `json:"commit"`
	Subject           string `json:"subject"`
	UpdatedAt         string `json:"updatedAt"`
	Merged            bool   `json:"merged"`
	Managed           bool   `json:"managed"`
	TaskBranch        bool   `json:"taskBranch"`
	Deletable         bool   `json:"deletable"`
	DeleteBlockReason string `json:"deleteBlockReason,omitempty"`
	Additions         int    `json:"additions"`
	Deletions         int    `json:"deletions"`
}

type codeProjectBranchRepository struct {
	Name          string              `json:"name"`
	Path          string              `json:"path"`
	Excluded      bool                `json:"excluded"`
	CurrentBranch string              `json:"currentBranch,omitempty"`
	Detached      bool                `json:"detached"`
	Dirty         bool                `json:"dirty"`
	ChangedFiles  int                 `json:"changedFiles"`
	Additions     int                 `json:"additions"`
	Deletions     int                 `json:"deletions"`
	Branches      []codeProjectBranch `json:"branches"`
}

type codeProjectBranches struct {
	Repositories  []codeProjectBranchRepository `json:"repositories"`
	TotalBranches int                           `json:"totalBranches"`
}

func GetCodeProjectBranches(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := repo.NewAIProjectRepo().GetProjectByID(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(errors.New("项目不存在")))
	}
	if !canManageAIProject(project, claims) {
		return c.JSON(e.Fail(errors.New("无权访问该项目")))
	}
	result, err := inspectCodeProjectBranches(project)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func inspectCodeProjectBranches(project *model.AIProject) (codeProjectBranches, error) {
	repositoryRoots, activeRoots, err := codeProjectBranchRepositoryRoots(project)
	if err != nil {
		return codeProjectBranches{}, err
	}
	result := codeProjectBranches{Repositories: make([]codeProjectBranchRepository, 0, len(repositoryRoots))}
	for _, root := range repositoryRoots {
		_, active := activeRoots[root]
		repository, inspectErr := inspectCodeProjectBranchRepository(project, root, !active)
		if inspectErr != nil {
			return codeProjectBranches{}, inspectErr
		}
		result.TotalBranches += len(repository.Branches)
		result.Repositories = append(result.Repositories, repository)
	}
	return result, nil
}

func codeProjectBranchRepositoryRoots(project *model.AIProject) ([]string, map[string]struct{}, error) {
	if project == nil {
		return nil, nil, errors.New("项目不可用")
	}
	sourceDirs := codeProjectSourceDirs(project)
	currentRoots, err := discoverCodeProjectBranchRepositories(sourceDirs)
	if err != nil {
		return nil, nil, err
	}
	activeRoots := make(map[string]struct{}, len(currentRoots))
	excluded := normalizeCodeExcludedRepositories(project.ExcludedRepositories)
	allRoots := make(map[string]struct{}, len(currentRoots))
	for _, root := range currentRoots {
		allRoots[root] = struct{}{}
		if !isCodeRepositoryExcluded(root, excluded) {
			activeRoots[root] = struct{}{}
		}
	}
	for _, root := range loadCodeProjectHistoricalRepositoryRoots(project.ID) {
		allRoots[root] = struct{}{}
	}
	result := make([]string, 0, len(allRoots))
	for root := range allRoots {
		result = append(result, root)
	}
	sort.Strings(result)
	return result, activeRoots, nil
}

func loadCodeProjectHistoricalRepositoryRoots(projectID uint) []string {
	if global.DB == nil || projectID == 0 {
		return nil
	}
	paths := make([]string, 0)
	queries := []struct {
		model any
		field string
		join  string
	}{
		{&model.AIDevSession{}, "ai_dev_sessions.source_work_dir", "JOIN ai_tasks ON ai_tasks.session_id = ai_dev_sessions.id"},
		{&model.AICodeDelivery{}, "ai_code_deliveries.source_work_dir", "JOIN ai_tasks ON ai_tasks.session_id = ai_code_deliveries.session_id"},
		{&model.AIDevSessionRepository{}, "ai_dev_session_repositories.source_dir", "JOIN ai_tasks ON ai_tasks.session_id = ai_dev_session_repositories.session_id"},
	}
	for _, query := range queries {
		var values []string
		if err := global.DB.Model(query.model).Joins(query.join).
			Where("ai_tasks.project_id = ? AND "+query.field+" <> ''", projectID).
			Pluck(query.field, &values).Error; err == nil {
			paths = append(paths, values...)
		}
	}
	var storedResults []string
	if err := global.DB.Model(&model.AICodeDeliveryJob{}).
		Joins("JOIN ai_tasks ON ai_tasks.session_id = ai_code_delivery_jobs.session_id").
		Where("ai_tasks.project_id = ? AND ai_code_delivery_jobs.repository_results <> ''", projectID).
		Pluck("ai_code_delivery_jobs.repository_results", &storedResults).Error; err == nil {
		for _, stored := range storedResults {
			var repositories []codeRepositoryDeliveryResult
			if json.Unmarshal([]byte(stored), &repositories) != nil {
				continue
			}
			for _, repository := range repositories {
				paths = append(paths, repository.RepositoryPath)
			}
		}
	}
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		root, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(path)))
		if err != nil || root == "." {
			continue
		}
		gitRoot, err := runCodeGit(root, "rev-parse", "--show-toplevel")
		if err != nil || filepath.Clean(gitRoot) != filepath.Clean(root) {
			continue
		}
		if _, exists := seen[root]; exists {
			continue
		}
		seen[root] = struct{}{}
		result = append(result, root)
	}
	return result
}

func discoverCodeProjectBranchRepositories(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{})
	repositories := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		boundary, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(sourceDir)))
		if err != nil {
			return nil, fmt.Errorf("项目目录不可访问：%s", sourceDir)
		}
		err = filepath.WalkDir(boundary, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			if path != boundary {
				if _, excluded := codeProjectBranchScanExcludedDirs[entry.Name()]; excluded {
					return filepath.SkipDir
				}
			}
			if _, statErr := os.Lstat(filepath.Join(path, ".git")); statErr != nil {
				return nil
			}
			root, gitErr := runCodeGit(path, "rev-parse", "--show-toplevel")
			if gitErr != nil {
				return nil
			}
			root, gitErr = filepath.EvalSymlinks(filepath.Clean(root))
			if gitErr != nil || root != path {
				return nil
			}
			if _, exists := seen[root]; !exists {
				seen[root] = struct{}{}
				repositories = append(repositories, root)
				if len(repositories) > maxCodeProjectBranchRepositories {
					return fmt.Errorf("项目目录中 Git 仓库超过 %d 个，请缩小目录范围", maxCodeProjectBranchRepositories)
				}
			}
			return filepath.SkipDir
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(repositories)
	return repositories, nil
}

func inspectCodeProjectBranchRepository(project *model.AIProject, root string, excludedOverride ...bool) (codeProjectBranchRepository, error) {
	excluded := len(excludedOverride) > 0 && excludedOverride[0]
	if len(excludedOverride) == 0 {
		excluded = project != nil && isCodeRepositoryExcluded(
			root, normalizeCodeExcludedRepositories(project.ExcludedRepositories),
		)
	}
	currentBranch, _ := runCodeGit(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	status, err := runCodeGit(root, "status", "--porcelain")
	if err != nil {
		return codeProjectBranchRepository{}, err
	}
	mergeTarget := codeProjectRepositoryDeliveryBranch(project, root, strings.TrimSpace(currentBranch))
	mergedOutput := ""
	if mergeTarget != "" {
		mergedOutput, _ = runCodeGit(
			root, "for-each-ref", "--merged=refs/heads/"+mergeTarget, "--format=%(refname)", "refs/heads",
		)
	}
	mergedRefs := make(map[string]struct{})
	for _, refName := range strings.Fields(mergedOutput) {
		mergedRefs[refName] = struct{}{}
	}
	output, err := runCodeGit(root, "for-each-ref", "--sort=-committerdate", "--format=%(refname)%00%(refname:short)%00%(objectname)%00%(objectname:short)%00%(subject)%00%(committerdate:iso-strict)%00%(upstream:short)%00%(HEAD)", "refs/heads", "refs/remotes")
	if err != nil {
		return codeProjectBranchRepository{}, err
	}
	branches := make([]codeProjectBranch, 0)
	commitStats := make(map[string][2]int)
	deliveryBranch := codeProjectRepositoryDeliveryBranch(project, root, strings.TrimSpace(currentBranch))
	protected, err := inspectCodeProjectProtectedBranches(root, !excluded)
	if err != nil {
		return codeProjectBranchRepository{}, err
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\x00")
		if len(fields) != 8 || strings.HasSuffix(fields[0], "/HEAD") {
			continue
		}
		scope := "local"
		if strings.HasPrefix(fields[0], "refs/remotes/") {
			scope = "remote"
		}
		_, merged := mergedRefs[fields[0]]
		stats, exists := commitStats[fields[2]]
		if !exists {
			numstat, statErr := runCodeGit(root, "show", "--format=", "--numstat", "--no-renames", fields[2])
			if statErr == nil {
				stats[0], stats[1] = parseCodeGitNumstat(numstat)
			}
			commitStats[fields[2]] = stats
		}
		branch := codeProjectBranch{
			Name: fields[1], Ref: fields[0], Scope: scope, Current: fields[7] == "*",
			Commit: fields[3], Subject: fields[4], UpdatedAt: fields[5], Upstream: fields[6], Merged: merged,
			Managed: strings.HasPrefix(fields[1], "gopanel/code-"), Additions: stats[0], Deletions: stats[1],
		}
		_, branch.TaskBranch = protected.Tasks[branch.Name]
		branch.DeleteBlockReason = codeProjectBranchDeleteBlockReason(deliveryBranch, branch, protected)
		branch.Deletable = branch.DeleteBlockReason == ""
		branches = append(branches, branch)
	}
	changedFiles := 0
	additions, deletions := 0, 0
	if strings.TrimSpace(status) != "" {
		unsavedStats := loadCodeProjectWorktreeStats(root)
		changedFiles = unsavedStats.Files
		additions, deletions = unsavedStats.Additions, unsavedStats.Deletions
		branch := codeProjectRepositoryDeliveryBranch(project, root, strings.TrimSpace(currentBranch))
		_, remoteRef := codeRepositoryRemoteTracking(root, branch)
		if matchesRemote, matchErr := codeGitWorktreeMatchesCommit(root, remoteRef); matchErr == nil && matchesRemote {
			changedFiles, additions, deletions = 0, 0, 0
		}
	}
	return codeProjectBranchRepository{
		Name: filepath.Base(root), Path: root, Excluded: excluded, CurrentBranch: strings.TrimSpace(currentBranch),
		Detached: strings.TrimSpace(currentBranch) == "", Dirty: changedFiles > 0,
		ChangedFiles: changedFiles, Additions: additions, Deletions: deletions, Branches: branches,
	}, nil
}

func codeProjectRepositoryDeliveryBranch(project *model.AIProject, root, currentBranch string) string {
	if project != nil && filepath.Clean(strings.TrimSpace(project.PrimaryRepository)) == filepath.Clean(root) {
		return strings.TrimSpace(project.DeliveryBranch)
	}
	if project != nil && strings.TrimSpace(project.PrimaryRepository) == "" && len(codeProjectSourceDirs(project)) == 1 {
		return strings.TrimSpace(project.DeliveryBranch)
	}
	return strings.TrimSpace(currentBranch)
}
