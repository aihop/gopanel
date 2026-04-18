package api

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func UserInfo(c fiber.Ctx) error {
	r, err := e.BodyToStruct[dto.UserInfo](c.Body())
	if err != nil {
		// 初始化 r 避免 nil 指针问题
		r = &dto.UserInfo{}
	}
	// 从 JWT 上下文获取 UserId（若请求体未传 id 则使用）
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if ok && r.ID == 0 {
		r.ID = info.UserId
	}

	user, err := service.NewUser().Get(r.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(user))
}

func CreateUser(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.UserCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewUser().Create(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func UpdateUser(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.UserUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewUser().UpdateUser(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func DeleteUser(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CommonID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := service.NewUser().Delete(req.ID); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PageUser(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.UserSearch](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	total, items, err := service.NewUser().Page(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(map[string]interface{}{
		"items": items,
		"total": total,
	}))
}
func ResetAccount(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.AuthSignin](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 从 JWT 上下文获取 UserId
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok {
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "Unauthorized"))
	}
	err = service.NewUser().ResetAccount(info.UserId, req.Email, req.Password)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}
