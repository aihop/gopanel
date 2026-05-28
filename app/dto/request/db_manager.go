package request

type GetTablesReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
}

type GetTableListReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	Page         int    `json:"page" validate:"required,min=1"`
	Limit        int    `json:"limit" validate:"required,min=1,max=100"`
	Keyword      string `json:"keyword"`
	SortField    string `json:"sortField"`
	SortOrder    string `json:"sortOrder"`
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
	Limit          int               `json:"limit" validate:"required,min=1,max=100"`
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

type ExportTableReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	TableName    string `json:"tableName" validate:"required"`
	Format       string `json:"format" validate:"required"` // csv or sql
}

type ImportTableReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	TableName    string `json:"tableName" validate:"required"`
	Format       string `json:"format" validate:"required"` // csv or sql
	Content      string `json:"content" validate:"required"`
}
