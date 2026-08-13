package api

import (
	"context"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
)

func buildSystemDiagnosticSnapshot() map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	snapshot := map[string]any{
		"checkedAt":    time.Now().Format(time.RFC3339),
		"controlPlane": service.DiagnoseControlPlane(ctx),
	}
	if global.DB == nil {
		snapshot["databaseError"] = "GoPanel 数据库不可用"
		return snapshot
	}
	snapshot["counts"] = map[string]int64{
		"websites":           diagnosticModelCount(&model.Website{}),
		"databaseServers":    diagnosticModelCount(&model.DatabaseServer{}),
		"backupRecords":      diagnosticModelCount(&model.BackupRecord{}),
		"cronjobs":           diagnosticModelCount(&model.Cronjob{}),
		"aiTasks":            diagnosticModelCount(&model.AITask{}),
		"pendingApprovals":   diagnosticWhereCount(&model.AIApproval{}, "status = ?", "pending"),
		"openSecurityEvents": diagnosticWhereCount(&model.SecurityEvent{}, "status <> ?", model.SecurityEventResolved),
	}
	var failedJobs []model.JobRecords
	_ = global.DB.Where("status NOT IN ?", []string{"success", "succeeded"}).Order("created_at desc").Limit(10).Find(&failedJobs).Error
	for index := range failedJobs {
		failedJobs[index].Message = sanitizeSystemDiagnosticText(failedJobs[index].Message)
	}
	snapshot["recentFailedJobs"] = failedJobs
	var failedRuns []model.AIExecutionRun
	_ = global.DB.Where("status = ?", "failed").Order("created_at desc").Limit(10).Find(&failedRuns).Error
	for index := range failedRuns {
		failedRuns[index].Prompt, failedRuns[index].Output, failedRuns[index].RawOutput = "", "", ""
		failedRuns[index].ErrorMessage = sanitizeSystemDiagnosticText(failedRuns[index].ErrorMessage)
	}
	snapshot["recentFailedAIRuns"] = failedRuns
	var operationLogs []model.OperationLog
	_ = global.DB.Where("status NOT IN ?", []string{"200", "success"}).Order("created_at desc").Limit(10).Find(&operationLogs).Error
	for index := range operationLogs {
		operationLogs[index].Message = sanitizeSystemDiagnosticText(operationLogs[index].Message)
		operationLogs[index].DetailZH = sanitizeSystemDiagnosticText(operationLogs[index].DetailZH)
		operationLogs[index].DetailEN = sanitizeSystemDiagnosticText(operationLogs[index].DetailEN)
	}
	snapshot["recentFailedOperations"] = operationLogs
	return snapshot
}

func diagnosticModelCount(value any) int64 {
	var count int64
	_ = global.DB.Model(value).Count(&count).Error
	return count
}

func diagnosticWhereCount(value any, query string, args ...any) int64 {
	var count int64
	_ = global.DB.Model(value).Where(query, args...).Count(&count).Error
	return count
}
