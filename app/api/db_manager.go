package api

import (
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
