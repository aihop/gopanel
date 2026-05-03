package api

import (
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"gorm.io/gorm"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func formatExecOutput(output []byte, fallback string) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return fallback
	}
	return text
}
func defaultAIAgentWorkDir(userID uint) string {
	hostHome, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(hostHome) == "" {
		hostHome = "/tmp"
	}
	if userID == 0 {
		return filepath.Join(hostHome, ".gopanel", "ai_agent", "workspace", "default")
	}
	return filepath.Join(hostHome, ".gopanel", "ai_agent", "workspace", fmt.Sprintf("user_%d", userID))
}
func normalizeAIAgentWorkDir(workDir string, userID uint) string {
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if workDir == "" || workDir == "." {
		return defaultAIAgentWorkDir(userID)
	}
	if runtime.GOOS == "darwin" {
		if workDir == "/" || workDir == "/root" || !strings.HasPrefix(workDir, "/Users/") {
			return defaultAIAgentWorkDir(userID)
		}
	}
	return workDir
}
func extractPreviewURLs(text string) []string {
	matches := previewURLPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	results := make([]string, 0, len(matches))
	for _, raw := range matches {
		candidate := strings.TrimSpace(raw)
		candidate = strings.TrimRight(candidate, ".,;)]}")
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(candidate, "http://") && !strings.HasPrefix(candidate, "https://") {
			candidate = "http://" + candidate
		}
		if _, err := url.ParseRequestURI(candidate); err != nil {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		results = append(results, candidate)
	}
	return results
}
func previewStatusForURL(previewURL string) string {
	u, err := url.Parse(previewURL)
	if err != nil {
		return "unknown"
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0":
		return "local"
	default:
		return "ready"
	}
}
func previewTitleForURL(previewURL string) string {
	u, err := url.Parse(previewURL)
	if err != nil {
		return "开发预览"
	}
	title := u.Host
	if strings.TrimSpace(u.Path) != "" && u.Path != "/" {
		title += u.Path
	}
	if strings.TrimSpace(title) == "" {
		return "开发预览"
	}
	return title
}
func upsertAIPreviews(sessionRepo repo.IAIDevSessionRepo, session *model.AIDevSession, task *model.AITask, instruction *model.AIInstruction, output string) ([]*model.AIPreview, error) {
	if session == nil {
		return nil, nil
	}
	if instruction != nil && !instruction.AutoPreview {
		return nil, nil
	}
	urls := extractPreviewURLs(output)
	if len(urls) == 0 {
		return nil, nil
	}
	now := time.Now()
	previews := make([]*model.AIPreview, 0, len(urls))
	for _, previewURL := range urls {
		existing, err := sessionRepo.FindPreviewByURL(session.ID, previewURL)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			preview := &model.AIPreview{SessionID: session.ID, PreviewType: "web", Source: "agent_output", Title: previewTitleForURL(previewURL), URL: previewURL, Status: previewStatusForURL(previewURL), LastCheckedAt: &now}
			if task != nil {
				preview.TaskID = task.ID
			}
			if instruction != nil {
				preview.InstructionID = instruction.ID
			}
			if createErr := sessionRepo.CreatePreview(preview); createErr != nil {
				return nil, createErr
			}
			previews = append(previews, preview)
			continue
		}
		existing.Title = previewTitleForURL(previewURL)
		existing.Status = previewStatusForURL(previewURL)
		existing.Source = "agent_output"
		existing.LastCheckedAt = &now
		if task != nil {
			existing.TaskID = task.ID
		}
		if instruction != nil {
			existing.InstructionID = instruction.ID
		}
		if updateErr := sessionRepo.UpdatePreview(existing); updateErr != nil {
			return nil, updateErr
		}
		previews = append(previews, existing)
	}
	return previews, nil
}
