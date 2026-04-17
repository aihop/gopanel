package api

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/viper"
)

type SettingPortReq struct {
	ServerPort uint32 `json:"serverPort" validate:"required"`
}

type SettingEntranceReq struct {
	Entrance string `json:"entrance" validate:"required"`
}

type SettingClearDirReq struct {
	Key string `json:"key" validate:"required" enum:"tmp,log,cache"`
	Dir string `json:"dir"`
}

func updateConfYamlFile(keys map[string]interface{}) error {
	workDir, _ := os.Getwd()
	configFile := path.Join(workDir, "config", "conf.yaml")
	// configFile := path.Join(global.CONF.System.BaseDir, "config", "config.yaml")

	// 使用 viper 更新配置
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	for key, value := range keys {
		v.Set(key, value)
	}

	// 写回配置文件
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

type Result struct {
	Count     int   `json:"count"`      // 删除的文件数
	TotalSize int64 `json:"total_size"` // 删除文件的总字节
}

// 递归删除文件（不删目录），并统计
func clearFilesWithStats(root string) (Result, error) {
	var res Result
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || d.IsDir() {
			return nil // 跳过根目录和目录
		}

		// 获取文件信息
		info, err := d.Info()
		if err != nil {
			return err
		}

		// 删除文件
		if err := os.Remove(path); err != nil {
			return err
		}

		// 累加统计
		res.Count++
		res.TotalSize += info.Size()
		return nil
	})
	return res, err
}

// @Tags System Setting
// @Summary Update system setting
// @Accept json
// @Param request body dto.SettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /settings/update [post]
// @x-panel-log {"bodyKeys":["key","value"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"修改系统配置 [key] => [value]","formatEN":"update system setting [key] => [value]"}
func SettingUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.SettingUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	if req.Key == "SecurityEntrance" {
		if !checkEntrancePattern(req.Value) {
			return c.JSON(e.Fail(fmt.Errorf("the format of the security entrance %s is incorrect.", req.Value)))
		}
	}
	settingService := service.NewSetting()

	if err := settingService.Update(req.Key, req.Value); err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Key == "SecurityEntrance" {
		entranceValue := base64.StdEncoding.EncodeToString([]byte(req.Value))
		c.Cookie(&fiber.Cookie{
			Name:  "SecurityEntrance",
			Value: entranceValue,
		})
	}
	return c.JSON(e.Succ())
}

func checkEntrancePattern(val string) bool {
	if len(val) == 0 {
		return true
	}
	result, _ := regexp.MatchString("^[a-zA-Z0-9]{5,116}$", val)
	return result
}
