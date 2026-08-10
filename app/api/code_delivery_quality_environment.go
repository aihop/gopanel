package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func prepareCodeDeliveryQualityEnvironment(roots []codeDeliveryQualityRoot) (func() error, error) {
	cleanups := make([]func() error, 0, len(roots))
	cleanupAll := func() error {
		var cleanupErr error
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanupErr = errors.Join(cleanupErr, cleanups[index]())
		}
		return cleanupErr
	}
	for _, root := range roots {
		cleanup, err := linkCodeDeliveryNodeModules(root)
		if err != nil {
			return func() error { return nil }, errors.Join(err, cleanupAll())
		}
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
		cleanup, err = linkCodeDeliveryDartTools(root)
		if err != nil {
			return func() error { return nil }, errors.Join(err, cleanupAll())
		}
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
	}
	return cleanupAll, nil
}

func linkCodeDeliveryNodeModules(root codeDeliveryQualityRoot) (func() error, error) {
	return linkFirstCodeDeliveryDependency(root, "node_modules", ".gopanel-quality-node_modules-*")
}

func linkCodeDeliveryDartTools(root codeDeliveryQualityRoot) (func() error, error) {
	if strings.TrimSpace(root.WorkDir) == "" {
		return nil, nil
	}
	cleanups := make([]func() error, 0)
	cleanupAll := func() error {
		var cleanupErr error
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanupErr = errors.Join(cleanupErr, cleanups[index]())
		}
		return cleanupErr
	}
	err := filepath.WalkDir(root.WorkDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && current != root.WorkDir && ignoredCodeDeliveryDartDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != "pubspec.yaml" {
			return nil
		}
		packageDir, relativeErr := filepath.Rel(root.WorkDir, filepath.Dir(current))
		if relativeErr != nil {
			return relativeErr
		}
		relativeDartTool := filepath.Join(packageDir, ".dart_tool")
		cleanup, linkErr := linkFirstCodeDeliveryDependency(root, relativeDartTool, ".gopanel-quality-dart_tool-*")
		if linkErr != nil {
			return linkErr
		}
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Join(err, cleanupAll())
	}
	return cleanupAll, nil
}

func linkFirstCodeDeliveryDependency(root codeDeliveryQualityRoot, relativePath, temporaryPattern string) (func() error, error) {
	for _, runtimeDir := range codeDeliveryRuntimeDirs(root) {
		candidate := root
		candidate.RuntimeDir = runtimeDir
		cleanup, err := linkCodeDeliveryDependency(candidate, relativePath, temporaryPattern)
		if err != nil || cleanup != nil {
			return cleanup, err
		}
	}
	return nil, nil
}

func codeDeliveryRuntimeDirs(root codeDeliveryQualityRoot) []string {
	candidates := []string{root.IdentityDir, root.RuntimeDir}
	runtimeDirs := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || filepath.Clean(candidate) == filepath.Clean(root.WorkDir) {
			continue
		}
		cleaned := filepath.Clean(candidate)
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		runtimeDirs = append(runtimeDirs, cleaned)
	}
	return runtimeDirs
}

func ignoredCodeDeliveryDartDirectory(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "build", "node_modules":
		return true
	default:
		return false
	}
}

func linkCodeDeliveryDependency(root codeDeliveryQualityRoot, relativePath, temporaryPattern string) (func() error, error) {
	if strings.TrimSpace(root.RuntimeDir) == "" || strings.TrimSpace(root.WorkDir) == "" {
		return nil, nil
	}
	source := filepath.Join(root.RuntimeDir, relativePath)
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() || filepath.Clean(root.RuntimeDir) == filepath.Clean(root.WorkDir) {
		return nil, nil
	}
	if _, err := runCodeGit(root.WorkDir, "check-ignore", "-q", "--", filepath.ToSlash(relativePath)); err != nil {
		return nil, nil
	}
	destination := filepath.Join(root.WorkDir, relativePath)
	if _, err := os.Stat(filepath.Dir(destination)); err != nil {
		return nil, nil
	}
	backup := ""
	if _, err := os.Lstat(destination); err == nil {
		placeholder, createErr := os.CreateTemp(filepath.Dir(destination), temporaryPattern)
		if createErr != nil {
			return nil, createErr
		}
		backup = placeholder.Name()
		if closeErr := placeholder.Close(); closeErr != nil {
			_ = os.Remove(backup)
			return nil, closeErr
		}
		if removeErr := os.Remove(backup); removeErr != nil {
			return nil, removeErr
		}
		if renameErr := os.Rename(destination, backup); renameErr != nil {
			return nil, renameErr
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.Symlink(source, destination); err != nil {
		if backup != "" {
			_ = os.Rename(backup, destination)
		}
		return nil, err
	}
	return func() error {
		removeErr := os.Remove(destination)
		if backup == "" {
			return removeErr
		}
		return errors.Join(removeErr, os.Rename(backup, destination))
	}, nil
}
