package api

import (
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/gofiber/fiber/v3"
)

func DatabaseUserList(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabaseUser().List(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	total, _ := service.NewDatabaseUser().CountByWhere(&gormx.Wherex{Wheres: R.Wheres})
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(dto.PageResult{Total: total, Items: data}))
}

func DatabaseUserCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseUserCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseUser().Create(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseUserUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseUserUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewDatabaseUser().Update(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func DatabaseUserDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseUserDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if R.ID == 0 && R.ServerId == 0 {
		return c.JSON(e.Fail(errors.New("id or serverId is empty")))
	}
	if R.ID > 0 {
		stored, getErr := service.NewDatabaseUser().Get(R.ID)
		if getErr != nil {
			return c.JSON(e.Fail(buserr.Err(getErr)))
		}
		R.ServerId = stored.ServerID
		R.Username = stored.Username
		R.Host = stored.Host
	}
	if R.ServerId > 0 && R.Username != "" {
		server, err := service.NewDatabaseServer().Get(R.ServerId)
		if err != nil {
			return c.JSON(e.Fail(buserr.Err(err)))
		}
		switch server.Type {
		case model.DatabaseTypeMysql:
			if R.Host == "" {
				return c.JSON(e.Fail(buserr.New(constant.ErrDatabaseUserHostRequired)))
			}
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
			defer func() { _ = mysql.Close() }()
			if err = mysql.UserDrop(R.Username, R.Host); err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
		case model.DatabaseTypePostgresql:
			postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
			if err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
			defer func() { _ = postgres.Close() }()
			if err = postgres.UserDrop(R.Username); err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
		default:
			return c.JSON(e.Fail(errors.New("unsupported database server type")))
		}
	}
	if R.ID > 0 {
		if err = service.NewDatabaseUser().Delete(R.ID); err != nil {
			return c.JSON(e.Fail(buserr.Err(err)))
		}
	}
	return c.JSON(e.Succ())
}

func DatabaseUserPassword(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseUserGet](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	password, managed, err := service.NewDatabaseUser().GetStoredPassword(R)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(e.Succ(fiber.Map{"password": password, "managed": managed}))
}

func DatabaseUserGet(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.DatabaseUserGet](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	var data *model.DatabaseUser
	if R.ID > 0 {
		data, err = service.NewDatabaseUser().Get(R.ID)
	} else {
		data, err = service.NewDatabaseUser().GetByIdentity(R.ServerID, R.Username, R.Host)
	}
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}
