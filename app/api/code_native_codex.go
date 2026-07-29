package api

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

func buildNativeCodexCommand(session *model.AIDevSession) (*exec.Cmd, error) {
	commandPath, commandEnv, err := resolveCodeExecutorCommand("codex")
	if err != nil {
		return nil, err
	}
	args := []string{
		"--ask-for-approval", codexApprovalPolicy(session.ApprovalPolicy),
		"--sandbox", "workspace-write",
		"--no-alt-screen",
		"--cd", session.WorkDir,
	}
	if strings.TrimSpace(session.NativeSessionID) != "" {
		args = append(args, "resume", session.NativeSessionID)
	}
	command := exec.Command(commandPath, args...)
	command.Dir = session.WorkDir
	command.Env = commandEnv
	if err := configureCodexCommand(command, session); err != nil {
		return nil, err
	}
	return command, nil
}

func discoverNativeCodexSession(session *model.AIDevSession, processID int, startedAt time.Time) {
	for attempt := 0; attempt < 20; attempt++ {
		if nativeSessionID := findNativeCodexSessionID(session.WorkDir, processID, startedAt); nativeSessionID != "" {
			current, err := repo.NewAIDevSessionRepo().GetSessionByID(session.ID)
			if err == nil && current.NativeSessionID != nativeSessionID {
				current.NativeSessionID = nativeSessionID
				_ = repo.NewAIDevSessionRepo().UpdateSession(current)
			}
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func findNativeCodexSessionID(workDir string, processID int, startedAt time.Time) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	pattern := filepath.Join(homeDir, ".codex", "sessions", "*", "*", "*", "*.jsonl")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return ""
	}
	cleanWorkDir := filepath.Clean(workDir)
	var latestID string
	var latestTime time.Time
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil || info.ModTime().Before(startedAt.Add(-2*time.Second)) || info.ModTime().Before(latestTime) {
			continue
		}
		sessionID, cwd, originPID := readCodexSessionMeta(path)
		if sessionID == "" || filepath.Clean(cwd) != cleanWorkDir {
			continue
		}
		if originPID > 0 && processID > 0 && originPID != processID {
			continue
		}
		latestID = sessionID
		latestTime = info.ModTime()
	}
	return latestID
}

func readCodexSessionMeta(path string) (string, string, int) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", 0
	}
	defer file.Close()
	var event struct {
		Type    string `json:"type"`
		Payload struct {
			SessionID string `json:"session_id"`
			Cwd       string `json:"cwd"`
			OriginPID int    `json:"originator_pid"`
		} `json:"payload"`
	}
	if json.NewDecoder(file).Decode(&event) != nil || event.Type != "session_meta" {
		return "", "", 0
	}
	return event.Payload.SessionID, event.Payload.Cwd, event.Payload.OriginPID
}
