package request

// CronjobCreate 创建计划任务
type CronjobCreate struct {
	Name string `json:"name" validate:"required"`
	Type string `json:"type" validate:"required,oneof=shell db_backup log_clean ssl_renew"`
	Spec string `json:"spec" validate:"required"`

	// shell
	Script string `json:"script"`

	// db_backup
	ServerID     uint   `json:"serverID"`
	DBType       string `json:"dbType"`
	DBName       string `json:"dbName"`
	RetainCopies int    `json:"retainCopies"`

	// log_clean
	LogType string `json:"logType"`
}

// CronjobUpdate 更新计划任务
type CronjobUpdate struct {
	ID   uint   `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
	Type string `json:"type" validate:"required,oneof=shell db_backup log_clean ssl_renew"`
	Spec string `json:"spec" validate:"required"`

	Script string `json:"script"`

	ServerID     uint   `json:"serverID"`
	DBType       string `json:"dbType"`
	DBName       string `json:"dbName"`
	RetainCopies int    `json:"retainCopies"`

	LogType string `json:"logType"`
}

// CronjobSetStatus 启用/禁用计划任务
type CronjobSetStatus struct {
	ID     uint   `json:"id" validate:"required"`
	Status string `json:"status" validate:"required,oneof=Enable Disable"`
}

// CronjobRecordList 查询某个计划任务的执行记录
type CronjobRecordList struct {
	CronjobID uint `json:"cronjobID" validate:"required"`
	Limit     int  `json:"limit"`
}

// CronjobRecordDelete 清空某个计划任务的执行记录
type CronjobRecordDelete struct {
	CronjobID uint `json:"cronjobID" validate:"required"`
}
