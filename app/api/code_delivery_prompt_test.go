package api

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeManagedDeliveryPromptOnlyAppliesToIsolatedSessions(t *testing.T) {
	prompt := "实现功能"
	isolated := codeManagedDeliveryPrompt(&model.AIDevSession{WorktreeBranch: "gopanel/code-1"}, prompt)
	if !strings.Contains(isolated, prompt) || !strings.Contains(isolated, "不要执行 git push") || !strings.Contains(isolated, "合并到本地项目目录") {
		t.Fatalf("managed delivery instruction unavailable: %q", isolated)
	}
	if plain := codeManagedDeliveryPrompt(&model.AIDevSession{}, prompt); plain != prompt {
		t.Fatalf("plain project prompt was modified: %q", plain)
	}
}
