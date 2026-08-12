package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

// 交付合并提交必须能说明合的是哪个会话、做了什么。
// 改回 `merge --no-edit` 后 Git 生成的是 "Merge commit 'sha' into HEAD"，本用例会失败。
func TestDeliveryMergeCommitCarriesSessionContext(t *testing.T) {
	session, sourceDir := createDeliveryWorktree(t, 41)
	session.CurrentTaskTitle = "修复移动端评审标签串页"
	session.AgentName = "claude"
	session.LastTaskID = 903
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("done\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "result.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "feat: add result"); err != nil {
		t.Fatal(err)
	}
	if _, err := mergeCodeSessionWorktree(session); err != nil {
		t.Fatal(err)
	}
	message, err := runCodeGit(sourceDir, "log", "-1", "--format=%B", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"merge: 修复移动端评审标签串页 (session #41)",
		"Session-Id: 41",
		"Task-Id: 903",
		"Executor: claude",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("merge message missing %q:\n%s", expected, message)
		}
	}
	if strings.Contains(message, "into HEAD") {
		t.Fatalf("merge message fell back to the Git default:\n%s", message)
	}
}

// 冲突解决后复用的 MERGE_MSG 里带着 Git 追加的「# Conflicts:」注释块。
// 去掉 --cleanup=strip 后这些注释会原样留在提交正文里，本用例会失败。
func TestConflictResolvedCommitDropsGitCommentBlock(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	conflictFile := filepath.Join(repositoryDir, "README.md")
	if _, err := runCodeGit(repositoryDir, "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitAs := func(content, message string) {
		t.Helper()
		if err := os.WriteFile(conflictFile, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repositoryDir, "add", "README.md"); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodeGit(repositoryDir, codeGitAuthoredArgs("commit", "-m", message)...); err != nil {
			t.Fatal(err)
		}
	}
	commitAs("feature\n", "feat: feature side")
	if _, err := runCodeGit(repositoryDir, "checkout", "-"); err != nil {
		t.Fatal(err)
	}
	commitAs("target\n", "feat: target side")

	session := &model.AIDevSession{ID: 51, CurrentTaskTitle: "冲突后收口"}
	mergeMessage := codeDeliveryMergeMessage(session, "")
	if _, err := runCodeGit(repositoryDir, codeGitAuthoredArgs(
		"merge", "--no-ff", "-m", mergeMessage, "feature",
	)...); err == nil {
		t.Fatal("expected the merge to conflict")
	}
	// 模拟网页冲突解决器：写回解决结果并暂存，然后走交付的收尾提交。
	if err := os.WriteFile(conflictFile, []byte("resolved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, codeResolvedMergeCommitArgs()...); err != nil {
		t.Fatal(err)
	}

	message, err := runCodeGit(repositoryDir, "log", "-1", "--format=%B", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(message, "# Conflicts:") || strings.Contains(message, "#\t") {
		t.Fatalf("git comment block leaked into the commit body:\n%s", message)
	}
	for _, expected := range []string{"merge: 冲突后收口 (session #51)", "Session-Id: 51"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("resolved merge lost %q:\n%s", expected, message)
		}
	}
}

// 会话提交要带上可追溯 trailer，但不能动用户写的提交说明本身。
func TestSessionCommitAppendsTrailersWithoutRewritingSubject(t *testing.T) {
	session, _ := createDeliveryWorktree(t, 42)
	session.AgentName = "codex"
	if err := os.WriteFile(filepath.Join(session.WorkDir, "result.txt"), []byte("done\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(session.WorkDir, "add", "result.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitCodeSessionWorktree(session, "fix: 收紧任务评审边界"); err != nil {
		t.Fatal(err)
	}
	subject, err := runCodeGit(session.WorkDir, "log", "-1", "--format=%s", "HEAD")
	if err != nil || strings.TrimSpace(subject) != "fix: 收紧任务评审边界" {
		t.Fatalf("user subject was rewritten: %q, %v", subject, err)
	}
	body, err := runCodeGit(session.WorkDir, "log", "-1", "--format=%b", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Session-Id: 42", "Executor: codex"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("commit body missing %q:\n%s", expected, body)
		}
	}
	author, err := runCodeGit(session.WorkDir, "log", "-1", "--format=%an <%ae>", "HEAD")
	if err != nil || strings.TrimSpace(author) != codeGitAuthorName+" <"+codeGitAuthorEmail+">" {
		t.Fatalf("unexpected commit author: %q, %v", author, err)
	}
}

// 昵称、执行器名都是外部可控的：带换行的值不能凭空造出额外的 trailer 行。
func TestCommitTrailersRejectInjectedLines(t *testing.T) {
	session := &model.AIDevSession{
		ID:            7,
		AgentName:     "claude\nCo-Authored-By: Attacker <evil@example.com>",
		ProviderModel: "sonnet\r\nSigned-off-by: Nobody <nobody@example.com>",
	}
	trailers := codeCommitTrailers(session)
	if len(trailers) != 3 {
		t.Fatalf("unexpected trailer count: %#v", trailers)
	}
	for _, trailer := range trailers {
		if strings.ContainsAny(trailer, "\r\n") {
			t.Fatalf("trailer carries a line break: %q", trailer)
		}
	}
	// 关键性质：注入的内容被压平进它所属的那一行，不能自成一条 trailer。
	// UserID 为 0 时本就不该出现 Co-Authored-By，出现了就说明注入成功了。
	for _, trailer := range trailers {
		if strings.HasPrefix(trailer, "Co-Authored-By:") || strings.HasPrefix(trailer, "Signed-off-by:") {
			t.Fatalf("injected value became a standalone trailer: %q", trailer)
		}
	}
	if trailers[1] != "Executor: claude Co-Authored-By: Attacker evil@example.com" {
		t.Fatalf("injected value was not flattened onto the executor line: %q", trailers[1])
	}
}

func TestDeliverySubjectFallsBackThroughTaskAndSession(t *testing.T) {
	withTask := &model.AIDevSession{ID: 5, Title: "会话标题", CurrentTaskTitle: "任务标题"}
	if subject := codeDeliverySubject(withTask); subject != "任务标题" {
		t.Fatalf("task title should win: %q", subject)
	}
	withSession := &model.AIDevSession{ID: 5, Title: "会话标题"}
	if subject := codeDeliverySubject(withSession); subject != "会话标题" {
		t.Fatalf("session title should be the fallback: %q", subject)
	}
	bare := &model.AIDevSession{ID: 5}
	if subject := codeDeliverySubject(bare); subject != "交付会话 #5 的变更" {
		t.Fatalf("session id should be the last resort: %q", subject)
	}
}

// 首行按 rune 截断，多字节标题不能被截成半个字符。
func TestCommitSubjectTruncatesOnRuneBoundary(t *testing.T) {
	subject := codeCommitSubjectText(strings.Repeat("修", 200))
	if runes := []rune(subject); len(runes) != codeCommitSubjectMaxRunes+1 || runes[len(runes)-1] != '…' {
		t.Fatalf("unexpected truncation: %q (%d runes)", subject, len(runes))
	}
	if !strings.HasPrefix(subject, strings.Repeat("修", codeCommitSubjectMaxRunes)) {
		t.Fatalf("truncated subject lost leading content: %q", subject)
	}
	if collapsed := codeCommitSubjectText("first line\nsecond line"); collapsed != "first line" {
		t.Fatalf("subject should stop at the first line: %q", collapsed)
	}
}

// 多仓交付的合并信息要带仓库名，否则父子仓的合并提交在历史里长得一模一样。
func TestMultiRepositoryMergeMessageNamesRepository(t *testing.T) {
	session := &model.AIDevSession{ID: 12, CurrentTaskTitle: "统一交付链路"}
	message := codeDeliveryMergeMessage(session, "admin")
	if !strings.HasPrefix(message, "merge: 统一交付链路 (session #12, admin)") {
		t.Fatalf("unexpected multi-repository merge subject: %q", message)
	}
	if single := codeDeliveryMergeMessage(session, ""); !strings.HasPrefix(single, "merge: 统一交付链路 (session #12)") {
		t.Fatalf("single repository subject should omit the repository name: %q", single)
	}
}
