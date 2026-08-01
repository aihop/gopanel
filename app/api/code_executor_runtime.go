package api

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func resolveCodeExecutorCommand(command string) (string, []string, error) {
	searchDirs := codeExecutorSearchDirs()
	if commandPath, err := exec.LookPath(command); err == nil {
		return commandPath, codeExecutorEnvironment(commandPath, searchDirs), nil
	}
	for _, dir := range searchDirs {
		commandPath, err := exec.LookPath(filepath.Join(dir, command))
		if err == nil {
			return commandPath, codeExecutorEnvironment(commandPath, searchDirs), nil
		}
	}
	return "", nil, exec.ErrNotFound
}

func codeExecutorSearchDirs() []string {
	dirs := make([]string, 0, 24)
	seen := make(map[string]struct{})
	appendDir := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || !filepath.IsAbs(dir) {
			return
		}
		dir = filepath.Clean(dir)
		if _, exists := seen[dir]; exists {
			return
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		appendDir(dir)
	}
	for _, dir := range []string{
		os.Getenv("NVM_BIN"),
		filepath.Join(os.Getenv("VOLTA_HOME"), "bin"),
		os.Getenv("PNPM_HOME"),
		filepath.Join(os.Getenv("NPM_CONFIG_PREFIX"), "bin"),
		filepath.Join(os.Getenv("HOMEBREW_PREFIX"), "bin"),
	} {
		appendDir(dir)
	}
	homeDir := codeExecutorHomeDir()
	for _, relativeDir := range []string{
		"sdk/node/bin",
		"sdk/go/bin",
		".local/bin",
		".npm-global/bin",
		".n/bin",
		".nodebrew/current/bin",
		".volta/bin",
		".asdf/shims",
		".local/share/mise/shims",
		".local/share/pnpm",
		"Library/pnpm",
		".bun/bin",
	} {
		appendDir(filepath.Join(homeDir, relativeDir))
	}
	for _, pattern := range []string{
		filepath.Join(homeDir, ".nvm", "versions", "node", "*", "bin"),
		filepath.Join(homeDir, ".local", "share", "fnm", "node-versions", "*", "installation", "bin"),
		filepath.Join(homeDir, "Library", "Application Support", "fnm", "node-versions", "*", "installation", "bin"),
	} {
		matches, _ := filepath.Glob(pattern)
		for index := len(matches) - 1; index >= 0; index-- {
			appendDir(matches[index])
		}
	}
	for _, dir := range []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/home/linuxbrew/.linuxbrew/bin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	} {
		appendDir(dir)
	}
	return dirs
}

func codeExecutorEnvironment(commandPath string, searchDirs []string) []string {
	pathDirs := append([]string{filepath.Dir(commandPath)}, searchDirs...)
	env := upsertEnvironment(os.Environ(), "PATH", strings.Join(uniqueStrings(pathDirs), string(os.PathListSeparator)))
	if homeDir := codeExecutorHomeDir(); homeDir != "" && strings.TrimSpace(os.Getenv("HOME")) == "" {
		env = upsertEnvironment(env, "HOME", homeDir)
	}
	return env
}

func codeExecutorHomeDir() string {
	if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
		return homeDir
	}
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(currentUser.HomeDir)
}

func upsertEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	for index, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[index] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
