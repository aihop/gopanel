package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
)

func TestBuildCodeTaskListItemsSummarizesRunsAndCommittedWorktree(t *testing.T) {
	database := withCodeGovernanceDB(t)
	repositoryDir := createCodeGitRepository(t)
	baseCommit, err := runCodeGit(repositoryDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryDir, "summary.txt"), []byte("first\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "add", "summary.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(repositoryDir, "-c", "user.name=GoPanel Test", "-c", "user.email=test@gopanel.local", "commit", "-m", "add summary"); err != nil {
		t.Fatal(err)
	}

	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "session", WorkDir: repositoryDir,
		SourceWorkDir: repositoryDir, WorktreeBranch: "task-summary", BaseCommit: baseCommit,
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: repositoryDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	older := time.Now().Add(-time.Minute)
	newer := time.Now()
	runs := []model.AIExecutionRun{
		{CreatedAt: older, SessionID: session.ID, TaskID: task.ID, ExecutorID: "codex", Model: "old-model", Prompt: "one", Status: "completed", DurationMS: 1250, TotalTokens: 120, TokenUsageStatus: codeTokenUsageRecorded, StartedAt: older},
		{CreatedAt: newer, SessionID: session.ID, TaskID: task.ID, ExecutorID: "claude", Model: "new-model", Prompt: "two", Status: "completed", DurationMS: 2750, TotalTokens: 80, TokenUsageStatus: codeTokenUsageRecovered, StartedAt: newer},
	}
	if err := database.Create(&runs).Error; err != nil {
		t.Fatal(err)
	}

	items, err := buildCodeTaskListItems([]*model.AITask{task}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	summary := items[0].Summary
	if summary.DurationMS != 4000 || summary.TotalTokens != 200 || summary.TokenUsageStatus != codeTokenUsageRecovered || summary.TokenRecoveredRuns != 1 || summary.Executor != "claude" || summary.Model != "new-model" {
		t.Fatalf("unexpected run summary: %#v", summary)
	}
	if summary.GitStatus != "committed" || summary.Branch != "task-summary" {
		t.Fatalf("unexpected Git summary: %#v", summary)
	}
	if !summary.HasDiff || summary.Additions != 2 || summary.Deletions != 0 || summary.ChangedFiles != 1 {
		t.Fatalf("unexpected diff summary: %#v", summary)
	}
}

func TestCodeTaskTokenUsageStatusMarksPartialData(t *testing.T) {
	status := codeTaskTokenUsageStatus(codeTaskDurationSummaryRow{RecordedRuns: 1, UnavailableRuns: 1})
	if status != "partial" {
		t.Fatalf("got %q, want partial", status)
	}
}

func TestCodeTaskDeliveryStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		pushStatus string
		want       string
	}{
		{name: "working", want: "working"},
		{name: "committed", status: codeDeliveryPrepared, want: "committed"},
		{name: "merged", status: codeDeliveryCompleted, want: "merged"},
		{name: "pushed", status: codeDeliveryCompleted, pushStatus: "pushed", want: "pushed"},
		{name: "push failed", status: codeDeliveryCompleted, pushStatus: codePushFailed, want: "push_failed"},
		{name: "conflict", status: "conflict", pushStatus: "pushed", want: "conflict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := codeTaskDeliveryStatus(test.status, test.pushStatus); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestApplyCodeTaskDeliverySummaryIncludesPushError(t *testing.T) {
	summary := codeTaskSummary{}
	applyCodeTaskDeliverySummary(&summary, model.AICodeDelivery{
		Status: codeDeliveryCompleted, PushStatus: codePushFailed, PushError: "remote rejected",
	}, make(map[string]codeTaskDiffStats))
	if summary.GitStatus != "push_failed" || summary.GitError != "remote rejected" {
		t.Fatalf("unexpected push failure summary: %#v", summary)
	}
}

func TestAggregateCodeTaskGitStatusesUsesLeastCompleteState(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     string
	}{
		{name: "conflict wins", statuses: []string{"pushed", "conflict", "working"}, want: "conflict"},
		{name: "push failure before working", statuses: []string{"working", "push_failed"}, want: "push_failed"},
		{name: "working before pushed", statuses: []string{"pushed", "working"}, want: "working"},
		{name: "committed before merged", statuses: []string{"merged", "committed"}, want: "committed"},
		{name: "merged before pushed", statuses: []string{"pushed", "merged"}, want: "merged"},
		{name: "all pushed", statuses: []string{"pushed", "pushed"}, want: "pushed"},
		{name: "empty", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := aggregateCodeTaskGitStatuses(test.statuses); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildCodeTaskListItemsCanSkipGitInspection(t *testing.T) {
	database := withCodeGovernanceDB(t)
	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "session", WorkDir: "/missing/worktree",
		WorktreeBranch: "task-summary", BaseCommit: "missing",
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: session.WorkDir}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	run := model.AIExecutionRun{
		SessionID: session.ID, TaskID: task.ID, ExecutorID: "codex", Model: "gpt-5",
		Prompt: "work", Status: "completed", StartedAt: time.Now(), DurationMS: 1500,
	}
	if err := database.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	items, err := buildCodeTaskListItems([]*model.AITask{task}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Summary.DurationMS != 1500 || items[0].Summary.GitStatus != "" {
		t.Fatalf("unexpected database-only summary: %#v", items)
	}
}

func TestApplyCodeTaskRepositorySummariesAggregatesMixedRepositories(t *testing.T) {
	summary := codeTaskSummary{}
	repositories := []model.AIDevSessionRepository{
		{Branch: "task-a", Status: codeDeliveryCompleted},
		{Branch: "task-b", Status: codeDeliveryPrepared},
	}
	applyCodeTaskRepositorySummaries(&summary, repositories, make(map[string]codeTaskDiffStats))
	if summary.GitStatus != "committed" || summary.Branch != "task-a" || len(summary.Repositories) != 2 ||
		summary.Repositories[0].Branch != "task-a" || summary.Repositories[1].Branch != "task-b" {
		t.Fatalf("unexpected repository summary: %#v", summary)
	}
}

func TestCodeTaskRepositorySummaryIncludesCanonicalRepositoryPath(t *testing.T) {
	repositoryDir := t.TempDir()
	summary := codeTaskRepositorySummaryFromStats(
		"repository", repositoryDir, "gopanel/code-1", "main", codeTaskDiffStats{Files: 1},
	)
	evaluatedPath, err := filepath.EvalSymlinks(repositoryDir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RepositoryPath != evaluatedPath || summary.Branch != "gopanel/code-1" || summary.TargetBranch != "main" {
		t.Fatalf("unexpected repository identity: %#v", summary)
	}

	legacy := codeTaskRepositorySummaryFromStats("legacy", "", "gopanel/code-old", "", codeTaskDiffStats{})
	if legacy.RepositoryPath != "" {
		t.Fatalf("empty historical repository path normalized incorrectly: %#v", legacy)
	}
}

func TestApplyCodeTaskRepositorySummariesHidesLowerPriorityPushError(t *testing.T) {
	summary := codeTaskSummary{}
	repositories := []model.AIDevSessionRepository{
		{Branch: "task-a", Status: codeDeliveryCompleted, PushStatus: codePushFailed, PushError: "remote rejected"},
		{Branch: "task-b", Status: "conflict"},
	}
	applyCodeTaskRepositorySummaries(&summary, repositories, make(map[string]codeTaskDiffStats))
	if summary.GitStatus != "conflict" || summary.GitError != "" {
		t.Fatalf("unexpected repository summary: %#v", summary)
	}
}

func TestBuildCodeTaskListItemsIncludesStageAndLatestMessages(t *testing.T) {
	database := withCodeGovernanceDB(t)
	lastInstructionAt := time.Now().Add(-10 * time.Minute)
	session := &model.AIDevSession{
		UserID: 1, ProjectID: 1, Title: "session", WorkDir: t.TempDir(),
		CurrentStage: "awaiting_approval", LastInstructionAt: &lastInstructionAt,
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.AITask{UserID: 1, SessionID: session.ID, ProjectID: 1, Title: "task", WorkDir: "/tmp"}
	if err := database.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	olderUserMessage := time.Now().Add(-6 * time.Minute)
	olderAgentMessage := time.Now().Add(-5 * time.Minute)
	newerUserMessage := time.Now().Add(-2 * time.Minute)
	newerAgentMessage := time.Now().Add(-time.Minute)
	messages := []model.AIMessage{
		{CreatedAt: olderUserMessage, SessionID: session.ID, TaskID: task.ID, Role: "user", Content: "旧的指令"},
		{CreatedAt: olderAgentMessage, SessionID: session.ID, TaskID: task.ID, Role: "agent", Content: "旧的回复"},
		{CreatedAt: newerUserMessage, SessionID: session.ID, TaskID: task.ID, Role: "user", Content: "  帮我检查测试\n并修好失败用例  "},
		{CreatedAt: newerAgentMessage, SessionID: session.ID, TaskID: task.ID, Role: "agent", Content: "  正在跑 npm test\n已通过 15 个用例  "},
	}
	if err := database.Create(&messages).Error; err != nil {
		t.Fatal(err)
	}

	// includeGit=false：阶段和最后输出属于纯 SQL 那一档，不该被昂贵路径的开关挡掉。
	items, err := buildCodeTaskListItems([]*model.AITask{task}, false)
	if err != nil {
		t.Fatal(err)
	}
	summary := items[0].Summary
	if summary.Stage != "awaiting_approval" {
		t.Fatalf("stage = %q, want awaiting_approval", summary.Stage)
	}
	if summary.LastUserMessage != "帮我检查测试 并修好失败用例" {
		t.Fatalf("last user message = %q", summary.LastUserMessage)
	}
	if summary.LastAgentMessage != "正在跑 npm test 已通过 15 个用例" {
		t.Fatalf("last agent message = %q", summary.LastAgentMessage)
	}
	if summary.LastActivityAt == nil || summary.LastActivityAt.Before(newerAgentMessage) {
		t.Fatalf("last activity should follow the newest message: %#v", summary.LastActivityAt)
	}
}

func TestTruncateCodeTaskMessage(t *testing.T) {
	if got := truncateCodeTaskMessage("  多行\n输出  "); got != "多行 输出" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("字", codeTaskMessageLimit+40)
	got := truncateCodeTaskMessage(long)
	if []rune(got)[codeTaskMessageLimit] != '…' || len([]rune(got)) != codeTaskMessageLimit+1 {
		t.Fatalf("unexpected truncation length: %d", len([]rune(got)))
	}
	if truncateCodeTaskMessage("   ") != "" {
		t.Fatal("blank content should stay empty")
	}
}
