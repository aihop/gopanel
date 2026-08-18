package api

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

const codeRuntimeProgressFileLimit = 100

const codeRuntimeGitCacheTTL = 2 * time.Second

var codeRuntimeGitCache = struct {
	sync.Mutex
	items map[uint]codeRuntimeGitCacheItem
}{items: make(map[uint]codeRuntimeGitCacheItem)}

type codeRuntimeGitCacheItem struct {
	files        []string
	changedFiles int
	additions    int
	deletions    int
	available    bool
	loadedAt     time.Time
}

func loadCodeRuntimeProgress(session *model.AIDevSession, plan *codeRuntimeProgress) *codeRuntimeProgress {
	progress := plan
	if progress == nil {
		progress = &codeRuntimeProgress{Source: "git", UpdatedAt: time.Now()}
	}
	files, changedFiles, additions, deletions, available := loadCodeRuntimeGitChanges(session)
	if !available && plan == nil {
		return nil
	}
	progress.ChangedFiles = changedFiles
	progress.Additions = additions
	progress.Deletions = deletions
	progress.Files = files
	if plan != nil && available {
		progress.Source = "codex_plan+git"
	}
	return progress
}

func loadCodeRuntimeGitChanges(session *model.AIDevSession) ([]string, int, int, int, bool) {
	if session == nil {
		return nil, 0, 0, 0, false
	}
	if cached, ok := getCodeRuntimeGitCache(session.ID); ok {
		return cached.files, cached.changedFiles, cached.additions, cached.deletions, cached.available
	}
	sourceDirs := []string{session.WorkDir}
	var excluded []string
	if session.ProjectID > 0 && global.DB != nil {
		var project model.AIProject
		if err := global.DB.Select("source_dirs", "excluded_repositories", "work_dir").First(&project, session.ProjectID).Error; err == nil {
			sourceDirs, excluded = project.SourceDirs, project.ExcludedRepositories
			if len(sourceDirs) == 0 && strings.TrimSpace(project.WorkDir) != "" {
				sourceDirs = []string{project.WorkDir}
			}
		}
	}
	repositories := discoverCodeGitRepositories(session, sourceDirs, excluded)
	if len(repositories) == 0 {
		storeCodeRuntimeGitCache(session.ID, codeRuntimeGitCacheItem{})
		return nil, 0, 0, 0, false
	}
	paths := make(map[string]struct{})
	additions, deletions := 0, 0
	for _, repository := range repositories {
		loaded, err := loadCodeGitRepositoryStatus(repository)
		if err != nil {
			continue
		}
		if repository.BaseCommit != "" {
			nameStatus, _, diffErr := runCodeGitReviewCommand(repository.root, false, 4*codeGitDiffOutputLimit,
				"--literal-pathspecs", "diff", "--name-status", "-z", "--find-renames", repository.BaseCommit)
			if diffErr == nil {
				for _, file := range parseCodeGitResultFiles(nameStatus, repository.workspacePrefix) {
					paths[file.WorkspacePath] = struct{}{}
				}
				numstat, _, _ := runCodeGitReviewCommand(repository.root, false, codeGitDiffOutputLimit,
					"--literal-pathspecs", "diff", "--numstat", repository.BaseCommit)
				added, deleted := parseCodeGitNumstat(numstat)
				additions, deletions = additions+added, deletions+deleted
				for _, file := range loaded.Files {
					if file.Untracked {
						paths[file.WorkspacePath] = struct{}{}
					}
				}
				continue
			}
		}
		for _, file := range loaded.Files {
			paths[file.WorkspacePath] = struct{}{}
		}
		numstat, _, _ := runCodeGitReviewCommand(repository.root, false, codeGitDiffOutputLimit,
			"--literal-pathspecs", "diff", "--numstat", "HEAD")
		added, deleted := parseCodeGitNumstat(numstat)
		additions, deletions = additions+added, deletions+deleted
	}
	files := make([]string, 0, len(paths))
	for file := range paths {
		files = append(files, filepath.ToSlash(file))
	}
	sort.Strings(files)
	changedFiles := len(files)
	if len(files) > codeRuntimeProgressFileLimit {
		files = files[:codeRuntimeProgressFileLimit]
	}
	result := codeRuntimeGitCacheItem{
		files: files, changedFiles: changedFiles, additions: additions, deletions: deletions, available: true,
	}
	storeCodeRuntimeGitCache(session.ID, result)
	return files, changedFiles, additions, deletions, true
}

func getCodeRuntimeGitCache(sessionID uint) (codeRuntimeGitCacheItem, bool) {
	if sessionID == 0 {
		return codeRuntimeGitCacheItem{}, false
	}
	codeRuntimeGitCache.Lock()
	defer codeRuntimeGitCache.Unlock()
	item, exists := codeRuntimeGitCache.items[sessionID]
	if !exists || time.Since(item.loadedAt) >= codeRuntimeGitCacheTTL {
		delete(codeRuntimeGitCache.items, sessionID)
		return codeRuntimeGitCacheItem{}, false
	}
	item.files = append([]string(nil), item.files...)
	return item, true
}

func storeCodeRuntimeGitCache(sessionID uint, item codeRuntimeGitCacheItem) {
	if sessionID == 0 {
		return
	}
	item.loadedAt = time.Now()
	item.files = append([]string(nil), item.files...)
	codeRuntimeGitCache.Lock()
	codeRuntimeGitCache.items[sessionID] = item
	codeRuntimeGitCache.Unlock()
}
