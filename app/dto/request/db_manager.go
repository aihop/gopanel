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
	SortField      string            `json:"sortField"`
	SortOrder      string            `json:"sortOrder"`
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
	ServerID          uint     `json:"serverId" validate:"required"`
	DatabaseName      string   `json:"databaseName" validate:"required"`
	TableName         string   `json:"tableName" validate:"required"`
	Format            string   `json:"format" validate:"required"` // csv or sql
	Columns           []string `json:"columns"`                    // 空 = 全部字段
	Where             string   `json:"where"`                      // 可选 WHERE 条件
	IncludeDropTable  bool     `json:"includeDropTable"`           // SQL 格式：包含 DROP TABLE
	IncludeCreateTable bool    `json:"includeCreateTable"`         // SQL 格式：包含 CREATE TABLE
}

type ImportTableReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	TableName    string `json:"tableName" validate:"required"`
	Format       string `json:"format" validate:"required"` // csv or sql
	Content      string `json:"content" validate:"required"`
}

// CreateDatabaseReq 创建数据库
type CreateDatabaseReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	Charset      string `json:"charset"`
	Collation    string `json:"collation"`
}

// DropDatabaseReq 删除数据库
type DropDatabaseReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
}

// GetTableInfoReq 获取表结构信息（SHOW CREATE TABLE / 建表语句）
type GetTableInfoReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	TableName    string `json:"tableName" validate:"required"`
}

// GetDatabaseInfoReq 获取数据库级信息
type GetDatabaseInfoReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
}

// ColumnDef 建表字段定义
type ColumnDef struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Length        string `json:"length"`
	Nullable      bool   `json:"nullable"`
	DefaultValue  string `json:"defaultValue"`
	AutoIncrement bool   `json:"autoIncrement"`
	Comment       string `json:"comment"`
	IsPrimary     bool   `json:"isPrimary"`
}

// CopyTableReq 复制表
type CopyTableReq struct {
	ServerID       uint   `json:"serverId" validate:"required"`
	DatabaseName   string `json:"databaseName" validate:"required"`
	SourceTable    string `json:"sourceTable" validate:"required"`
	TargetTable    string `json:"targetTable" validate:"required"`
	CopyData       bool   `json:"copyData"`
}

// ChangeTableOwnerReq 修改表的所有者（仅 PostgreSQL 支持真正的表级 owner）
type ChangeTableOwnerReq struct {
	ServerID     uint   `json:"serverId" validate:"required"`
	DatabaseName string `json:"databaseName" validate:"required"`
	TableName    string `json:"tableName" validate:"required"`
	Owner        string `json:"owner" validate:"required"`
}

// CreateTableReq 创建表
type CreateTableReq struct {
	ServerID     uint        `json:"serverId" validate:"required"`
	DatabaseName string      `json:"databaseName" validate:"required"`
	TableName    string      `json:"tableName" validate:"required"`
	Engine       string      `json:"engine"`
	Charset      string      `json:"charset"`
	Collation    string      `json:"collation"`
	Comment      string      `json:"comment"`
	Columns      []ColumnDef `json:"columns" validate:"required,min=1"`
}
