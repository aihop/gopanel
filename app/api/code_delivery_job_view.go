package api

import (
	"encoding/json"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 交付作业的展示视图：把队列里的作业状态整理成前端需要的形状。
// 与队列的调度、租约、执行逻辑分开，后者在 code_delivery_queue.go。

type codeDeliveryJobView struct {
	ID                    uint                           `json:"id"`
	SessionID             uint                           `json:"sessionId"`
	TaskID                uint                           `json:"taskId,omitempty"`
	Status                string                         `json:"status"`
	Stage                 string                         `json:"stage"`
	Progress              int                            `json:"progress"`
	Attempt               int                            `json:"attempt"`
	QueuePosition         int                            `json:"queuePosition"`
	TargetBranch          string                         `json:"targetBranch,omitempty"`
	ResultCommit          string                         `json:"resultCommit,omitempty"`
	ResultType            string                         `json:"resultType,omitempty"`
	FailureCode           string                         `json:"failureCode,omitempty"`
	HasPendingChanges     bool                           `json:"hasPendingChanges"`
	HasPendingCommits     bool                           `json:"hasPendingCommits"`
	HasUncommittedChanges bool                           `json:"hasUncommittedChanges"`
	Repositories          []codeRepositoryDeliveryResult `json:"repositories,omitempty"`
	Facts                 []codeDeliveryFact             `json:"facts,omitempty"`
	ErrorMessage          string                         `json:"errorMessage,omitempty"`
	ConflictFiles         []string                       `json:"conflictFiles"`
	CreatedAt             time.Time                      `json:"createdAt"`
	UpdatedAt             time.Time                      `json:"updatedAt"`
	StartedAt             *time.Time                     `json:"startedAt,omitempty"`
	CompletedAt           *time.Time                     `json:"completedAt,omitempty"`
}

func loadCodeDeliveryJobView(sessionID uint) (*codeDeliveryJobView, error) {
	var job model.AICodeDeliveryJob
	if err := global.DB.Where("session_id = ?", sessionID).First(&job).Error; err != nil {
		return nil, err
	}
	var conflicts []string
	_ = json.Unmarshal([]byte(job.ConflictFiles), &conflicts)
	var repositories []codeRepositoryDeliveryResult
	_ = json.Unmarshal([]byte(job.RepositoryResults), &repositories)
	position := 0
	if job.Status == codeDeliveryJobQueued {
		var count int64
		_ = global.DB.Model(&model.AICodeDeliveryJob{}).Where(
			"status IN ? AND (created_at < ? OR (created_at = ? AND id < ?))",
			[]string{codeDeliveryJobQueued, codeDeliveryJobRunning}, job.CreatedAt, job.CreatedAt, job.ID,
		).Count(&count).Error
		position = int(count) + 1
	}
	pending := codeSessionPostSnapshotStatus{}
	if job.Status == codeDeliveryJobCompleted {
		pending = inspectCodeSessionPostSnapshotStatus(global.DB, sessionID)
	}
	return &codeDeliveryJobView{
		ID: job.ID, SessionID: job.SessionID, TaskID: job.TaskID, Status: job.Status, Stage: job.Stage, Progress: job.Progress,
		Attempt: job.Attempt, QueuePosition: position, TargetBranch: job.TargetBranch, ResultCommit: job.ResultCommit,
		ResultType: job.ResultType, FailureCode: job.FailureCode,
		HasPendingChanges: pending.HasChanges, HasPendingCommits: pending.HasCommits,
		HasUncommittedChanges: pending.HasUncommittedChanges,
		Repositories:          repositories,
		Facts:                 loadCodeDeliveryFacts(sessionID, repositories),
		ErrorMessage:          job.ErrorMessage, ConflictFiles: conflicts, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		StartedAt: job.StartedAt, CompletedAt: job.CompletedAt,
	}, nil
}
