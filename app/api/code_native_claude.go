package api

import (
	"os"
	"path/filepath"
	"strings"
)

func nativeClaudeSessionExists(nativeSessionID string, commandEnv []string) bool {
	nativeSessionID = strings.TrimSpace(nativeSessionID)
	if nativeSessionID == "" || filepath.Base(nativeSessionID) != nativeSessionID {
		return false
	}
	configDir := environmentValueFrom(commandEnv, "CLAUDE_CONFIG_DIR")
	if configDir == "" {
		homeDir := environmentValueFrom(commandEnv, "HOME")
		if homeDir == "" {
			homeDir = codeExecutorHomeDir()
		}
		configDir = filepath.Join(homeDir, ".claude")
	}
	paths, err := filepath.Glob(filepath.Join(configDir, "projects", "*", nativeSessionID+".jsonl"))
	if err != nil {
		return false
	}
	for _, path := range paths {
		if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() && info.Size() > 0 {
			return true
		}
	}
	return false
}

func environmentValueFrom(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(item, prefix))
		}
	}
	return ""
}
