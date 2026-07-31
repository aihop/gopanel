package api

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

func discoverNativeOpenCodeSession(session *model.AIDevSession, commandPath string, commandEnv []string, startedAt time.Time, done <-chan struct{}) {
	for attempt := 0; attempt < 20; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		command := exec.CommandContext(ctx, commandPath, "session", "list", "--format", "json", "--max-count", "20")
		command.Env = commandEnv
		output, err := command.Output()
		cancel()
		if err == nil {
			if nativeSessionID := findNativeOpenCodeSessionID(output, session.WorkDir, startedAt); nativeSessionID != "" {
				current, getErr := repo.NewAIDevSessionRepo().GetSessionByID(session.ID)
				if getErr == nil && current.NativeSessionID != nativeSessionID {
					current.NativeSessionID = nativeSessionID
					_ = repo.NewAIDevSessionRepo().UpdateSession(current)
				}
				return
			}
		}
		select {
		case <-done:
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
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
