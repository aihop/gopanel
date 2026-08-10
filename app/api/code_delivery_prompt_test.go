package api

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestCodeManagedDeliveryPromptOnlyAppliesToIsolatedSessions(t *testing.T) {
	prompt := "实现功能"
	isolated := codeManagedDeliveryPrompt(&model.AIDevSession{WorktreeBranch: "gopanel/code-1"}, prompt)
	if !strings.Contains(isolated, prompt) || !strings.Contains(isolated, "不要执行 git push") ||
		!strings.Contains(isolated, "安全合并到本地目标分支") || !strings.Contains(isolated, "单独执行“推送远端”") {
		t.Fatalf("managed delivery instruction unavailable: %q", isolated)
	}
	if plain := codeManagedDeliveryPrompt(&model.AIDevSession{}, prompt); plain != prompt {
		t.Fatalf("plain project prompt was modified: %q", plain)
	}
}
