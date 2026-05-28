package api

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
)

// 获取数据库的表列表
func GetDBManagerTables(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.GetTablesReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	tables, err := service.NewDBManagerService().GetTables(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(tables))
}

// 获取数据库表列表（带分页）
func GetDBManagerTableList(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.GetTableListReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	data, err := service.NewDBManagerService().GetTableList(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// 获取表的数据
func GetDBManagerTableData(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.GetTableDataReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	data, err := service.NewDBManagerService().GetTableData(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// 执行任意 SQL
func ExecDBManagerSql(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ExecSqlReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := service.NewDBManagerService().ExecSql(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

// @Tags DBManager
// @Summary Insert database manager record
// @Accept json
// @Param request body request.InsertRecordReq true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/manager/insert [post]
// @x-panel-log {"bodyKeys":["databaseName", "tableName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"在数据库 [databaseName] 表 [tableName] 中插入记录","formatEN":"Insert record in database [databaseName] table [tableName]"}
func InsertDBManagerRecord(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.InsertRecordReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().InsertRecord(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("插入成功"))
}

// @Tags DBManager
// @Summary Update database manager record
// @Accept json
// @Param request body request.UpdateRecordReq true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/manager/update [post]
// @x-panel-log {"bodyKeys":["databaseName", "tableName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新数据库 [databaseName] 表 [tableName] 中的记录","formatEN":"Update record in database [databaseName] table [tableName]"}
func UpdateDBManagerRecord(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.UpdateRecordReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().UpdateRecord(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("更新成功"))
}

// @Tags DBManager
// @Summary Delete database manager record
// @Accept json
// @Param request body request.DeleteRecordReq true "request"
// @Success 200 {object} e.Response
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /database/manager/delete [post]
// @x-panel-log {"bodyKeys":["databaseName", "tableName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除数据库 [databaseName] 表 [tableName] 中的记录","formatEN":"Delete record in database [databaseName] table [tableName]"}
func DeleteDBManagerRecord(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.DeleteRecordReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().DeleteRecord(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("删除成功"))
}

// ExportDBManagerTable exports table data as CSV or SQL dump
// UploadImportDBManagerTable imports data from an uploaded CSV/SQL file (multipart)
func UploadImportDBManagerTable(c fiber.Ctx) error {
	serverID, err := strconv.Atoi(c.FormValue("serverId"))
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("invalid serverId")))
	}
	databaseName := c.FormValue("databaseName")
	tableName := c.FormValue("tableName")
	format := c.FormValue("format")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("file is required: %v", err)))
	}

	// Auto-detect format from file extension if not specified
	if format == "" {
		if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".sql") {
			format = "sql"
		} else {
			format = "csv"
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("failed to open uploaded file: %v", err)))
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("failed to read uploaded file: %v", err)))
	}

	imported, err := service.NewDBManagerService().ImportTable(request.ImportTableReq{
		ServerID:     uint(serverID),
		DatabaseName: databaseName,
		TableName:    tableName,
		Format:       format,
		Content:      string(contentBytes),
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"imported": imported}))
}

// ImportDBManagerTable imports data from CSV content or SQL dump into a table
func ImportDBManagerTable(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ImportTableReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	imported, err := service.NewDBManagerService().ImportTable(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"imported": imported}))
}

func ExportDBManagerTable(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ExportTableReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	content, filename, err := service.NewDBManagerService().ExportTable(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.SendString(content)
}

// CreateDBManagerDatabase 创建数据库
// @Router /database/manager/create-database [post]
func CreateDBManagerDatabase(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CreateDatabaseReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().CreateDatabase(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("创建成功"))
}

// DropDBManagerDatabase 删除数据库
// @Router /database/manager/drop-database [post]
func DropDBManagerDatabase(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.DropDatabaseReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().DropDatabase(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("删除成功"))
}

// GetDBManagerTableInfo 获取表结构信息（SHOW CREATE TABLE）
// @Router /database/manager/table-info [post]
func GetDBManagerTableInfo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.GetTableInfoReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	data, err := service.NewDBManagerService().GetTableInfo(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// GetDBManagerDatabaseInfo 获取数据库统计信息
// @Router /database/manager/database-info [post]
func GetDBManagerDatabaseInfo(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.GetDatabaseInfoReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	data, err := service.NewDBManagerService().GetDatabaseInfo(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(data))
}

// CopyDBManagerTable 复制表
// @Router /database/manager/copy-table [post]
func CopyDBManagerTable(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CopyTableReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().CopyTable(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("复制成功"))
}

// CreateDBManagerTable 创建表
// @Router /database/manager/create-table [post]
func CreateDBManagerTable(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CreateTableReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDBManagerService().CreateTable(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ("创建成功"))
}
