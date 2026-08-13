package api

import (
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func FlowPage(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	total, items, err := service.NewFlowApplication(global.DB).Page(
		claims.UserId, claims.Role == constant.UserRoleSuper, page, limit,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": total}))
}

func FlowCreate(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	input, err := e.BodyToStruct[service.FlowCreateInput](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	item, err := service.NewFlowApplication(global.DB).Create(
		*input, claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func FlowUpdate(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	input, err := e.BodyToStruct[service.FlowUpdateInput](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	item, err := service.NewFlowApplication(global.DB).Update(
		uint(id), *input, claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func FlowDelete(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	if err := service.NewFlowApplication(global.DB).Delete(
		uint(id), claims.UserId, claims.Role == constant.UserRoleSuper,
	); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func FlowRunPage(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	flowID, _ := strconv.Atoi(c.Query("flowId", "0"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	total, items, err := service.NewFlowRunApplication(global.DB).Page(
		uint(flowID), claims.UserId, claims.Role == constant.UserRoleSuper, page, limit,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": total}))
}

func FlowRunGet(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	item, err := service.NewFlowRunApplication(global.DB).Get(
		uint(id), claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func FlowRunResume(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	item, err := service.NewFlowRunApplication(global.DB).Resume(
		uint(id), claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func FlowCodeDeliverySources(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	flowID, parseErr := strconv.Atoi(c.Params("id"))
	if parseErr != nil || flowID <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	items, err := service.NewFlowRunApplication(global.DB).CodeDeliverySources(
		uint(flowID), claims.UserId, claims.Role == constant.UserRoleSuper, 30,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(items))
}

func FlowCodeBaselineSource(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	flowID, parseErr := strconv.Atoi(c.Params("id"))
	if parseErr != nil || flowID <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrParameterError)))
	}
	item, err := service.NewFlowRunApplication(global.DB).CodeBaselineSource(
		uint(flowID), claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func FlowRunCreate(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	input, err := e.BodyToStruct[service.FlowRunCreateInput](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	item, err := service.NewFlowRunApplication(global.DB).Create(
		*input, claims.UserId, claims.Role == constant.UserRoleSuper,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}
