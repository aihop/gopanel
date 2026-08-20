package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func writeWorkspaceManifest(t *testing.T, sources []aiProjectWorkspaceSource) string {
	t.Helper()
	dir := t.TempDir()
	payload, err := json.Marshal(aiProjectWorkspaceManifest{Version: 1, Sources: sources})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, aiProjectWorkspaceManifestName), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// 多仓项目的工作目录是合成容器，模型看到的是链接路径、工具解析后是真实路径。
// 不把映射摆出来，模型会把同一份文件当成两个地方。
func TestWorkspacePathsPromptExposesRealPaths(t *testing.T) {
	dir := writeWorkspaceManifest(t, []aiProjectWorkspaceSource{
		{Path: "/Users/hugh/code/aihop/apay", LinkName: "apay"},
		{Path: "/Users/hugh/code/aihop/qingpu-ai", LinkName: "qingpu-ai"},
	})
	prompt := codeWorkspacePathsPrompt(&model.AIDevSession{WorkDir: dir}, "改一下登录接口")

	if !strings.HasPrefix(prompt, "改一下登录接口") {
		t.Fatalf("原始指令必须保留在最前面：%q", prompt)
	}
	for _, expected := range []string{
		"apay/ → /Users/hugh/code/aihop/apay",
		"qingpu-ai/ → /Users/hugh/code/aihop/qingpu-ai",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("缺少映射 %q：%s", expected, prompt)
		}
	}
}

// 单仓 direct 和 Worktree 隔离下，模型看到的本来就是唯一且真实的路径，
// 再插一段只会是噪音，还白白占上下文。
func TestWorkspacePathsPromptStaysQuietWithoutSyntheticWorkspace(t *testing.T) {
	t.Run("没有清单的普通目录", func(t *testing.T) {
		session := &model.AIDevSession{WorkDir: t.TempDir()}
		if got := codeWorkspacePathsPrompt(session, "任务"); got != "任务" {
			t.Fatalf("不该改动提示：%q", got)
		}
	})

	t.Run("Worktree 隔离即使有清单也不注入", func(t *testing.T) {
		dir := writeWorkspaceManifest(t, []aiProjectWorkspaceSource{
			{Path: "/repo/a", LinkName: "a"},
		})
		session := &model.AIDevSession{WorkDir: dir, WorktreeBranch: "gopanel/code-1"}
		if got := codeWorkspacePathsPrompt(session, "任务"); got != "任务" {
			t.Fatalf("Worktree 会话不该注入路径映射：%q", got)
		}
	})

	t.Run("空会话", func(t *testing.T) {
		if got := codeWorkspacePathsPrompt(nil, "任务"); got != "任务" {
			t.Fatalf("空会话不该改动提示：%q", got)
		}
	})
}

// 注入块必须能从对话展示里剔干净，否则用户会在自己的消息里看到一堆路径清单。
func TestWorkspacePathsPromptIsStrippedFromConversation(t *testing.T) {
	dir := writeWorkspaceManifest(t, []aiProjectWorkspaceSource{
		{Path: "/Users/hugh/code/aihop/apay", LinkName: "apay"},
	})
	prompt := codeWorkspacePathsPrompt(&model.AIDevSession{WorkDir: dir}, "改一下登录接口")
	if got := stripInjectedConversationPrompt(prompt); got != "改一下登录接口" {
		t.Fatalf("展示时应只剩用户原话，实际 %q", got)
	}
}
