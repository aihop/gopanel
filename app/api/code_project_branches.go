package api

import (
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
	Name      string `json:"name"`
	Ref       string `json:"ref"`
	Scope     string `json:"scope"`
	Current   bool   `json:"current"`
	Upstream  string `json:"upstream,omitempty"`
	Commit    string `json:"commit"`
	Subject   string `json:"subject"`
	UpdatedAt string `json:"updatedAt"`
	Merged    bool   `json:"merged"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type codeProjectBranchRepository struct {
	Name          string              `json:"name"`
	Path          string              `json:"path"`
	CurrentBranch string              `json:"currentBranch,omitempty"`
	Detached      bool                `json:"detached"`
	Dirty         bool                `json:"dirty"`
	ChangedFiles  int                 `json:"changedFiles"`
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
	project, err := repo.NewAIGroupRepo().GetGroupByID(uint(projectID))
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

func inspectCodeProjectBranches(project *model.AIGroup) (codeProjectBranches, error) {
	sourceDirs := project.SourceDirs
	if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
		sourceDirs = aiProjectWorkspaceSourceDirs(project.WorkDir)
		if len(sourceDirs) == 0 {
			sourceDirs = []string{project.WorkDir}
		}
	}
	repositoryRoots, err := discoverCodeProjectBranchRepositories(sourceDirs)
	if err != nil {
		return codeProjectBranches{}, err
	}
	result := codeProjectBranches{Repositories: make([]codeProjectBranchRepository, 0, len(repositoryRoots))}
	for _, root := range repositoryRoots {
		repository, inspectErr := inspectCodeProjectBranchRepository(root)
		if inspectErr != nil {
			return codeProjectBranches{}, inspectErr
		}
		result.TotalBranches += len(repository.Branches)
		result.Repositories = append(result.Repositories, repository)
	}
	return result, nil
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

func inspectCodeProjectBranchRepository(root string) (codeProjectBranchRepository, error) {
	currentBranch, _ := runCodeGit(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	status, err := runCodeGit(root, "status", "--porcelain")
	if err != nil {
		return codeProjectBranchRepository{}, err
	}
	mergedOutput, _ := runCodeGit(root, "for-each-ref", "--merged=HEAD", "--format=%(refname)", "refs/heads")
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
		branches = append(branches, codeProjectBranch{
			Name: fields[1], Ref: fields[0], Scope: scope, Current: fields[7] == "*",
			Commit: fields[3], Subject: fields[4], UpdatedAt: fields[5], Upstream: fields[6], Merged: merged,
			Additions: stats[0], Deletions: stats[1],
		})
	}
	changedFiles := 0
	if strings.TrimSpace(status) != "" {
		changedFiles = len(strings.Split(strings.TrimSpace(status), "\n"))
	}
	return codeProjectBranchRepository{
		Name: filepath.Base(root), Path: root, CurrentBranch: strings.TrimSpace(currentBranch),
		Detached: strings.TrimSpace(currentBranch) == "", Dirty: changedFiles > 0,
		ChangedFiles: changedFiles, Branches: branches,
	}, nil
}
