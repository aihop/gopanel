package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const codeGitLocalExcludeHeader = "# GoPanel Code defaults"

var (
	codeGitLocalExcludeMu       sync.Mutex
	codeGitLocalExcludePatterns = []string{".DS_Store"}
)

func ensureCodeGitLocalExcludes(repositoryDir string) error {
	commonDir, err := runCodeGit(repositoryDir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	excludePath := filepath.Join(filepath.Clean(commonDir), "info", "exclude")
	codeGitLocalExcludeMu.Lock()
	defer codeGitLocalExcludeMu.Unlock()

	content, err := os.ReadFile(excludePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	missing := missingCodeGitLocalExcludePatterns(string(content))
	if len(missing) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0750); err != nil {
		return err
	}
	file, err := os.OpenFile(excludePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	prefix := ""
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		prefix = "\n"
	}
	_, writeErr := file.WriteString(prefix + codeGitLocalExcludeHeader + "\n" + strings.Join(missing, "\n") + "\n")
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func missingCodeGitLocalExcludePatterns(content string) []string {
	existing := make(map[string]struct{})
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			existing[line] = struct{}{}
		}
	}
	missing := make([]string, 0, len(codeGitLocalExcludePatterns))
	for _, pattern := range codeGitLocalExcludePatterns {
		if _, found := existing[pattern]; !found {
			missing = append(missing, pattern)
		}
	}
	return missing
}
