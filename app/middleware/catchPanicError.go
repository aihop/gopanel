package middleware

import (
	"fmt"
	"runtime"

	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func CatchPanicError(ctx fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			var (
				ok  bool
				err error
				buf = make([]byte, 10240)
				url = ctx.BaseURL() + ctx.Path()
			)
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("%v", r)
			}
			buf = buf[:runtime.Stack(buf, false)]
			global.LOG.Error("Panic Error!", "url", url, "error", err, "buf", string(buf))
			_ = ctx.Status(500).SendString("Internal Server Error")
		}
	}()
	return ctx.Next()
}
