package api

import (
	"errors"
	"os"
	"path/filepath"
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
	}
	return cleanupAll, nil
}

func linkCodeDeliveryNodeModules(root codeDeliveryQualityRoot) (func() error, error) {
	source := filepath.Join(root.RuntimeDir, "node_modules")
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() || filepath.Clean(root.RuntimeDir) == filepath.Clean(root.WorkDir) {
		return nil, nil
	}
	if _, err := runCodeGit(root.WorkDir, "check-ignore", "-q", "--", "node_modules"); err != nil {
		return nil, nil
	}
	destination := filepath.Join(root.WorkDir, "node_modules")
	backup := ""
	if _, err := os.Lstat(destination); err == nil {
		placeholder, createErr := os.CreateTemp(root.WorkDir, ".gopanel-quality-node_modules-*")
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
