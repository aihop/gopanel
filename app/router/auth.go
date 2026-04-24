package router

import (
	"github.com/aihop/gopanel/app/api"
	"github.com/gofiber/fiber/v3"
)

func AuthRouter(r fiber.Router) {
	r.Post("/auth/signin", api.Login)
	r.Post("/auth/login", api.Login)
	r.Post("/auth/verify/captcha/get", api.VerifyCaptchaGet)
	r.Post("/auth/verify/captcha/check", api.VerifyCaptchaCheck)
}
