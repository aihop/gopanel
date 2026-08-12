package api

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxCodeDiscoveredRepositories = 50

var codeGitScanExcludedDirectories = map[string]struct{}{
	".git": {}, ".cache": {}, ".claude": {}, ".codex": {}, ".next": {},
	".nuxt": {}, ".output": {}, ".pnpm-store": {}, ".tmp": {}, ".turbo": {},
	".venv": {}, "build": {}, "coverage": {}, "dist": {}, "node_modules": {},
	"target": {}, "vendor": {},
}

func discoverCodeRepositoryRoots(sourceDirs []string) ([]string, error) {
	seen := make(map[string]struct{})
	repositories := make([]string, 0, len(sourceDirs))
	for _, sourceDir := range sourceDirs {
		boundary, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
		if err != nil {
			return nil, fmt.Errorf("项目目录不可访问：%s", sourceDir)
		}
		if root, ordinary := ordinaryCodeRepositoryRoot(boundary); ordinary && root == boundary {
			if err := appendCodeRepositoryWithGitlinks(root, &repositories, seen); err != nil {
				return nil, err
			}
			continue
		}
		if err := discoverCodeRepositoryRootsWithin(boundary, &repositories, seen); err != nil {
			return nil, err
		}
	}
	sort.Strings(repositories)
	return repositories, nil
}

func discoverCodeRepositoryRootsWithin(boundary string, repositories *[]string, seen map[string]struct{}) error {
	return filepath.WalkDir(boundary, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if path != boundary {
			if _, excluded := codeGitScanExcludedDirectories[entry.Name()]; excluded {
				return filepath.SkipDir
			}
		}
		if _, err := os.Lstat(filepath.Join(path, ".git")); err != nil {
			return nil
		}
		root, ordinary := ordinaryCodeRepositoryRoot(path)
		if !ordinary || root != path {
			return filepath.SkipDir
		}
		if err := appendCodeRepositoryWithGitlinks(root, repositories, seen); err != nil {
			return err
		}
		return filepath.SkipDir
	})
}

func ordinaryCodeRepositoryRoot(path string) (string, bool) {
	root, err := runCodeGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", false
	}
	root, err = filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return "", false
	}
	gitDir, err := runCodeGit(path, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", false
	}
	commonDir, err := runCodeGit(path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", false
	}
	gitDir, gitErr := filepath.EvalSymlinks(filepath.Clean(gitDir))
	commonDir, commonErr := filepath.EvalSymlinks(filepath.Clean(commonDir))
	return root, gitErr == nil && commonErr == nil && gitDir == commonDir
}

func appendCodeRepositoryWithGitlinks(root string, repositories *[]string, seen map[string]struct{}) error {
	if _, exists := seen[root]; exists {
		return nil
	}
	if err := ensureCodeGitLocalExcludes(root); err != nil {
		return fmt.Errorf("配置仓库 %s 的 Code 默认忽略规则失败：%w", filepath.Base(root), err)
	}
	seen[root] = struct{}{}
	*repositories = append(*repositories, root)
	if len(*repositories) > maxCodeDiscoveredRepositories {
		return fmt.Errorf("项目目录中 Git 仓库超过 %d 个，请缩小目录范围", maxCodeDiscoveredRepositories)
	}
	gitlinks, err := codeRepositoryGitlinkPaths(root)
	if err != nil {
		return err
	}
	for _, gitlinkPath := range gitlinks {
		childPath, resolveErr := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(gitlinkPath)))
		if resolveErr != nil || !isPathInside(childPath, root) {
			continue
		}
		childRoot, ordinary := ordinaryCodeRepositoryRoot(childPath)
		if !ordinary || childRoot != childPath {
			continue
		}
		if err := appendCodeRepositoryWithGitlinks(childRoot, repositories, seen); err != nil {
			return err
		}
	}
	return nil
}

func codeRepositoryGitlinkPaths(sourceDir string) ([]string, error) {
	entries, err := runCodeGit(sourceDir, "ls-files", "-s", "-z")
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, line := range strings.Split(entries, "\x00") {
		metadata, path, found := strings.Cut(line, "\t")
		if found && strings.HasPrefix(metadata, "160000 ") && strings.TrimSpace(path) != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}
