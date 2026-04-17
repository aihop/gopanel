package api

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/gofiber/fiber/v3"
)

func SSLLogsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.SendString("event: error\ndata: invalid id\n\n")
	}

	logger := service.GetSSLLogger(uint(id))
	ch := logger.Subscribe()

	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer logger.Unsubscribe(ch)

		// 发送历史日志
		for _, logMsg := range logger.GetLogs() {
			fmt.Fprintf(w, "data: %s\n\n", logMsg)
			if err := w.Flush(); err != nil {
				return
			}
		}

		for {
			select {
			case logMsg, ok := <-ch:
				if !ok || logMsg == "EOF" || logMsg == "[\"EOF\"]" || strings.Contains(logMsg, "EOF") {
					fmt.Fprintf(w, "data: EOF\n\n")
					_ = w.Flush()
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", logMsg)
				if err := w.Flush(); err != nil {
					return
				}
			case <-time.After(1 * time.Second): // keep-alive
				fmt.Fprintf(w, "event: ping\ndata: ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

func SSLSearch(c fiber.Ctx) error {
	R, err := e.BodyToContext(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewSSL().List(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLList(c fiber.Ctx) error {
	return SSLSearch(c)
}

func SSLCount(c fiber.Ctx) error {
	R, err := e.BodyToWhere(c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	data, err := service.NewSSL().CountByWhere(&R)
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLPushCDN(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.SSLPushCDN](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err := service.NewSSL().PushCDN(c.Context(), *req); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLCreate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SSLCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	res, err := service.NewSSL().Create(R)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(res))
}

func SSLDelete(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewSSL().Delete(R.ID); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLGet(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	data, err := service.NewSSL().Get(uint(id))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLGetByWebsite(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("websiteId"))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	data, err := service.NewSSL().GetByWebsiteID(uint(id))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(data))
}

func SSLApply(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SSLApply](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewSSL().Apply(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLUpdate(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SSLUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	if err = service.NewSSL().Update(R); err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}

func SSLObtain(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	res, err := service.NewSSL().Obtain(R.ID)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(res))
}

func SSLRenew(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.ID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	err = service.NewSSL().Renew(R.ID)
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ())
}
