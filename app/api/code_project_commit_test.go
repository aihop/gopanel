package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitCodeProjectRepositoryCommitsEverything(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	// 三种形态各来一个：改过的、新增未跟踪的、已暂存的。
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "new.txt"), []byte("new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "staged.txt"), []byte("staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "staged.txt")

	result := commitCodeProjectRepository(repositoryDir, "chore: 开会话前先提交")
	if result.Status != codeProjectCommitStatusCommitted || result.Commit == "" {
		t.Fatalf("应提交成功：%#v", result)
	}
	if result.Files != 3 {
		t.Fatalf("三个文件都应进提交，实际 %d", result.Files)
	}
	// 「先提交」的意图是把工作区清干净，留下未暂存的改动等于没解决问题。
	status := mustCodeGit(t, repositoryDir, "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("提交后工作区应干净：%q", status)
	}
}

// 干净的仓库不该产生空提交——批量提交时其余仓库往往本来就是干净的。
func TestCommitCodeProjectRepositorySkipsCleanRepository(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	before := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))

	result := commitCodeProjectRepository(repositoryDir, "chore: 不该产生提交")
	if result.Status != codeProjectCommitStatusClean || result.Commit != "" {
		t.Fatalf("干净仓库应跳过：%#v", result)
	}
	after := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))
	if before != after {
		t.Fatalf("干净仓库不该产生新提交：%s -> %s", before, after)
	}
}

// 改动全被 .gitignore 挡住时，status 非空但暂存区为空，同样不该造空提交。
func TestCommitCodeProjectRepositorySkipsWhenEverythingIsIgnored(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, ".gitignore"), []byte("junk/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", ".gitignore")
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "chore: add ignore")...)
	if err := os.MkdirAll(filepath.Join(repositoryDir, "junk"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "junk", "x.log"), []byte("noise\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))

	result := commitCodeProjectRepository(repositoryDir, "chore: 只有被忽略的文件")
	if result.Status != codeProjectCommitStatusClean {
		t.Fatalf("被忽略的改动应视为干净：%#v", result)
	}
	if after := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD")); before != after {
		t.Fatalf("不该产生新提交：%s -> %s", before, after)
	}
}

// 冲突标记留在文件里就提交，等于把半成品固化进历史。
func TestCommitCodeProjectRepositoryRefusesConflictMarkers(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "kept-staged.txt"), []byte("keep staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "kept-staged.txt")
	indexBefore := mustCodeGit(t, repositoryDir, "diff", "--cached", "--name-only")
	conflicted := strings.Join([]string{
		"<<<<<<< HEAD", "ours", "=======", "theirs", ">>>>>>> other", "",
	}, "\n")
	if err := os.WriteFile(filepath.Join(repositoryDir, "README.md"), []byte(conflicted), 0600); err != nil {
		t.Fatal(err)
	}
	before := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))

	result := commitCodeProjectRepository(repositoryDir, "chore: 带冲突标记")
	if result.Status != codeProjectCommitStatusFailed {
		t.Fatalf("带冲突标记应被拒：%#v", result)
	}
	if after := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD")); before != after {
		t.Fatalf("被拒时不该产生提交：%s -> %s", before, after)
	}
	if indexAfter := mustCodeGit(t, repositoryDir, "diff", "--cached", "--name-only"); indexAfter != indexBefore {
		t.Fatalf("被拒时应恢复原暂存区：before=%q after=%q", indexBefore, indexAfter)
	}
}

func TestCommitCodeProjectRepositoryRefusesSensitiveFilesWithoutChangingIndex(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "kept-staged.txt"), []byte("keep staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "kept-staged.txt")
	indexBefore := mustCodeGit(t, repositoryDir, "diff", "--cached", "--name-only")
	if err := os.WriteFile(filepath.Join(repositoryDir, ".env"), []byte("SECRET=value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result := commitCodeProjectRepository(repositoryDir, "chore: 不应提交凭据")
	if result.Status != codeProjectCommitStatusFailed || !strings.Contains(result.ErrorMessage, "敏感") {
		t.Fatalf("敏感文件应被拒：%#v", result)
	}
	if indexAfter := mustCodeGit(t, repositoryDir, "diff", "--cached", "--name-only"); indexAfter != indexBefore {
		t.Fatalf("被拒时不应改变暂存区：before=%q after=%q", indexBefore, indexAfter)
	}
}

func TestSummarizeCodeProjectCommitReportsWorstOutcome(t *testing.T) {
	cases := []struct {
		results  []codeProjectCommitResult
		expected string
	}{
		{[]codeProjectCommitResult{{Status: codeProjectCommitStatusCommitted}}, "success"},
		{[]codeProjectCommitResult{{Status: codeProjectCommitStatusClean}}, "success"},
		{[]codeProjectCommitResult{
			{Status: codeProjectCommitStatusCommitted}, {Status: codeProjectCommitStatusFailed},
		}, "failed"},
	}
	for _, testCase := range cases {
		if actual := summarizeCodeProjectCommit(testCase.results); actual != testCase.expected {
			t.Fatalf("%#v 应汇总为 %q，实际 %q", testCase.results, testCase.expected, actual)
		}
	}
}

// 提示要说清「差在哪、该做什么」：原本只说「请先处理分支差异」，
// 用户不知道该 pull 还是 push 还是 commit，实测失败了 4 次。
func TestCodeRepositoryDivergenceErrorNamesTheActualGap(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	targetBranch := strings.TrimSpace(mustCodeGit(t, repositoryDir, "branch", "--show-current"))
	// 造一个「远端」引用，然后让本地领先两个提交。
	mustCodeGit(t, repositoryDir, "update-ref", "refs/remotes/origin/"+targetBranch, "HEAD")
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(repositoryDir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		mustCodeGit(t, repositoryDir, "add", name)
		mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "feat: "+name)...)
	}

	ahead, behind, ok := codeRepositoryAheadBehind(repositoryDir, targetBranch, "refs/remotes/origin/"+targetBranch)
	if !ok || ahead != 2 || behind != 0 {
		t.Fatalf("应数出领先 2 落后 0：ahead=%d behind=%d ok=%v", ahead, behind, ok)
	}
	err := codeRepositoryDivergenceError(repositoryDir, targetBranch, "refs/remotes/origin/"+targetBranch)
	if !strings.Contains(err.Error(), "领先") || !strings.Contains(err.Error(), "push") {
		t.Fatalf("领先远端时应建议推送：%v", err)
	}
}

// 数不出来时退回不带数字的说法——少一个数字好过报一个错的数字。
func TestCodeRepositoryDivergenceErrorFallsBackWithoutCounts(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	err := codeRepositoryDivergenceError(repositoryDir, "main", "refs/remotes/origin/does-not-exist")
	if err == nil || strings.Contains(err.Error(), "领先") || strings.Contains(err.Error(), "落后") {
		t.Fatalf("取不到计数时不该编造数字：%v", err)
	}
	if !strings.Contains(err.Error(), "未提交变更") {
		t.Fatalf("仍应说明根本原因：%v", err)
	}
}
