package api

import (
	"fmt"

	"sync"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/pkg/websocket"
	websocket2 "github.com/aihop/gopanel/utils/websocket"
	"github.com/gofiber/fiber/v3"
)

func ProcessWs(c *websocket.Conn) {
	wsClient := websocket2.NewWsClient("processClient", c)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		wsClient.Read()
	}()
	go func() {
		defer wg.Done()
		wsClient.Write()
	}()
	wg.Wait() // 等待读写goroutine完成
	fmt.Println("WebSocket connection closed")
}

func ListProcess(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ProcessListReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	list, err := processService.List(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(list))
}

// @Tags Process
// @Summary Stop Process
// @Param request body request.ProcessReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /process/stop [post]
// @x-panel-log {"bodyKeys":["PID"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"结束进程 [PID]","formatEN":"结束进程 [PID]"}
func StopProcess(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.ProcessReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := processService.StopProcess(*req); err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err))
	}
	return c.JSON(e.Succ())
}

// 检查进程端口是否被占用
func CheckProcessPort(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.PortReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pid, err := processService.CheckProcessPort(req.Port)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(map[string]interface{}{
		"port": req.Port,
		"pid":  pid,
		"used": pid != 0,
	}))
}
