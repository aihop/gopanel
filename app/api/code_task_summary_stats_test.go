package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func commitCodeSummaryFiles(t *testing.T, repositoryDir string, files map[string]string) string {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(repositoryDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runCodeGit(repositoryDir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(
		repositoryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local",
		"commit", "-m", "summary fixture",
	); err != nil {
		t.Fatal(err)
	}
	commit, err := runCodeGit(repositoryDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func longCodeFixtureText(lines int) string {
	content := ""
	for index := 0; index < lines; index++ {
		content += "generated line\n"
	}
	return content
}

// 依赖安装刷新 lock 文件会贡献上万行，不该被算作任务产出。
func TestLoadCodeTaskDiffStatsExcludesGeneratedLockFiles(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	base := commitCodeSummaryFiles(t, repositoryDir, map[string]string{
		"app.go":                    "package main\n",
		"package-lock.json":         "{}\n",
		"web/package-lock.json":     "{}\n",
		"web/sub/pubspec.lock":      "{}\n",
		"client/macos/Podfile.lock": "PODS:\n",
	})
	head := commitCodeSummaryFiles(t, repositoryDir, map[string]string{
		"app.go":                    "package main\n\nfunc main() {}\n",
		"package-lock.json":         longCodeFixtureText(400),
		"web/package-lock.json":     longCodeFixtureText(500),
		"web/sub/pubspec.lock":      longCodeFixtureText(600),
		"client/macos/Podfile.lock": longCodeFixtureText(700),
	})

	stats, ok := loadCodeTaskDiffStats(repositoryDir, base, head, map[string]codeTaskDiffStats{})
	if !ok {
		t.Fatal("diff stats should be available")
	}
	// 只应统计 app.go 的两行新增；子目录里的 lock 文件同样要被排除。
	if stats.Files != 1 || stats.Additions != 2 {
		t.Fatalf("generated lock files leaked into task stats: %#v", stats)
	}
}

func TestLoadCodeTaskDiffStatsCountsRealCodeInNestedDirs(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	base := commitCodeSummaryFiles(t, repositoryDir, map[string]string{"pkg/svc/a.go": "package svc\n"})
	head := commitCodeSummaryFiles(t, repositoryDir, map[string]string{
		"pkg/svc/a.go": "package svc\n\nfunc A() {}\n",
		"pkg/svc/b.go": "package svc\n\nfunc B() {}\n",
	})

	stats, ok := loadCodeTaskDiffStats(repositoryDir, base, head, map[string]codeTaskDiffStats{})
	if !ok || stats.Files != 2 || stats.Additions != 5 {
		t.Fatalf("nested source changes must still be counted: %#v (ok=%v)", stats, ok)
	}
}

// 交付快照固化统计后，即使 worktree 提交已不可达，历史任务的行数也不该归零。
func TestCodeTaskSummaryUsesStoredStatsWhenCommitUnreachable(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)

	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "session", WorkDir: repositoryDir,
		SourceWorkDir: repositoryDir, WorktreeBranch: "stats-branch",
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: repositoryDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	// base/worktree 提交都指向不存在的对象，实时 diff 必然算不出来。
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, ProjectID: 1, UserID: 1, Status: codeDeliveryCompleted,
		SourceWorkDir: repositoryDir, WorkDir: repositoryDir, WorktreeBranch: "stats-branch",
		BaseCommit:     "0000000000000000000000000000000000000001",
		WorktreeCommit: "0000000000000000000000000000000000000002",
		StatAdditions:  137, StatDeletions: 42, StatFiles: 9,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}

	items, err := buildCodeTaskListItems([]*model.AITask{task}, true)
	if err != nil {
		t.Fatal(err)
	}
	summary := items[0].Summary
	if summary.Additions != 137 || summary.Deletions != 42 || summary.ChangedFiles != 9 || !summary.HasDiff ||
		len(summary.Repositories) != 1 || summary.Repositories[0].Branch != "stats-branch" ||
		summary.Repositories[0].Additions != 137 || summary.Repositories[0].ChangedFiles != 9 {
		t.Fatalf("stored delivery stats were not used: %#v", summary)
	}
}

// 没有固化统计的历史数据必须回退到实时计算，不能因为新字段而丢掉行数。
func TestCodeTaskSummaryFallsBackToLiveDiffWithoutStoredStats(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	base, err := runCodeGit(repositoryDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	head := commitCodeSummaryFiles(t, repositoryDir, map[string]string{"legacy.go": "package legacy\n\nfunc L() {}\n"})

	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "session", WorkDir: repositoryDir,
		SourceWorkDir: repositoryDir, WorktreeBranch: "legacy-branch",
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: repositoryDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	delivery := &model.AICodeDelivery{
		SessionID: session.ID, ProjectID: 1, UserID: 1, Status: codeDeliveryCompleted,
		SourceWorkDir: repositoryDir, WorkDir: repositoryDir, WorktreeBranch: "legacy-branch",
		BaseCommit: base, WorktreeCommit: head,
	}
	if err := database.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}

	items, err := buildCodeTaskListItems([]*model.AITask{task}, true)
	if err != nil {
		t.Fatal(err)
	}
	summary := items[0].Summary
	if !summary.HasDiff || summary.Additions != 3 || summary.ChangedFiles != 1 {
		t.Fatalf("legacy rows must fall back to live diff: %#v", summary)
	}
}

func TestCodeTaskSummaryRestoresCleanedMultiRepositoryHistory(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "history", WorkDir: t.TempDir(),
		IsolationMode: codeIsolationMultiWorktree,
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: session.WorkDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	repositoryResults, err := json.Marshal([]codeRepositoryDeliveryResult{
		{
			RepositoryName: "api", Branch: "gopanel/code-96-api", Status: codeDeliveryCompleted,
			PushStatus: "local", Additions: 21, Deletions: 5, ChangedFiles: 3,
		},
		{
			RepositoryName: "web", Branch: "gopanel/code-96-web", Status: codeDeliveryCompleted,
			PushStatus: "local", Additions: 8, Deletions: 2, ChangedFiles: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	job := &model.AICodeDeliveryJob{
		SessionID: session.ID, TaskID: task.ID, ProjectID: 1, UserID: 1,
		Status: codeDeliveryJobCompleted, Stage: codeDeliveryStageCompleted, Progress: 100,
		RepositoryResults: string(repositoryResults),
	}
	if err := database.Create(job).Error; err != nil {
		t.Fatal(err)
	}

	items, err := buildCodeTaskListItems([]*model.AITask{task}, true)
	if err != nil {
		t.Fatal(err)
	}
	summary := items[0].Summary
	if summary.Branch != "gopanel/code-96-api" || summary.GitStatus != "merged" ||
		summary.Additions != 29 || summary.Deletions != 7 || summary.ChangedFiles != 4 ||
		len(summary.Repositories) != 2 || summary.Repositories[1].Branch != "gopanel/code-96-web" {
		t.Fatalf("cleaned repository history was not restored: %#v", summary)
	}
}
