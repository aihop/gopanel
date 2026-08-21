package service

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

var errFileAccessDenied = buserr.New(constant.ErrFileAccessDenied)

// ValidatePathWithinBase checks both directory boundaries and symlink targets.
// Non-existent paths are resolved from their nearest existing ancestor.
func ValidatePathWithinBase(baseDir, targetPath string) error {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(targetPath) == "" {
		return errFileAccessDenied
	}
	base, err := canonicalFilePath(baseDir)
	if err != nil {
		return errFileAccessDenied
	}
	target, err := canonicalFilePath(targetPath)
	if err != nil {
		return errFileAccessDenied
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errFileAccessDenied
	}
	return nil
}

func canonicalFilePath(path string) (string, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}

	current := absPath
	var missing []string
	for {
		resolved, evalErr := filepath.EvalSymlinks(current)
		if evalErr == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(evalErr) {
			return "", evalErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", evalErr
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
