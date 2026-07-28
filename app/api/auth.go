package api

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/geo"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func Login(c fiber.Ctx) error {
	r, err := e.BodyToStruct[dto.AuthSignin](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)

	logService := service.NewLogService()
	loginLog := model.LoginLog{
		IP:      c.IP(),
		Agent:   string(c.Request().Header.UserAgent()),
		Address: geo.Region(c.IP()),
	}
	if message, blocked := defaultLoginAttemptGuard.Check(c.IP(), r); blocked {
		loginLog.Status = constant.StatusFailed
		loginLog.Message = "login blocked by rate limiter"
		_ = logService.CreateLoginLog(loginLog)
		return c.JSON(e.RetError(constant.StatusCodeFullFail, message))
	}
	if defaultLoginAttemptGuard.RequiresCaptcha(c.IP(), r) {
		if err := consumeVerifiedLoginCaptchaToken(r.CaptchaToken); err != nil {
			loginLog.Status = constant.StatusFailed
			loginLog.Message = "captcha verification required"
			_ = logService.CreateLoginLog(loginLog)
			return c.JSON(e.RetError(constant.StatusCodeFullFail, "请先完成滑块验证"))
		}
	}

	failLogin := func(logMessage string) error {
		loginLog.Status = constant.StatusFailed
		loginLog.Message = logMessage
		defaultLoginAttemptGuard.RegisterFailure(c.IP(), r)
		time.Sleep(loginAttemptDelay)
		_ = logService.CreateLoginLog(loginLog)
		return c.JSON(e.Fail(buserr.WithDetail("ErrLoginFailed", logMessage)))
	}

	userService := service.NewUser()
	var user *model.User
	if r.Email != "" {
		user, err = userService.GetByEmail(r.Email)
	} else if r.Mobile != "" {
		user, err = userService.GetByMobile(r.Mobile)
	}
	if err != nil || user == nil {
		return failLogin("user not found")
	}
	if user.Status != constant.UserStatusNormal {
		return failLogin("user is disabled")
	}
	if !cryptx.ValidatePassword(user.Password, r.Password) {
		return failLogin("Password verification failed")
	}
	jwt, err := token.Create(user.ID, user.Role, user.Salt, user.FileBaseDir, time.Duration(constant.AuthAdminLoginExpires))
	if err != nil {
		loginLog.Status = constant.StatusFailed
		loginLog.Message = err.Error()
		_ = logService.CreateLoginLog(loginLog)
		return c.JSON(e.Fail(err))
	}

	loginLog.Status = constant.StatusSuccess
	loginLog.Message = "Login success"
	_ = logService.CreateLoginLog(loginLog)
	defaultLoginAttemptGuard.RegisterSuccess(c.IP(), r)

	user.LoginAt = time.Now()
	userService.Update(&model.User{ID: user.ID, LoginAt: time.Now()})
	data := map[string]interface{}{
		"xAuth":     jwt,
		"userInfo":  user,
		"expiresIn": time.Now().Unix(),
	}

	entrance := base64.StdEncoding.EncodeToString([]byte(global.CONF.System.Entrance))
	c.Cookie(&fiber.Cookie{
		Name:     constant.Entrance,
		Value:    entrance,
		Expires:  time.Now().Add(1 * 24 * time.Hour),
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   c.Scheme() == "https",
	})

	sID := c.Cookies(constant.SessionName)
	sessionUser, err := global.SESSION.Get(sID)
	if err != nil {
		sID = uuid.New().String()
		// 设置 cookie，有效期1天
		c.Cookie(&fiber.Cookie{
			Name:     constant.SessionName,
			Value:    sID,
			Expires:  time.Now().Add(1 * 24 * time.Hour),
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   c.Scheme() == "https",
		})
		err := global.SESSION.Set(sID, sessionUser, 86400)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
		return c.JSON(e.Succ(data))
	}
	if err := global.SESSION.Set(sID, sessionUser, 86400); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(data))
}

func ResetPassword(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.AuthResetPasswordReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 从 JWT 上下文获取 UserId
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok {
		return c.JSON(e.Fail(err))
	}
	// 获取用户model
	userService := service.NewUser()
	user, err := userService.Get(info.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 验证旧密码
	if !cryptx.ValidatePassword(user.Password, req.Password) {
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "Old password verification failed"))
	}
	// 加密新密码
	encodedPassword, err := cryptx.EncodePassword(req.NewPassword)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	user.Password = encodedPassword
	err = userService.Update(user)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func UserToken(c fiber.Ctx) error {
	_, err := e.BodyToStruct[dto.UserToken](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok {
		return c.JSON(e.Fail(err))
	}
	user, err := service.NewUser().Get(info.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(user.Token))
}

func UserEditInfo(c fiber.Ctx) error {
	R, err := e.BodyToStruct[dto.UserEditInfo](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	info, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok {
		return c.JSON(e.Fail(err))
	}
	userService := service.NewUser()
	user, err := userService.Get(info.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	userService.Update(&model.User{ID: user.ID, Email: R.Email, NickName: R.NickName})
	return c.JSON(e.Succ())
}
