package api

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

const (
	nativeCodexDiscoveryAttempts = 240
	nativeCodexStartTolerance    = 5 * time.Second
)

type nativeCodexSessionMeta struct {
	ID        string
	WorkDir   string
	ProcessID int
	StartedAt time.Time
}

func buildNativeCodexCommand(session *model.AIDevSession) (*exec.Cmd, error) {
	commandPath, commandEnv, err := resolveCodeExecutorCommand("codex")
	if err != nil {
		return nil, err
	}
	args := append(codexSandboxArgs(session.ApprovalPolicy),
		"--no-alt-screen",
		"--cd", session.WorkDir,
	)
	writableDirs, err := codexWritableDirsForSessionWithRepair(session)
	if err != nil {
		return nil, err
	}
	args = addCodexWritableDirArgs(args, writableDirs)
	if strings.TrimSpace(session.NativeSessionID) != "" {
		args = append(args, "resume", session.NativeSessionID)
	}
	command := exec.Command(commandPath, args...)
	command.Dir = session.WorkDir
	command.Env = commandEnv
	if err := configureCodeProviderCommand("codex", command, session); err != nil {
		return nil, err
	}
	return command, nil
}

func discoverNativeCodexSession(session *model.AIDevSession, processID int, startedAt time.Time) {
	for attempt := 0; attempt < nativeCodexDiscoveryAttempts; attempt++ {
		if nativeSessionID := findNativeCodexSessionID(session.WorkDir, processID, startedAt); nativeSessionID != "" {
			_ = bindNativeCodexSession(session, nativeSessionID)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func repairNativeCodexSessionBinding(session *model.AIDevSession) error {
	if session == nil || strings.TrimSpace(session.NativeSessionID) != "" || session.CreatedAt.IsZero() {
		return nil
	}
	// 交付完成后 session.WorkDir 会被改写成源仓路径，而 rollout 里记录的是
	// 当初运行 codex 的隔离 Worktree 目录，只用 WorkDir 匹配会永久失联。
	// 隔离目录是由 用户+会话号 确定性推导出来的，这里补一次回溯。
	candidates := []string{session.WorkDir}
	if session.ID != 0 && session.UserID != 0 {
		if worktreeDir := aiSessionWorktreeDir(session.UserID, session.ID); worktreeDir != "" {
			candidates = append(candidates, worktreeDir)
		}
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if nativeSessionID := findNativeCodexSessionID(candidate, 0, session.CreatedAt); nativeSessionID != "" {
			return bindNativeCodexSession(session, nativeSessionID)
		}
	}
	return nil
}

func bindNativeCodexSession(session *model.AIDevSession, nativeSessionID string) error {
	if session == nil || session.ID == 0 || strings.TrimSpace(nativeSessionID) == "" {
		return nil
	}
	result := global.DB.Model(&model.AIDevSession{}).
		Where("id = ? AND (native_session_id = '' OR native_session_id IS NULL)", session.ID).
		Update("native_session_id", nativeSessionID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		session.NativeSessionID = nativeSessionID
		return nil
	}
	return global.DB.Model(&model.AIDevSession{}).Where("id = ?", session.ID).
		Pluck("native_session_id", &session.NativeSessionID).Error
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
	var closestDifference time.Duration
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		meta := readCodexSessionMeta(path)
		if meta.ID == "" || filepath.Clean(meta.WorkDir) != cleanWorkDir {
			continue
		}
		if meta.ProcessID > 0 && processID > 0 && meta.ProcessID != processID {
			continue
		}
		candidateTime := meta.StartedAt
		if candidateTime.IsZero() {
			candidateTime = info.ModTime()
		}
		difference := candidateTime.Sub(startedAt)
		if difference < 0 {
			difference = -difference
		}
		if difference > nativeCodexStartTolerance || latestID != "" && difference >= closestDifference {
			continue
		}
		latestID = meta.ID
		closestDifference = difference
	}
	return latestID
}

func readCodexSessionMeta(path string) nativeCodexSessionMeta {
	file, err := os.Open(path)
	if err != nil {
		return nativeCodexSessionMeta{}
	}
	defer file.Close()
	var event struct {
		Type    string    `json:"type"`
		Time    time.Time `json:"timestamp"`
		Payload struct {
			SessionID string    `json:"session_id"`
			Cwd       string    `json:"cwd"`
			OriginPID int       `json:"originator_pid"`
			Timestamp time.Time `json:"timestamp"`
		} `json:"payload"`
	}
	if json.NewDecoder(file).Decode(&event) != nil || event.Type != "session_meta" {
		return nativeCodexSessionMeta{}
	}
	startedAt := event.Payload.Timestamp
	if startedAt.IsZero() {
		startedAt = event.Time
	}
	return nativeCodexSessionMeta{
		ID: event.Payload.SessionID, WorkDir: event.Payload.Cwd, ProcessID: event.Payload.OriginPID, StartedAt: startedAt,
	}
}
