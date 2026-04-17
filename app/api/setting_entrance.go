package api

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

// 更新安全入口
func SettingEntranceUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[SettingEntranceReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	req.Entrance = strings.TrimSpace(req.Entrance)
	if req.Entrance == "" {
		return c.JSON(e.Fail(fmt.Errorf("entrance cannot be empty")))
	}

	// 定义已存在的路由组前缀，防止冲突
	existingGroups := []string{"api", "web"}
	// 检查是否与现有路由组冲突
	for _, group := range existingGroups {
		if req.Entrance == group {
			return c.JSON(e.Fail(fmt.Errorf("entrance '%s' conflicts with an existing route group", req.Entrance)))
		}
	}

	updateConfYamlFile(map[string]interface{}{
		"system.entrance": req.Entrance,
	})

	// 返回当前配置
	return c.JSON(e.Succ())
}
