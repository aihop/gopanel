package api

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/gofiber/fiber/v3"
)

type ProcessInfo struct {
	Name          string `xml:"name" json:"name"`
	Group         string `xml:"group" json:"group"`
	Description   string `xml:"description" json:"description"`
	Start         int    `xml:"start" json:"start"`
	Stop          int    `xml:"stop" json:"stop"`
	Now           int    `xml:"now" json:"now"`
	State         int    `xml:"state" json:"state"`
	Statename     string `xml:"statename" json:"statename"`
	Spawnerr      string `xml:"spawnerr" json:"spawnerr"`
	Exitstatus    int    `xml:"exitstatus" json:"exitstatus"`
	Logfile       string `xml:"logfile" json:"logfile"`
	StdoutLogfile string `xml:"stdout_logfile" json:"stdout_logfile"`
	StderrLogfile string `xml:"stderr_logfile" json:"stderr_logfile"`
	Pid           int    `xml:"pid" json:"pid"`
}

func DaemonStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_STATUS", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(out))
}

func DaemonStart(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_START", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonReload(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_RELOAD", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonStop(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_STOP", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonListProcess(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_LIST", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var replys []ProcessInfo
	if err := json.Unmarshal([]byte(resp.Output), &replys); err != nil {
		return c.JSON(e.Fail(err))
	}

	// 获取配置
	daemonService := service.NewDaemonConfigManager()
	configs, err := daemonService.GetConfig()

	// 创建配置名称的快速查找映射
	configMap := make(map[string]*service.ProcCfg)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg
	}

	// 创建新的结果结构
	type ProcessInfoWithConfig struct {
		ProcessInfo
		Config *service.ProcCfg `json:"config,omitempty"`
	}

	result := make([]ProcessInfoWithConfig, len(replys))

	// 为每个进程信息添加配置
	for i, process := range replys {
		result[i] = ProcessInfoWithConfig{
			ProcessInfo: process,
			Config:      configMap[process.Name],
		}
	}

	return c.JSON(e.Succ(result))
}

func DaemonStartProcess(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_START", map[string]interface{}{"name": name})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonStopProcess(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_STOP", map[string]interface{}{"name": name})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonReloadProcess(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_RESTART", map[string]interface{}{"name": name, "mode": "reload"})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonGracefulRestart(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_RESTART", map[string]interface{}{"name": name, "mode": "graceful"})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ(out))
}

func DaemonProcessLog(c fiber.Ctx) error {
	type LogReq struct {
		Name   string `json:"name"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
		Tail   int    `json:"tail"`
	}
	req, err := e.BodyToStruct[LogReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	length := req.Length
	if length == 0 && req.Tail > 0 {
		length = req.Tail
	}
	if length == 0 {
		length = 10240
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_LOG", map[string]interface{}{
		"name":   req.Name,
		"offset": req.Offset,
		"length": length,
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(out))
}

func DaemonProcessLogClear(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.JSON(e.Fail(errors.New("process name cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "DAEMON_APP_LOG_CLEAR", map[string]interface{}{"name": name})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if resp.Output != "" {
		_ = json.Unmarshal([]byte(resp.Output), &out)
	}
	return c.JSON(e.Succ())
}

type Names struct {
	Names []string `json:"names"`
}

func DaemonConfigFileLoad(c fiber.Ctx) error {
	file_path := service.NewDaemonConfigManager().FilePath
	if _, err := os.Stat(file_path); err != nil {
		return c.JSON(e.Fail(err))
	}
	content, err := os.ReadFile(file_path)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(string(content)))
}

func DaemonConfigFileSave(c fiber.Ctx) error {
	type Content struct {
		Content string `json:"content"`
	}
	req, err := e.BodyToStruct[Content](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	file_path := service.NewDaemonConfigManager().FilePath
	// 直接写入到文件中
	if err := os.WriteFile(file_path, []byte(req.Content), 0644); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func DaemonConfigAdd(c fiber.Ctx) error {
	req, err := e.BodyToStruct[service.ProcCfg](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = service.NewDaemonConfigManager().AddConfig(req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func DaemonConfigUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[service.ProcCfg](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 写入到配置文件
	err = service.NewDaemonConfigManager().UpdateConfig(req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func DaemonConfigDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[Names](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	manage := service.NewDaemonConfigManager()
	for _, name := range req.Names {
		err := manage.DeleteConfig(name)
		if err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	return c.JSON(e.Succ())
}
