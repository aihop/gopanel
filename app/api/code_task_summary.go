package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type codeTaskSummary struct {
	DurationMS            int64  `json:"durationMs"`
	Executor              string `json:"executor,omitempty"`
	Model                 string `json:"model,omitempty"`
	GitStatus             string `json:"gitStatus,omitempty"`
	GitError              string `json:"gitError,omitempty"`
	Branch                string `json:"branch,omitempty"`
	Additions             int    `json:"additions"`
	Deletions             int    `json:"deletions"`
	ChangedFiles          int    `json:"changedFiles"`
	HasDiff               bool   `json:"hasDiff"`
	DeliveryStatus        string `json:"deliveryStatus,omitempty"`
	DeliveryStage         string `json:"deliveryStage,omitempty"`
	DeliveryProgress      int    `json:"deliveryProgress"`
	DeliveryQueuePosition int    `json:"deliveryQueuePosition"`
	DeliveryAttempt       int    `json:"deliveryAttempt"`
	DeliveryResultType    string `json:"deliveryResultType,omitempty"`
	DeliveryError         string `json:"deliveryError,omitempty"`
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
	TaskID     uint
	DurationMS int64
}

type codeTaskDiffStats struct {
	Additions int
	Deletions int
	Files     int
}

func buildCodeTaskListItems(tasks []*model.AITask, includeGit bool) ([]codeTaskListItem, error) {
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
	if includeGit {
		if err := loadCodeTaskGitSummaries(tasks, sessionIDs, summaries, make(map[string]codeTaskDiffStats)); err != nil {
			return nil, err
		}
	}
	for _, task := range tasks {
		items = append(items, codeTaskListItem{AITask: task, Summary: summaries[task.ID]})
	}
	return items, nil
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
	var durations []codeTaskDurationSummaryRow
	err := global.DB.Model(&model.AIExecutionRun{}).
		Select("task_id, COALESCE(SUM(duration_ms), 0) AS duration_ms").
		Where("task_id IN ?", taskIDs).
		Group("task_id").Scan(&durations).Error
	if err != nil {
		return err
	}
	for _, row := range durations {
		summary := summaries[row.TaskID]
		summary.DurationMS = row.DurationMS
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

func loadCodeTaskGitSummaries(tasks []*model.AITask, sessionIDs []uint, summaries map[uint]codeTaskSummary, diffStatsCache map[string]codeTaskDiffStats) error {
	if len(sessionIDs) == 0 {
		return nil
	}
	var sessions []model.AIDevSession
	if err := global.DB.Select("id, work_dir, source_work_dir, worktree_branch, base_commit, isolation_mode").Where("id IN ?", sessionIDs).Find(&sessions).Error; err != nil {
		return err
	}
	var deliveries []model.AICodeDelivery
	if err := global.DB.Where("session_id IN ?", sessionIDs).Find(&deliveries).Error; err != nil {
		return err
	}
	var repositories []model.AIDevSessionRepository
	if err := global.DB.Where("session_id IN ?", sessionIDs).Find(&repositories).Error; err != nil {
		return err
	}
	sessionsByID := make(map[uint]model.AIDevSession, len(sessions))
	for _, session := range sessions {
		sessionsByID[session.ID] = session
	}
	deliveriesBySession := make(map[uint]model.AICodeDelivery, len(deliveries))
	for _, delivery := range deliveries {
		deliveriesBySession[delivery.SessionID] = delivery
	}
	repositoriesBySession := make(map[uint][]model.AIDevSessionRepository)
	for _, repository := range repositories {
		repositoriesBySession[repository.SessionID] = append(repositoriesBySession[repository.SessionID], repository)
	}
	for _, task := range tasks {
		summary := summaries[task.ID]
		session := sessionsByID[task.SessionID]
		summary.Branch = session.WorktreeBranch
		if session.IsolationMode == codeIsolationMultiWorktree || len(repositoriesBySession[task.SessionID]) > 0 {
			applyCodeTaskRepositorySummaries(&summary, repositoriesBySession[task.SessionID], diffStatsCache)
		} else if delivery, exists := deliveriesBySession[task.SessionID]; exists {
			applyCodeTaskDeliverySummary(&summary, delivery, diffStatsCache)
		} else if summary.Branch != "" {
			applyCodeTaskWorktreeSummary(&summary, session.WorkDir, session.BaseCommit, diffStatsCache)
		}
		summaries[task.ID] = summary
	}
	return nil
}

func applyCodeTaskDeliverySummary(summary *codeTaskSummary, delivery model.AICodeDelivery, diffStatsCache map[string]codeTaskDiffStats) {
	summary.Branch = delivery.WorktreeBranch
	summary.GitStatus = codeTaskDeliveryStatus(delivery.Status, delivery.PushStatus)
	if summary.GitStatus == "push_failed" {
		summary.GitError = delivery.PushError
	}
	// 优先用交付快照时固化的统计：worktree 提交可能已被回收，实时 diff 会静默算不出来。
	if delivery.StatFiles > 0 {
		applyCodeTaskDiffStats(summary, codeTaskDiffStats{
			Additions: delivery.StatAdditions, Deletions: delivery.StatDeletions, Files: delivery.StatFiles,
		})
		return
	}
	if delivery.BaseCommit == "" || delivery.WorktreeCommit == "" {
		return
	}
	stats, ok := loadCodeTaskDiffStats(delivery.SourceWorkDir, delivery.BaseCommit, delivery.WorktreeCommit, diffStatsCache)
	if ok {
		applyCodeTaskDiffStats(summary, stats)
	}
}

func applyCodeTaskRepositorySummaries(summary *codeTaskSummary, repositories []model.AIDevSessionRepository, diffStatsCache map[string]codeTaskDiffStats) {
	if len(repositories) == 0 {
		return
	}
	statuses := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		if summary.Branch == "" {
			summary.Branch = repository.Branch
		}
		repositoryStatus := codeTaskDeliveryStatus(repository.Status, repository.PushStatus)
		if repositoryStatus == "push_failed" && summary.GitError == "" {
			summary.GitError = repository.PushError
		}
		if repository.WorktreeCommit == "" && repositoryStatus == "working" {
			worktreeSummary := codeTaskSummary{GitStatus: repositoryStatus}
			applyCodeTaskWorktreeSummary(&worktreeSummary, repository.WorktreeDir, repository.BaseCommit, diffStatsCache)
			repositoryStatus = worktreeSummary.GitStatus
			if worktreeSummary.HasDiff {
				applyCodeTaskDiffStats(summary, codeTaskDiffStats{
					Additions: worktreeSummary.Additions,
					Deletions: worktreeSummary.Deletions,
					Files:     worktreeSummary.ChangedFiles,
				})
			}
		}
		statuses = append(statuses, repositoryStatus)
		// 优先用交付快照时固化的统计，理由同单仓路径。
		if repository.StatFiles > 0 {
			applyCodeTaskDiffStats(summary, codeTaskDiffStats{
				Additions: repository.StatAdditions, Deletions: repository.StatDeletions,
				Files: repository.StatFiles,
			})
			continue
		}
		if repository.BaseCommit == "" || repository.WorktreeCommit == "" {
			continue
		}
		if stats, ok := loadCodeTaskDiffStats(repository.SourceDir, repository.BaseCommit, repository.WorktreeCommit, diffStatsCache); ok {
			applyCodeTaskDiffStats(summary, stats)
		}
	}
	summary.GitStatus = aggregateCodeTaskGitStatuses(statuses)
	if summary.GitStatus != "push_failed" {
		summary.GitError = ""
	}
}

func applyCodeTaskWorktreeSummary(summary *codeTaskSummary, worktreeDir, baseCommit string, diffStatsCache map[string]codeTaskDiffStats) {
	summary.GitStatus = "working"
	if strings.TrimSpace(worktreeDir) == "" || strings.TrimSpace(baseCommit) == "" {
		return
	}
	headCommit, err := runCodeGit(worktreeDir, "rev-parse", "HEAD")
	if err != nil || headCommit == baseCommit {
		return
	}
	status, err := runCodeGit(worktreeDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) != "" {
		return
	}
	summary.GitStatus = "committed"
	if stats, ok := loadCodeTaskDiffStats(worktreeDir, baseCommit, headCommit, diffStatsCache); ok {
		applyCodeTaskDiffStats(summary, stats)
	}
}

func codeTaskDeliveryStatus(status, pushStatus string) string {
	if status == "conflict" {
		return "conflict"
	}
	if pushStatus == codePushFailed {
		return "push_failed"
	}
	if pushStatus == "pushed" {
		return "pushed"
	}
	switch status {
	case codeDeliveryCompleted, codeDeliveryMerged, codeDeliveryWorktreeCleaned:
		return "merged"
	case codeDeliveryPrepared:
		return "committed"
	default:
		return "working"
	}
}

func aggregateCodeTaskGitStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return ""
	}
	for _, candidate := range []string{"conflict", "push_failed", "working", "committed", "merged"} {
		for _, status := range statuses {
			if status == candidate {
				return candidate
			}
		}
	}
	return "pushed"
}

// codeTaskDiffExcludedFiles 是统计任务产出行数时排除的机器生成文件。
// 一次依赖安装刷新 lock 文件就能贡献上万行，让「这个任务改了多少代码」完全失真。
// 只排除公认由工具生成的 lock 文件：dist/、vendor/ 之类一旦进了版本库，
// 说明项目有意跟踪它们，排除掉反而会漏算真实改动。
var codeTaskDiffExcludedFiles = []string{
	"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
	"go.sum", "Cargo.lock", "composer.lock", "Gemfile.lock",
	"Podfile.lock", "poetry.lock", "Pipfile.lock", "pubspec.lock",
}

// codeTaskDiffPathspec 生成排除用的 pathspec。
// 前缀 * 让匹配跨目录层级——裸文件名只会匹配仓库根目录下的同名文件，
// 子目录里的 lock 文件会漏网。
func codeTaskDiffPathspec() []string {
	pathspec := make([]string, 0, len(codeTaskDiffExcludedFiles)+1)
	pathspec = append(pathspec, "--")
	for _, name := range codeTaskDiffExcludedFiles {
		pathspec = append(pathspec, ":(exclude)*"+name)
	}
	return pathspec
}

func loadCodeTaskDiffStats(root, baseCommit, headCommit string, cache map[string]codeTaskDiffStats) (codeTaskDiffStats, bool) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || baseCommit == "" || headCommit == "" {
		return codeTaskDiffStats{}, false
	}
	cacheKey := fmt.Sprintf("%s\x00%s\x00%s", root, baseCommit, headCommit)
	if cached, exists := cache[cacheKey]; exists {
		return cached, true
	}
	args := append(
		[]string{"diff", "--numstat", "--no-renames", baseCommit + ".." + headCommit},
		codeTaskDiffPathspec()...,
	)
	output, err := runCodeGit(root, args...)
	if err != nil {
		return codeTaskDiffStats{}, false
	}
	stats := codeTaskDiffStats{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		stats.Files++
		added, deleted := parseCodeGitNumstat(line)
		stats.Additions += added
		stats.Deletions += deleted
	}
	cache[cacheKey] = stats
	return stats, true
}

func applyCodeTaskDiffStats(summary *codeTaskSummary, stats codeTaskDiffStats) {
	if stats.Files == 0 {
		return
	}
	summary.HasDiff = true
	summary.Additions += stats.Additions
	summary.Deletions += stats.Deletions
	summary.ChangedFiles += stats.Files
}
