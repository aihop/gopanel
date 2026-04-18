package request

type GetTablesReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
}

type SearchCondition struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type GetTableDataReq struct {
	ServerID       uint              `json:"serverId" validate:"required"`
	DatabaseName   string            `json:"databaseName" validate:"required"`
	TableName      string            `json:"tableName" validate:"required"`
	Page           int               `json:"page" validate:"required,min=1"`
	PageSize       int               `json:"pageSize" validate:"required,min=1,max=100"`
	SearchColumn   string            `json:"searchColumn"`
	SearchValue    string            `json:"searchValue"`
	AdvancedSearch []SearchCondition `json:"advancedSearch"`
}

type ExecSqlReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	SQL          string `json:"sql" validate:"required"`
}

type InsertRecordReq struct {
	ServerID     uint                   `json:"serverId" validate:"required"`
	DatabaseName string                 `json:"databaseName" validate:"required"`
	TableName    string                 `json:"tableName" validate:"required"`
	Data         map[string]interface{} `json:"data" validate:"required"`
}

type UpdateRecordReq struct {
	ServerID     uint                   `json:"serverId" validate:"required"`
	DatabaseName string                 `json:"databaseName" validate:"required"`
	TableName    string                 `json:"tableName" validate:"required"`
	Data         map[string]interface{} `json:"data" validate:"required"`
	Conditions   map[string]interface{} `json:"conditions" validate:"required"`
}

type DeleteRecordReq struct {
	ServerID     uint                   `json:"serverId" validate:"required"`
	DatabaseName string                 `json:"databaseName" validate:"required"`
	TableName    string                 `json:"tableName" validate:"required"`
	Conditions   map[string]interface{} `json:"conditions" validate:"required"`
}
