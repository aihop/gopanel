package api

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

func discoverNativeOpenCodeSession(session *model.AIDevSession, commandPath string, commandEnv []string, startedAt time.Time, done <-chan struct{}) {
	for attempt := 0; attempt < 120; attempt++ {
		if nativeSessionID, err := findNativeOpenCodeSessionInDatabase(session.WorkDir, startedAt); err == nil && nativeSessionID != "" {
			_ = bindNativeOpenCodeSession(session, nativeSessionID)
			return
		}
		if attempt%10 == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			command := exec.CommandContext(ctx, commandPath, "session", "list", "--format", "json", "--max-count", "20")
			command.Env = commandEnv
			output, err := command.Output()
			cancel()
			if err == nil {
				if nativeSessionID := findNativeOpenCodeSessionID(output, session.WorkDir, startedAt); nativeSessionID != "" {
					_ = bindNativeOpenCodeSession(session, nativeSessionID)
					return
				}
			}
		}
		select {
		case <-done:
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func repairNativeOpenCodeSessionBinding(session *model.AIDevSession) error {
	if session == nil || strings.TrimSpace(session.NativeSessionID) != "" || session.CreatedAt.IsZero() {
		return nil
	}
	candidates := []string{session.WorkDir}
	if session.ID != 0 && session.UserID != 0 {
		candidates = append(candidates, aiSessionWorktreeDir(session.UserID, session.ID))
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		nativeSessionID, err := findNativeOpenCodeSessionInDatabase(candidate, session.CreatedAt)
		if err != nil {
			return err
		}
		if nativeSessionID != "" {
			return bindNativeOpenCodeSession(session, nativeSessionID)
		}
	}
	return nil
}

func bindNativeOpenCodeSession(session *model.AIDevSession, nativeSessionID string) error {
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
	current, err := repo.NewAIDevSessionRepo().GetSessionByID(session.ID)
	if err == nil {
		session.NativeSessionID = current.NativeSessionID
	}
	return err
}

func findNativeOpenCodeSessionID(output []byte, workDir string, startedAt time.Time) string {
	var sessions []struct {
		ID          string `json:"id"`
		Directory   string `json:"directory"`
		TimeCreated int64  `json:"time_created"`
	}
	if json.Unmarshal(output, &sessions) != nil {
		return ""
	}
	cleanWorkDir := filepath.Clean(workDir)
	var latestID string
	var latestCreated int64
	minimumCreated := startedAt.Add(-2 * time.Second).UnixMilli()
	for _, session := range sessions {
		if session.ID == "" || filepath.Clean(session.Directory) != cleanWorkDir || session.TimeCreated < minimumCreated {
			continue
		}
		if session.TimeCreated > latestCreated {
			latestID = session.ID
			latestCreated = session.TimeCreated
		}
	}
	return latestID
}
