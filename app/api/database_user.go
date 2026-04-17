package api

import (
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/init/db"
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
	return c.JSON(e.Succ(data))
}

func DatabaseUserCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewDatabaseUser().CountByWhere(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
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
		if err = service.NewDatabaseUser().Delete(R.ID); err != nil {
			return c.JSON(e.Fail(buserr.Err(err)))
		}
	}
	if R.ServerId > 0 && R.Username != "" {
		server, err := service.NewDatabaseServer().Get(R.ServerId)
		if err != nil {
			return c.JSON(e.Fail(buserr.Err(err)))
		}
		switch server.Type {
		case model.DatabaseTypeMysql:
			mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
			defer func() { _ = mysql.Close() }()
			users, err := mysql.Users()
			if err != nil {
				return c.JSON(e.Fail(buserr.Err(err)))
			}
			for _, user := range users {
				if user.User == R.Username {
					if err = mysql.UserDrop(user.User, user.Host); err != nil {
						return c.JSON(e.Fail(buserr.Err(err)))
					}
				}
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
	return c.JSON(e.Succ())
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
