package service

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/aihop/gopanel/global"
)

func gitInRepo(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitEmpty(t *testing.T, dir, subject string) string {
	t.Helper()
	gitInRepo(t, dir, "commit", "--allow-empty", "-m", subject)
	return gitInRepo(t, dir, "rev-parse", "HEAD")
}

func TestCollectPipelineChangelog(t *testing.T) {
	dir := t.TempDir()
	gitInRepo(t, dir, "init", "-q", "-b", "main")
	first := commitEmpty(t, dir, "feat: 初始化项目")
	commitEmpty(t, dir, "fix: 修复登录跳转")
	commitEmpty(t, dir, "feat: 新增节点管理")

	ctx := context.Background()
	// GetPipelineLogger 会按 BaseDir 落日志文件，测试里指到临时目录，别在仓库里拉屎
	oldBaseDir := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = t.TempDir()
	defer func() { global.CONF.System.BaseDir = oldBaseDir }()
	logger := GetPipelineLogger(0)
	defer RemovePipelineLogger(0)

	t.Run("从上次成功构建的 commit 起算", func(t *testing.T) {
		got := collectPipelineChangelog(ctx, logger, dir, first)
		// git log 是倒序，最新的提交在前
		want := "feat: 新增节点管理\nfix: 修复登录跳转"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("没有上次记录时取最近若干条", func(t *testing.T) {
		got := collectPipelineChangelog(ctx, logger, dir, "")
		if len(strings.Split(got, "\n")) != 3 {
			t.Fatalf("首次构建应列出全部 3 条，got %q", got)
		}
	})

	t.Run("上次的 commit 不在本地历史时回退", func(t *testing.T) {
		got := collectPipelineChangelog(ctx, logger, dir, "0000000000000000000000000000000000000000")
		if !strings.Contains(got, "feat: 新增节点管理") {
			t.Fatalf("应回退到最近提交，got %q", got)
		}
	})

	t.Run("没有新提交时为空", func(t *testing.T) {
		head := gitInRepo(t, dir, "rev-parse", "HEAD")
		if got := collectPipelineChangelog(ctx, logger, dir, head); got != "" {
			t.Fatalf("want empty, got %q", got)
		}
	})

	t.Run("merge 提交被排除", func(t *testing.T) {
		base := gitInRepo(t, dir, "rev-parse", "HEAD")
		gitInRepo(t, dir, "checkout", "-q", "-b", "feature")
		commitEmpty(t, dir, "feat: 分支上的改动")
		gitInRepo(t, dir, "checkout", "-q", "main")
		gitInRepo(t, dir, "merge", "--no-ff", "-q", "feature", "-m", "Merge branch 'feature'")
		got := collectPipelineChangelog(ctx, logger, dir, base)
		if strings.Contains(got, "Merge branch") {
			t.Fatalf("merge 提交不应出现: %q", got)
		}
		if !strings.Contains(got, "feat: 分支上的改动") {
			t.Fatalf("分支提交应保留: %q", got)
		}
	})
}

func TestNormalizeChangelog(t *testing.T) {
	t.Run("去空行去重", func(t *testing.T) {
		got := normalizeChangelog("a\n\n  b  \na\n")
		if got != "a\nb" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("单行过长按 rune 截断", func(t *testing.T) {
		long := strings.Repeat("中", 300)
		got := normalizeChangelog(long)
		if !strings.HasSuffix(got, "…") {
			t.Fatalf("应有截断标记: %q", got)
		}
		if len([]rune(got)) != 201 {
			t.Fatalf("应截到 200 个 rune + 省略号，got %d", len([]rune(got)))
		}
		if strings.Contains(got, "�") {
			t.Fatal("中文被切坏了")
		}
	})

	t.Run("条数超限截断", func(t *testing.T) {
		var lines []string
		for i := 0; i < changelogMaxCommits+10; i++ {
			lines = append(lines, strings.Repeat("x", i%7+1)+string(rune('a'+i%26))+string(rune('0'+i%10))+"-"+strings.Repeat("y", i%3))
		}
		got := strings.Split(normalizeChangelog(strings.Join(lines, "\n")), "\n")
		if len(got) != changelogMaxCommits+1 {
			t.Fatalf("应截到 %d 条 + 提示行，got %d", changelogMaxCommits, len(got))
		}
		if !strings.Contains(got[len(got)-1], "截断") {
			t.Fatalf("末行应为截断提示: %q", got[len(got)-1])
		}
	})
}
