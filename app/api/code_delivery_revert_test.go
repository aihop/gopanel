package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

// deliverIntoTargetBranch 在临时仓库里造一次真实交付：
// 建分支改文件、提交，再 --no-ff 合回目标分支，返回合并提交。
func deliverIntoTargetBranch(t *testing.T, repositoryDir, branch, file, content string) string {
	t.Helper()
	targetBranch := strings.TrimSpace(mustCodeGit(t, repositoryDir, "branch", "--show-current"))
	mustCodeGit(t, repositoryDir, "checkout", "-b", branch)
	if err := os.WriteFile(filepath.Join(repositoryDir, file), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", file)
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "feat: "+file)...)
	mustCodeGit(t, repositoryDir, "checkout", targetBranch)
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("merge", "--no-ff", "-m", "merge: "+branch, branch)...)
	return strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))
}

func mustCodeGit(t *testing.T, workDir string, args ...string) string {
	t.Helper()
	output, err := runCodeGit(workDir, args...)
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return output
}

func revertRequestFor(repositoryDir, mergeCommit string) codeDeliveryRevertRequest {
	return codeDeliveryRevertRequest{SourceDir: repositoryDir, MergeCommit: mergeCommit, TargetBranch: "master"}
}

func targetBranchOf(t *testing.T, repositoryDir string) string {
	t.Helper()
	return strings.TrimSpace(mustCodeGit(t, repositoryDir, "branch", "--show-current"))
}

func TestRevertCodeDeliveryUndoesTheDeliveredChange(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	mergeCommit := deliverIntoTargetBranch(t, repositoryDir, "gopanel/code-61", "feature.txt", "delivered\n")
	if _, err := os.Stat(filepath.Join(repositoryDir, "feature.txt")); err != nil {
		t.Fatalf("交付后文件应存在：%v", err)
	}
	request := revertRequestFor(repositoryDir, mergeCommit)
	request.TargetBranch = targetBranchOf(t, repositoryDir)

	session := &model.AIDevSession{ID: 61, CurrentTaskTitle: "撤销验证"}
	result, err := revertCodeDeliveryInRepository(request, session)
	if err != nil || result.Status != codeRevertStatusReverted {
		t.Fatalf("撤销应成功：%#v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(repositoryDir, "feature.txt")); !os.IsNotExist(err) {
		t.Fatalf("交付的文件应被撤销掉：%v", err)
	}
	message := mustCodeGit(t, repositoryDir, "log", "-1", "--format=%B", "HEAD")
	for _, expected := range []string{
		"revert: 撤销验证 (session #61)", "This reverts commit " + mergeCommit + ".", "Session-Id: 61",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("撤销提交缺少 %q：\n%s", expected, message)
		}
	}
	// 原合并提交必须留在历史里：撤销是追加反向提交，不是改写历史。
	if _, err := runCodeGit(repositoryDir, "merge-base", "--is-ancestor", mergeCommit, "HEAD"); err != nil {
		t.Fatal("原交付提交不应从历史里消失")
	}
}

// 反向提交撤销后原合并提交仍在分支上，只看 Git 无法判断是否撤过。
// 第二次撤销必须被识别为「已无改动可撤」，否则会把交付内容又加回去。
func TestRevertCodeDeliveryTwiceDoesNotReapplyTheChange(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	mergeCommit := deliverIntoTargetBranch(t, repositoryDir, "gopanel/code-62", "feature.txt", "delivered\n")
	request := revertRequestFor(repositoryDir, mergeCommit)
	request.TargetBranch = targetBranchOf(t, repositoryDir)
	session := &model.AIDevSession{ID: 62}

	if result, err := revertCodeDeliveryInRepository(request, session); err != nil || result.Status != codeRevertStatusReverted {
		t.Fatalf("首次撤销应成功：%#v, %v", result, err)
	}
	result, err := revertCodeDeliveryInRepository(request, session)
	if err != nil {
		t.Fatalf("二次撤销不该报错：%v", err)
	}
	if result.Status != codeRevertStatusSkipped {
		t.Fatalf("二次撤销应被跳过，实际 %#v", result)
	}
	if _, err := os.Stat(filepath.Join(repositoryDir, "feature.txt")); !os.IsNotExist(err) {
		t.Fatal("二次撤销把交付内容又加回来了")
	}
}

// 交付提交不在目标分支上时不该凭空造一个反向提交。
func TestRevertCodeDeliverySkipsWhenDeliveryIsNotOnTarget(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	targetBranch := targetBranchOf(t, repositoryDir)
	mustCodeGit(t, repositoryDir, "checkout", "-b", "sidetrack")
	if err := os.WriteFile(filepath.Join(repositoryDir, "side.txt"), []byte("side\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "side.txt")
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "feat: side")...)
	sideCommit := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))
	mustCodeGit(t, repositoryDir, "checkout", targetBranch)

	request := revertRequestFor(repositoryDir, sideCommit)
	request.TargetBranch = targetBranch
	result, err := revertCodeDeliveryInRepository(request, &model.AIDevSession{ID: 63})
	if err != nil {
		t.Fatalf("不该报错：%v", err)
	}
	if result.Status != codeRevertStatusSkipped || !strings.Contains(result.ErrorMessage, "不在目标分支") {
		t.Fatalf("应跳过并说明原因：%#v", result)
	}
}

// 撤销与目标分支上的后续改动冲突时，必须原样保留分支并报告冲突文件，
// 不能留下半吊子的中间状态。
func TestRevertCodeDeliveryReportsConflictAndLeavesTargetIntact(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	mergeCommit := deliverIntoTargetBranch(t, repositoryDir, "gopanel/code-64", "feature.txt", "delivered\n")
	if err := os.WriteFile(filepath.Join(repositoryDir, "feature.txt"), []byte("changed by someone else\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "feature.txt")
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "fix: 后续改动")...)
	tipBefore := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))

	request := revertRequestFor(repositoryDir, mergeCommit)
	request.TargetBranch = targetBranchOf(t, repositoryDir)
	result, err := revertCodeDeliveryInRepository(request, &model.AIDevSession{ID: 64})
	if err != nil {
		t.Fatalf("冲突应作为结果返回而不是错误：%v", err)
	}
	if result.Status != codeRevertStatusConflict || len(result.ConflictFiles) == 0 {
		t.Fatalf("应报告冲突：%#v", result)
	}
	if tip := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD")); tip != tipBefore {
		t.Fatalf("冲突时目标分支不该被改动：%s -> %s", tipBefore, tip)
	}
	status := mustCodeGit(t, repositoryDir, "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("冲突时不该在用户工作区留下痕迹：%s", status)
	}
}

// 目标分支能快进时交付提交是普通提交，带 -m 1 会被 Git 拒绝。
func TestRevertCodeDeliveryHandlesNonMergeDeliveryCommit(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if err := os.WriteFile(filepath.Join(repositoryDir, "plain.txt"), []byte("plain\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustCodeGit(t, repositoryDir, "add", "plain.txt")
	mustCodeGit(t, repositoryDir, codeGitAuthoredArgs("commit", "-m", "feat: plain delivery")...)
	plainCommit := strings.TrimSpace(mustCodeGit(t, repositoryDir, "rev-parse", "HEAD"))

	request := revertRequestFor(repositoryDir, plainCommit)
	request.TargetBranch = targetBranchOf(t, repositoryDir)
	result, err := revertCodeDeliveryInRepository(request, &model.AIDevSession{ID: 65})
	if err != nil || result.Status != codeRevertStatusReverted {
		t.Fatalf("普通提交也应能撤销：%#v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(repositoryDir, "plain.txt")); !os.IsNotExist(err) {
		t.Fatal("普通提交的改动应被撤销")
	}
}

func TestSummarizeCodeRevertStatusReflectsWorstOutcome(t *testing.T) {
	cases := []struct {
		name     string
		results  []codeRepositoryRevertResult
		expected string
	}{
		{"全部成功", []codeRepositoryRevertResult{{Status: codeRevertStatusReverted}, {Status: codeRevertStatusReverted}}, codeRevertStatusReverted},
		{"部分成功", []codeRepositoryRevertResult{{Status: codeRevertStatusReverted}, {Status: codeRevertStatusConflict}}, "partial"},
		{"全部冲突", []codeRepositoryRevertResult{{Status: codeRevertStatusConflict}}, codeRevertStatusConflict},
		{"全部失败", []codeRepositoryRevertResult{{Status: "failed"}}, "failed"},
		{"全部跳过", []codeRepositoryRevertResult{{Status: codeRevertStatusSkipped}}, codeRevertStatusSkipped},
		{"成功加跳过", []codeRepositoryRevertResult{{Status: codeRevertStatusReverted}, {Status: codeRevertStatusSkipped}}, codeRevertStatusReverted},
	}
	for _, testCase := range cases {
		if actual := summarizeCodeRevertStatus(testCase.results); actual != testCase.expected {
			t.Fatalf("%s：期望 %q，实际 %q", testCase.name, testCase.expected, actual)
		}
	}
}
