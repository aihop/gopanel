package model

import "time"

type Flow struct {
	BaseModel
	ProjectID                  uint              `gorm:"column:project_id;not null;uniqueIndex" json:"projectId"`
	Name                       string            `gorm:"column:name;type:varchar(100);not null" json:"name"`
	PipelineID                 uint              `gorm:"column:pipeline_id;not null;index" json:"pipelineId"`
	Enabled                    bool              `gorm:"column:enabled;not null;default:true;index" json:"enabled"`
	AutoStartAfterCodeDelivery bool              `gorm:"column:auto_start_after_code_delivery;not null;default:false" json:"autoStartAfterCodeDelivery"`
	CreatedBy                  uint              `gorm:"column:created_by;not null;index" json:"createdBy"`
	ProjectName                string            `gorm:"-" json:"projectName"`
	PipelineName               string            `gorm:"-" json:"pipelineName"`
	Environments               []FlowEnvironment `gorm:"foreignKey:FlowID" json:"environments"`
}

func (Flow) TableName() string {
	return "flows"
}

type FlowEnvironment struct {
	BaseModel
	FlowID                          uint   `gorm:"column:flow_id;not null;uniqueIndex:uniq_flow_environment" json:"flowId"`
	Name                            string `gorm:"column:name;type:varchar(32);not null;uniqueIndex:uniq_flow_environment" json:"name"`
	WebsiteID                       uint   `gorm:"column:website_id;not null;index" json:"websiteId"`
	AutoDeploy                      bool   `gorm:"column:auto_deploy;not null;default:false" json:"autoDeploy"`
	ApprovalRequired                bool   `gorm:"column:approval_required;not null" json:"approvalRequired"`
	HealthCheckSuccessCount         int    `gorm:"column:health_check_success_count;not null;default:2" json:"healthCheckSuccessCount"`
	ExternalVerifyTimeoutSeconds    int    `gorm:"column:external_verify_timeout_seconds;not null;default:60" json:"externalVerifyTimeoutSeconds"`
	StabilizationMinutes            int    `gorm:"column:stabilization_minutes;not null;default:5" json:"stabilizationMinutes"`
	RuntimeMonitorEnabled           bool   `gorm:"column:runtime_monitor_enabled;not null;default:true" json:"runtimeMonitorEnabled"`
	RuntimeFailureThreshold         int    `gorm:"column:runtime_failure_threshold;not null;default:3" json:"runtimeFailureThreshold"`
	RuntimeRecoveryThreshold        int    `gorm:"column:runtime_recovery_threshold;not null;default:2" json:"runtimeRecoveryThreshold"`
	AutoRollbackDuringStabilization bool   `gorm:"column:auto_rollback_during_stabilization;not null;default:true" json:"autoRollbackDuringStabilization"`
	RetainPreviousMinutes           int    `gorm:"column:retain_previous_minutes;not null;default:30" json:"retainPreviousMinutes"`
	Sort                            int    `gorm:"column:sort;not null;default:0" json:"sort"`
	Enabled                         bool   `gorm:"column:enabled;not null;default:true;index" json:"enabled"`
	WebsiteName                     string `gorm:"-" json:"websiteName"`
}

func (FlowEnvironment) TableName() string {
	return "flow_environments"
}

type FlowRun struct {
	BaseModel
	FlowID            uint           `gorm:"column:flow_id;not null;uniqueIndex:uniq_flow_run_version;index" json:"flowId"`
	ProjectID         uint           `gorm:"column:project_id;not null;index" json:"projectId"`
	PipelineID        uint           `gorm:"column:pipeline_id;not null;index" json:"pipelineId"`
	Version           string         `gorm:"column:version;type:varchar(50);not null;uniqueIndex:uniq_flow_run_version" json:"version"`
	SourceRepository  string         `gorm:"column:source_repository;type:varchar(512)" json:"-"`
	SourceBranch      string         `gorm:"column:source_branch;type:varchar(255)" json:"sourceBranch"`
	SourceCommit      string         `gorm:"column:source_commit;type:varchar(64);not null;index" json:"sourceCommit"`
	SessionID         uint           `gorm:"column:session_id;index" json:"sessionId"`
	TaskID            uint           `gorm:"column:task_id;index" json:"taskId"`
	CodeDeliveryJobID uint           `gorm:"column:code_delivery_job_id;index" json:"codeDeliveryJobId"`
	PipelineRecordID  uint           `gorm:"column:pipeline_record_id;index" json:"pipelineRecordId"`
	ReleaseID         uint           `gorm:"column:release_id;index" json:"releaseId"`
	CurrentStage      string         `gorm:"column:current_stage;type:varchar(32);not null;index" json:"currentStage"`
	Status            string         `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	FailureCode       string         `gorm:"column:failure_code;type:varchar(64)" json:"failureCode"`
	ErrorSummary      string         `gorm:"column:error_summary;type:text" json:"errorSummary"`
	RequestedBy       uint           `gorm:"column:requested_by;not null;index" json:"requestedBy"`
	StartedAt         *time.Time     `gorm:"column:started_at" json:"startedAt,omitempty"`
	CompletedAt       *time.Time     `gorm:"column:completed_at" json:"completedAt,omitempty"`
	LeaseOwner        string         `gorm:"column:lease_owner;type:varchar(128);index" json:"-"`
	LeaseExpiresAt    *time.Time     `gorm:"column:lease_expires_at;index" json:"-"`
	FlowName          string         `gorm:"-" json:"flowName"`
	ProjectName       string         `gorm:"-" json:"projectName"`
	PipelineName      string         `gorm:"-" json:"pipelineName"`
	ArtifactDigest    string         `gorm:"-" json:"artifactDigest"`
	Stages            []FlowStageRun `gorm:"foreignKey:FlowRunID" json:"stages,omitempty"`
}

func (FlowRun) TableName() string {
	return "flow_runs"
}

type FlowStageRun struct {
	BaseModel
	FlowRunID      uint       `gorm:"column:flow_run_id;not null;uniqueIndex:uniq_flow_stage_attempt;index" json:"flowRunId"`
	Stage          string     `gorm:"column:stage;type:varchar(32);not null;uniqueIndex:uniq_flow_stage_attempt" json:"stage"`
	Attempt        int        `gorm:"column:attempt;not null;default:1;uniqueIndex:uniq_flow_stage_attempt" json:"attempt"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	IdempotencyKey string     `gorm:"column:idempotency_key;type:varchar(128);not null;index" json:"idempotencyKey"`
	ResourceType   string     `gorm:"column:resource_type;type:varchar(32)" json:"resourceType"`
	ResourceID     uint       `gorm:"column:resource_id;index" json:"resourceId"`
	Summary        string     `gorm:"column:summary;type:varchar(255)" json:"summary"`
	ErrorCode      string     `gorm:"column:error_code;type:varchar(64)" json:"errorCode"`
	ErrorDetail    string     `gorm:"column:error_detail;type:text" json:"errorDetail"`
	StartedAt      *time.Time `gorm:"column:started_at" json:"startedAt,omitempty"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (FlowStageRun) TableName() string {
	return "flow_stage_runs"
}
