package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type codeTaskSummary struct {
	DurationMS            int64                       `json:"durationMs"`
	TotalTokens           int64                       `json:"totalTokens"`
	TokenUsageStatus      string                      `json:"tokenUsageStatus"`
	TokenRecoveredRuns    int64                       `json:"tokenRecoveredRuns"`
	TokenUnavailableRuns  int64                       `json:"tokenUnavailableRuns"`
	TokenPendingRuns      int64                       `json:"tokenPendingRuns"`
	Executor              string                      `json:"executor,omitempty"`
	Model                 string                      `json:"model,omitempty"`
	GitStatus             string                      `json:"gitStatus,omitempty"`
	GitError              string                      `json:"gitError,omitempty"`
	Branch                string                      `json:"branch,omitempty"`
	Repositories          []codeTaskRepositorySummary `json:"repositories,omitempty"`
	Additions             int                         `json:"additions"`
	Deletions             int                         `json:"deletions"`
	ChangedFiles          int                         `json:"changedFiles"`
	HasDiff               bool                        `json:"hasDiff"`
	UnsavedAdditions      int                         `json:"unsavedAdditions"`
	UnsavedDeletions      int                         `json:"unsavedDeletions"`
	UnsavedFiles          int                         `json:"unsavedFiles"`
	HasUnsavedChanges     bool                        `json:"hasUnsavedChanges"`
	DeliveryStatus        string                      `json:"deliveryStatus,omitempty"`
	DeliveryStage         string                      `json:"deliveryStage,omitempty"`
	DeliveryProgress      int                         `json:"deliveryProgress"`
	DeliveryQueuePosition int                         `json:"deliveryQueuePosition"`
	DeliveryAttempt       int                         `json:"deliveryAttempt"`
	DeliveryResultType    string                      `json:"deliveryResultType,omitempty"`
	DeliveryError         string                      `json:"deliveryError,omitempty"`
	// 会话当前阶段（executing / awaiting_approval / instruction_queued / completed…）。
	// 比任务 status 细一档，用来回答「卡在哪一步」。
	Stage string `json:"stage,omitempty"`
	// 最后一条用户消息和执行器回复（截断）。对话已经由 code_native_history 固化进 ai_messages，
	// 所以拿这些不用碰终端、不用解析 codex 的 rollout 文件，一次 SQL 就有。
	LastUserMessage  string     `json:"lastUserMessage,omitempty"`
	LastAgentMessage string     `json:"lastAgentMessage,omitempty"`
	LastActivityAt   *time.Time `json:"lastActivityAt,omitempty"`
}

type codeTaskRepositorySummary struct {
	Name           string `json:"name"`
	RepositoryPath string `json:"repositoryPath,omitempty"`
	Branch         string `json:"branch"`
	TargetBranch   string `json:"targetBranch,omitempty"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	ChangedFiles   int    `json:"changedFiles"`
	HasDiff        bool   `json:"hasDiff"`
}

// 列表里只需要一眼扫过的摘要，整段回复没必要传给前端。
const codeTaskMessageLimit = 160

func truncateCodeTaskMessage(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	// 折叠换行：多行回复在单行行内显示时，换行只会变成一堆空白。
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	runes := []rune(trimmed)
	if len(runes) <= codeTaskMessageLimit {
		return trimmed
	}
	return string(runes[:codeTaskMessageLimit]) + "…"
}

type codeTaskListItem struct {
	*model.AITask
	Summary codeTaskSummary `json:"summary"`
}

type codeTaskRunSummaryRow struct {
	TaskID     uint
	DurationMS int64
	ExecutorID string
	Model      string
}

type codeTaskDurationSummaryRow struct {
	TaskID          uint
	DurationMS      int64
	TotalTokens     int64
	RecordedRuns    int64
	RecoveredRuns   int64
	UnavailableRuns int64
	PendingRuns     int64
}

type codeTaskGitScope string

const (
	codeTaskGitScopeFull codeTaskGitScope = "full"
	codeTaskGitScopeLive codeTaskGitScope = "live"
)

type codeTaskListOptions struct {
	IncludeGit     bool
	GitScope       codeTaskGitScope
	SelectedTaskID uint
}

func buildCodeTaskListItems(tasks []*model.AITask, includeGit bool) ([]codeTaskListItem, error) {
	return buildCodeTaskListItemsWithOptions(tasks, codeTaskListOptions{IncludeGit: includeGit, GitScope: codeTaskGitScopeFull})
}

func buildCodeTaskListItemsWithOptions(tasks []*model.AITask, options codeTaskListOptions) ([]codeTaskListItem, error) {
	items := make([]codeTaskListItem, 0, len(tasks))
	if len(tasks) == 0 {
		return items, nil
	}
	taskIDs := make([]uint, 0, len(tasks))
	sessionIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
		if task.SessionID > 0 {
			sessionIDs = append(sessionIDs, task.SessionID)
		}
	}
	summaries := make(map[uint]codeTaskSummary, len(tasks))
	if err := loadCodeTaskRunSummaries(taskIDs, summaries); err != nil {
		return nil, err
	}
	if err := loadCodeTaskDeliverySummaries(sessionIDs, summaries, tasks); err != nil {
		return nil, err
	}
	if err := loadCodeTaskActivitySummaries(tasks, sessionIDs, summaries); err != nil {
		return nil, err
	}
	if options.IncludeGit {
		gitTasks := selectCodeTasksForGitSummary(tasks, options.GitScope, options.SelectedTaskID)
		gitSessionIDs := make([]uint, 0, len(gitTasks))
		for _, task := range gitTasks {
			if task.SessionID > 0 {
				gitSessionIDs = append(gitSessionIDs, task.SessionID)
			}
		}
		if err := loadCodeTaskGitSummaries(gitTasks, gitSessionIDs, summaries, make(map[string]codeTaskDiffStats)); err != nil {
			return nil, err
		}
	}
	for _, task := range tasks {
		items = append(items, codeTaskListItem{AITask: task, Summary: summaries[task.ID]})
	}
	return items, nil
}

func selectCodeTasksForGitSummary(tasks []*model.AITask, scope codeTaskGitScope, selectedTaskID uint) []*model.AITask {
	if scope != codeTaskGitScopeLive {
		return tasks
	}
	selected := make([]*model.AITask, 0, len(tasks))
	for _, task := range tasks {
		if task.ID == selectedTaskID || codeTaskNeedsLiveGitSummary(task.Status) {
			selected = append(selected, task)
		}
	}
	return selected
}

func codeTaskNeedsLiveGitSummary(status string) bool {
	switch status {
	case "active", "queued", "running", "pending_approval", "delivering":
		return true
	default:
		return false
	}
}

func loadCodeTaskDeliverySummaries(sessionIDs []uint, summaries map[uint]codeTaskSummary, tasks []*model.AITask) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	var jobs []model.AICodeDeliveryJob
	if err := global.DB.Where("session_id IN ?", sessionIDs).Find(&jobs).Error; err != nil {
		return err
	}
	jobsBySession := make(map[uint]model.AICodeDeliveryJob, len(jobs))
	for _, job := range jobs {
		jobsBySession[job.SessionID] = job
	}
	for _, task := range tasks {
		job, exists := jobsBySession[task.SessionID]
		if !exists || (job.TaskID > 0 && job.TaskID != task.ID) {
			continue
		}
		summary := summaries[task.ID]
		summary.DeliveryStatus = job.Status
		summary.DeliveryStage = job.Stage
		summary.DeliveryProgress = job.Progress
		summary.DeliveryAttempt = job.Attempt
		summary.DeliveryResultType = job.ResultType
		summary.DeliveryError = job.ErrorMessage
		var repositories []codeRepositoryDeliveryResult
		if json.Unmarshal([]byte(job.RepositoryResults), &repositories) == nil {
			applyCodeTaskStoredRepositorySummaries(&summary, repositories)
		}
		if job.Status == codeDeliveryJobQueued {
			if view, err := loadCodeDeliveryJobView(job.SessionID); err == nil {
				summary.DeliveryQueuePosition = view.QueuePosition
			}
		}
		summaries[task.ID] = summary
	}
	return nil
}

func loadCodeTaskRunSummaries(taskIDs []uint, summaries map[uint]codeTaskSummary) error {
	runQuery := global.DB.Model(&model.AIExecutionRun{}).Where("task_id IN ?", taskIDs)
	var durations []codeTaskDurationSummaryRow
	err := runQuery.
		Select(`task_id, COALESCE(SUM(duration_ms), 0) AS duration_ms, COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(CASE WHEN token_usage_status = 'recorded' THEN 1 ELSE 0 END), 0) AS recorded_runs,
			COALESCE(SUM(CASE WHEN token_usage_status = 'recovered' THEN 1 ELSE 0 END), 0) AS recovered_runs,
			COALESCE(SUM(CASE WHEN token_usage_status = 'unavailable' THEN 1 ELSE 0 END), 0) AS unavailable_runs,
			COALESCE(SUM(CASE WHEN token_usage_status = 'pending' OR token_usage_status = '' THEN 1 ELSE 0 END), 0) AS pending_runs`).
		Group("task_id").Scan(&durations).Error
	if err != nil {
		return err
	}
	for _, row := range durations {
		summary := summaries[row.TaskID]
		summary.DurationMS = row.DurationMS
		summary.TotalTokens = row.TotalTokens
		summary.TokenRecoveredRuns = row.RecoveredRuns
		summary.TokenUnavailableRuns = row.UnavailableRuns
		summary.TokenPendingRuns = row.PendingRuns
		summary.TokenUsageStatus = codeTaskTokenUsageStatus(row)
		summaries[row.TaskID] = summary
	}
	var latestRuns []codeTaskRunSummaryRow
	rankedRuns := global.DB.Model(&model.AIExecutionRun{}).
		Select("task_id, executor_id, model, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY created_at DESC, id DESC) AS row_number").
		Where("task_id IN ?", taskIDs)
	if err := global.DB.Table("(?) AS ranked_runs", rankedRuns).
		Select("task_id, executor_id, model").Where("row_number = 1").Scan(&latestRuns).Error; err != nil {
		return err
	}
	for _, row := range latestRuns {
		summary := summaries[row.TaskID]
		summary.Executor = row.ExecutorID
		summary.Model = row.Model
		summaries[row.TaskID] = summary
	}
	return nil
}

// loadCodeTaskActivitySummaries 补上「现在卡在哪一步」和对话双方最后说了什么。
//
// 两条纯 SQL，刻意不放在 includeGit 门控里：git 汇总要按会话读工作区算 diff（磁盘 IO），
// 这两条只是走索引的批量查询，每轮都带得起。
//
// 另一条没走的路：codex 的 lastAssistantPreview 信息更好，但它要在磁盘上定位并解析
// rollout 文件（见 getCodexRuntimeState），还只对 codex 执行器有效 ——
// 列表里 N 条任务就是 N 次文件解析，扛不住。ai_messages 里已经固化了同样的对话。
func loadCodeTaskActivitySummaries(tasks []*model.AITask, sessionIDs []uint, summaries map[uint]codeTaskSummary) error {
	taskIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	if len(taskIDs) == 0 {
		return nil
	}

	if len(sessionIDs) > 0 {
		type sessionStageRow struct {
			ID                uint
			CurrentStage      string
			LastInstructionAt *time.Time
		}
		var stages []sessionStageRow
		if err := global.DB.Model(&model.AIDevSession{}).
			Select("id, current_stage, last_instruction_at").
			Where("id IN ?", sessionIDs).Scan(&stages).Error; err != nil {
			return err
		}
		stagesBySession := make(map[uint]sessionStageRow, len(stages))
		for _, row := range stages {
			stagesBySession[row.ID] = row
		}
		for _, task := range tasks {
			row, exists := stagesBySession[task.SessionID]
			if !exists {
				continue
			}
			summary := summaries[task.ID]
			summary.Stage = row.CurrentStage
			summary.LastActivityAt = row.LastInstructionAt
			summaries[task.ID] = summary
		}
	}

	// 每个任务分别取最新一条 user / agent 消息，写法和 loadCodeTaskRunSummaries 里取最新 run 一致。
	type latestMessageRow struct {
		TaskID    uint
		Role      string
		Content   string
		CreatedAt time.Time
	}
	var latestMessages []latestMessageRow
	rankedMessages := global.DB.Model(&model.AIMessage{}).
		Select("task_id, role, content, created_at, ROW_NUMBER() OVER (PARTITION BY task_id, role ORDER BY created_at DESC, id DESC) AS row_number").
		Where("task_id IN ? AND role IN ?", taskIDs, []string{"user", "agent"})
	if err := global.DB.Table("(?) AS ranked_messages", rankedMessages).
		Select("task_id, role, content, created_at").Where("row_number = 1").Scan(&latestMessages).Error; err != nil {
		return err
	}
	for _, row := range latestMessages {
		summary := summaries[row.TaskID]
		if row.Role == "user" {
			summary.LastUserMessage = truncateCodeTaskMessage(row.Content)
		} else {
			summary.LastAgentMessage = truncateCodeTaskMessage(row.Content)
		}
		// 有实际消息时，它比 last_instruction_at 更能代表「最后一次动静」。
		if summary.LastActivityAt == nil || row.CreatedAt.After(*summary.LastActivityAt) {
			createdAt := row.CreatedAt
			summary.LastActivityAt = &createdAt
		}
		summaries[row.TaskID] = summary
	}
	return nil
}

func codeTaskTokenUsageStatus(row codeTaskDurationSummaryRow) string {
	knownRuns := row.RecordedRuns + row.RecoveredRuns
	missingRuns := row.UnavailableRuns + row.PendingRuns
	if missingRuns == 0 {
		if row.RecoveredRuns > 0 {
			return codeTokenUsageRecovered
		}
		return codeTokenUsageRecorded
	}
	if knownRuns > 0 {
		return "partial"
	}
	if row.PendingRuns > 0 {
		return codeTokenUsagePending
	}
	return codeTokenUsageUnavailable
}
