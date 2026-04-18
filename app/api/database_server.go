package api

import (
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func DatabaseServerList(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabaseServer().List(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func DatabaseServerCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabaseServer().CountByWhere(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func DatabaseServerCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseServerCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseServer().Create(R, model.DatabaseModeRemote); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseServerUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseServerUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseServer().Update(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseServerDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseServer().Delete(R.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseServerSync(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseServer().Sync(R.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseServerGet(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabaseServer().Get(R.ID)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}
