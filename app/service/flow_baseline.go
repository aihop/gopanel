package service

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

const maxFlowBaselineRepositories = 50

var flowGitScanExcludedDirectories = map[string]struct{}{
	".git": {}, ".cache": {}, ".claude": {}, ".codex": {}, ".next": {},
	".nuxt": {}, ".output": {}, ".pnpm-store": {}, ".tmp": {}, ".turbo": {},
	".venv": {}, "build": {}, "coverage": {}, "dist": {}, "node_modules": {},
	"target": {}, "vendor": {},
}

type FlowCodeBaselineSource struct {
	Available             bool                         `json:"available"`
	SourceDigest          string                       `json:"sourceDigest,omitempty"`
	HasUncommittedChanges bool                         `json:"hasUncommittedChanges"`
	Repositories          []model.FlowSourceRepository `json:"repositories"`
}

func (s *FlowRunApplicationService) CodeBaselineSource(flowID, userID uint, includeAll bool) (*FlowCodeBaselineSource, error) {
	flow, err := s.repo.Get(flowID)
	if err != nil {
		return nil, buserr.New(constant.ErrFlowNotFound)
	}
	if !includeAll && flow.CreatedBy != userID {
		return nil, buserr.New(constant.ErrFlowForbidden)
	}
	pipeline, err := repoPipeline(s.db, flow.PipelineID)
	if err != nil {
		return nil, buserr.New(constant.ErrFlowPipelineNotFound)
	}
	if pipelineSourceType(pipeline) != "code" || pipeline.CodeProjectID != flow.ProjectID {
		return &FlowCodeBaselineSource{Repositories: []model.FlowSourceRepository{}}, nil
	}
	manifest, digest, dirty, available, err := s.resolveFlowProjectBaseline(flow.ProjectID)
	if err != nil {
		return nil, err
	}
	result := &FlowCodeBaselineSource{Available: available, Repositories: []model.FlowSourceRepository{}}
	if available {
		result.SourceDigest = digest
		result.HasUncommittedChanges = dirty
		result.Repositories = flowPublicSourceRepositories(manifest)
	}
	return result, nil
}

func (s *FlowRunApplicationService) resolveFlowProjectBaseline(projectID uint) (flowSourceManifest, string, bool, bool, error) {
	var project model.AIProject
	if err := s.db.First(&project, projectID).Error; err != nil {
		return flowSourceManifest{}, "", false, false, buserr.New(constant.ErrFlowProjectNotFound)
	}
	var jobs []model.AICodeDeliveryJob
	if err := s.db.Where("project_id = ? AND status = ?", projectID, "completed").Find(&jobs).Error; err != nil {
		return flowSourceManifest{}, "", false, false, err
	}
	for index := range jobs {
		if _, _, err := s.resolveFlowCodeDeliveryManifest(&project, &jobs[index]); err == nil {
			return flowSourceManifest{}, "", false, false, nil
		}
	}
	manifest, digest, dirty, err := resolveFlowProjectBaselineManifest(&project)
	if err != nil {
		return flowSourceManifest{}, "", false, false, buserr.New(constant.ErrFlowCodeBaselineInvalid)
	}
	return manifest, digest, dirty, true, nil
}

func resolveFlowProjectBaselineManifest(project *model.AIProject) (flowSourceManifest, string, bool, error) {
	if project == nil || len(project.SourceDirs) == 0 {
		return flowSourceManifest{}, "", false, buserr.New(constant.ErrFlowProjectNoSource)
	}
	repositories, err := discoverFlowGitRepositories(project.SourceDirs)
	if err != nil {
		return flowSourceManifest{}, "", false, err
	}
	manifest := flowSourceManifest{
		SchemaVersion: flowSourceManifestSchemaVersion,
		SourceType:    "code_baseline",
		Repositories:  make([]flowSourceManifestRepository, 0, len(repositories)),
	}
	dirty := false
	usedPaths := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		if flowRepositoryExcluded(repository, project.ExcludedRepositories) {
			continue
		}
		commit, err := flowGitCommand(repository, "rev-parse", "HEAD")
		if err != nil {
			return flowSourceManifest{}, "", false, err
		}
		commit, err = normalizePipelineExpectedCommit(commit)
		if err != nil || commit == "" {
			return flowSourceManifest{}, "", false, buserr.New(constant.ErrFlowProjectRepoHeadInvalid)
		}
		branch, _ := flowGitCommand(repository, "branch", "--show-current")
		_, workspacePath, err := flowRepositoryWorkspacePath(project.SourceDirs, repository)
		if err != nil {
			return flowSourceManifest{}, "", false, err
		}
		if _, exists := usedPaths[workspacePath]; exists {
			return flowSourceManifest{}, "", false, buserr.New(constant.ErrFlowProjectRepoDuplicate)
		}
		usedPaths[workspacePath] = struct{}{}
		status, statusErr := flowGitCommand(repository, "status", "--porcelain", "--ignore-submodules=dirty")
		if statusErr != nil {
			return flowSourceManifest{}, "", false, statusErr
		}
		dirty = dirty || strings.TrimSpace(status) != ""
		manifest.Repositories = append(manifest.Repositories, flowSourceManifestRepository{
			Name: filepath.Base(repository), SourceDir: repository, WorkspacePath: workspacePath,
			TargetBranch: strings.TrimSpace(branch), Commit: commit,
		})
	}
	if len(manifest.Repositories) == 0 {
		return flowSourceManifest{}, "", false, buserr.New(constant.ErrFlowProjectNoGitRepo)
	}
	sort.Slice(manifest.Repositories, func(left, right int) bool {
		return manifest.Repositories[left].WorkspacePath < manifest.Repositories[right].WorkspacePath
	})
	digest, err := flowSourceManifestDigest(manifest)
	return manifest, digest, dirty, err
}

func discoverFlowGitRepositories(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{})
	repositories := make([]string, 0, len(sourceDirs))
	for _, rawSource := range sourceDirs {
		boundary, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(rawSource)))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(boundary)
		if err != nil || !info.IsDir() {
			return nil, buserr.New(constant.ErrFlowProjectSourceInaccessible)
		}
		if root, ordinary := flowOrdinaryGitRoot(boundary); ordinary && root == boundary {
			if err := appendFlowGitRepository(root, &repositories, seen); err != nil {
				return nil, err
			}
			continue
		}
		if err := filepath.WalkDir(boundary, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			if path != boundary {
				if _, excluded := flowGitScanExcludedDirectories[entry.Name()]; excluded {
					return filepath.SkipDir
				}
			}
			if _, err := os.Lstat(filepath.Join(path, ".git")); err != nil {
				return nil
			}
			root, ordinary := flowOrdinaryGitRoot(path)
			if !ordinary || root != path {
				return filepath.SkipDir
			}
			if err := appendFlowGitRepository(root, &repositories, seen); err != nil {
				return err
			}
			return filepath.SkipDir
		}); err != nil {
			return nil, err
		}
	}
	sort.Strings(repositories)
	return repositories, nil
}

func appendFlowGitRepository(root string, repositories *[]string, seen map[string]struct{}) error {
	if _, exists := seen[root]; exists {
		return nil
	}
	seen[root] = struct{}{}
	*repositories = append(*repositories, root)
	if len(*repositories) > maxFlowBaselineRepositories {
		return buserr.WithMap(constant.ErrFlowProjectGitRepoExceeded, map[string]interface{}{"max": maxFlowBaselineRepositories})
	}
	entries, err := flowGitCommand(root, "ls-files", "-s", "-z")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(entries, "\x00") {
		metadata, path, found := strings.Cut(line, "\t")
		if !found || !strings.HasPrefix(metadata, "160000 ") || strings.TrimSpace(path) == "" {
			continue
		}
		child, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || !codeSnapshotPathWithin(root, child) {
			continue
		}
		childRoot, ordinary := flowOrdinaryGitRoot(child)
		if ordinary && childRoot == child {
			if err := appendFlowGitRepository(childRoot, repositories, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func flowOrdinaryGitRoot(path string) (string, bool) {
	root, err := flowGitCommand(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", false
	}
	root, err = filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return "", false
	}
	gitDir, gitErr := flowGitCommand(path, "rev-parse", "--absolute-git-dir")
	commonDir, commonErr := flowGitCommand(path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if gitErr != nil || commonErr != nil {
		return "", false
	}
	gitDir, gitErr = filepath.EvalSymlinks(filepath.Clean(gitDir))
	commonDir, commonErr = filepath.EvalSymlinks(filepath.Clean(commonDir))
	return root, gitErr == nil && commonErr == nil && gitDir == commonDir
}

func flowRepositoryExcluded(repository string, excluded []string) bool {
	repository = filepath.Clean(repository)
	for _, raw := range excluded {
		item := filepath.Clean(strings.TrimSpace(raw))
		if resolved, err := filepath.EvalSymlinks(item); err == nil {
			item = filepath.Clean(resolved)
		}
		if item != "." && (repository == item || codeSnapshotPathWithin(item, repository)) {
			return true
		}
	}
	return false
}

func flowGitCommand(repository string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repository
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", buserr.WithDetail(constant.ErrFlowGitOperationFailed, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
