package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/gofiber/fiber/v3"
)

// 获取计划任务列表
func CronjobList(c fiber.Ctx) error {
	ctx, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	svc := service.NewCronjobService()
	data, err := svc.List(&ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	total, err := svc.CountByWhere(&gormx.Wherex{Wheres: ctx.Wheres})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.PageResult{Total: total, Items: data}))
}

// 创建计划任务
func CronjobCreate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CronjobCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	job, err := service.NewCronjobService().Create(req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(job))
}

// 更新计划任务
func CronjobUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CronjobUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewCronjobService().Update(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// 获取单个计划任务
func CronjobGet(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	job, err := service.NewCronjobService().Get(req.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(job))
}

// 删除计划任务
func CronjobDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewCronjobService().Delete(req.ID); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// 启用/禁用计划任务
func CronjobSetStatus(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CronjobSetStatus](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewCronjobService().SetStatus(req.ID, req.Status); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

// 立即执行一次
func CronjobRun(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	go service.NewCronjobService().Run(req.ID)
	return c.JSON(e.Succ())
}

// 查询执行记录
func CronjobRecordList(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CronjobRecordList](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	records, err := service.NewCronjobService().RecordList(req.CronjobID, req.Limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(records))
}

// 清空执行记录
func CronjobRecordDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CronjobRecordDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewCronjobService().RecordDelete(req.CronjobID); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
