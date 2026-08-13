package model

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
