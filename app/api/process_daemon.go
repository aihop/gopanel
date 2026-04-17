package api

import (
	"fmt"
	"os"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/init/daemon"
	"github.com/aihop/gopanel/init/daemon/supervisord"
	"github.com/gofiber/fiber/v3"
	"github.com/ochinchina/supervisord/types"
)

func StatusSupervisord(c fiber.Ctx) error {
	var reply struct{ StateInfo supervisord.StateInfo }
	err := daemon.Supervisor.GetState(nil, nil, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(reply.StateInfo))
}

func StartSupervisord(c fiber.Ctx) error {
	// 启动所有进程
	var reply struct{ RPCTaskResults []supervisord.RPCTaskResult }
	err := daemon.Supervisor.StartAllProcesses(nil, &struct {
		Wait bool `default:"true"`
	}{Wait: true}, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(reply.RPCTaskResults))
}

func ReloadSupervisord(c fiber.Ctx) error {
	// 重载配置
	_, _, _, err := daemon.Supervisor.Reload(true)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

func StopSupervisord(c fiber.Ctx) error {
	// 停止所有进程
	var reply struct{ RPCTaskResults []supervisord.RPCTaskResult }
	err := daemon.Supervisor.StopAllProcesses(nil, &struct {
		Wait bool `default:"true"`
	}{Wait: true}, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(reply.RPCTaskResults))
}

func DaemonListProcess(c fiber.Ctx) error {
	var reply struct{ AllProcessInfo []types.ProcessInfo }
	err := daemon.Supervisor.GetAllProcessInfo(nil, nil, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 获取配置
	daemonService := service.NewDaemonConfigManager()
	configs, err := daemonService.GetConfig()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 创建配置名称的快速查找映射
	configMap := make(map[string]*service.ProcCfg)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg
	}

	// 创建新的结果结构
	type ProcessInfoWithConfig struct {
		types.ProcessInfo
		Config *service.ProcCfg `json:"config,omitempty"`
	}

	result := make([]ProcessInfoWithConfig, len(reply.AllProcessInfo))

	// 为每个进程信息添加配置
	for i, process := range reply.AllProcessInfo {
		result[i] = ProcessInfoWithConfig{
			ProcessInfo: process,
			Config:      configMap[process.Name],
		}
	}

	return c.JSON(e.Succ(result))
}

func DaemonStartProcess(c fiber.Ctx) error {
	name := c.Params("name")
	var reply struct{ Success bool }
	err := daemon.Supervisor.StartProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(reply.Success))
}

func DaemonStopProcess(c fiber.Ctx) error {
	name := c.Params("name")
	var reply struct{ Success bool }
	err := daemon.Supervisor.StopProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(reply.Success))
}

func DaemonReloadProcess(c fiber.Ctx) error {
	name := c.Params("name")
	// 先 stop 再 start
	var stopReply struct{ Success bool }
	_ = daemon.Supervisor.StopProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &stopReply)
	var startReply struct{ Success bool }
	err := daemon.Supervisor.StartProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &startReply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(startReply.Success))
}

func DaemonGracefulRestart(c fiber.Ctx) error {
	name := c.Params("name")

	// 读取当前 pid
	var beforeInfo struct{ ProcInfo types.ProcessInfo }
	_ = daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: name}, &beforeInfo)
	beforePid := beforeInfo.ProcInfo.Pid

	// 优先尝试信号重载：SIGUSR2 -> SIGHUP -> SIGTERM
	var sigReply struct{ Success bool }
	for _, sig := range []string{"SIGUSR2", "SIGHUP", "SIGTERM"} {
		err := daemon.Supervisor.SignalProcess(nil, &types.ProcessSignal{Name: name, Signal: sig}, &sigReply)
		if err == nil && sigReply.Success {
			// 等待一段时间观察 pid/状态变化，认为重载成功的条件：pid 变化或状态短暂变化后恢复
			success := false
			for i := 0; i < 10; i++ {
				time.Sleep(500 * time.Millisecond)
				var info struct{ ProcInfo types.ProcessInfo }
				if err := daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: name}, &info); err != nil {
					continue
				}
				if info.ProcInfo.Pid != beforePid {
					success = true
					break
				}
				// 若进程短暂退出再启动，也视为成功
				if info.ProcInfo.Statename != beforeInfo.ProcInfo.Statename {
					success = true
					break
				}
			}
			if success {
				return c.JSON(e.Succ(map[string]interface{}{"result": "signalled", "signal": sig}))
			}
			// 否则继续尝试下一个信号或回退
		}
	}

	// 信号未触发平滑重启，回退到 stop -> start（带 Wait=true）
	var stopReply struct{ Success bool }
	if err := daemon.Supervisor.StopProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &stopReply); err != nil {
		return c.JSON(e.Fail(err))
	}

	var startReply struct{ Success bool }
	if err := daemon.Supervisor.StartProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &startReply); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(map[string]interface{}{"result": "restarted", "stopped": stopReply.Success, "started": startReply.Success}))
}

func DaemonProcessLog(c fiber.Ctx) error {
	type LogReq struct {
		Name   string `json:"name"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
	}
	req, err := e.BodyToStruct[LogReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 日志文件假设日志路径可通过进程信息获取
	var procInfo struct{ ProcInfo types.ProcessInfo }
	err = daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: req.Name}, &procInfo)
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("get process info failed: %v", err)))
	}
	logfile := procInfo.ProcInfo.StdoutLogfile
	fileInfo, err := os.Stat(logfile)
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("log file not found: %v", err)))
	}
	size := fileInfo.Size()
	readOffset := int64(req.Offset)
	readLength := int64(req.Length)
	if readLength <= 0 {
		readLength = 10240
	}
	if size > 0 {
		readOffset = size - int64(req.Offset) - readLength
		if readOffset < 0 {
			readOffset = 0
		}
		if readOffset > size {
			readOffset = size
		}
		if remain := size - readOffset; readLength > remain {
			readLength = remain
		}
	}
	var reply struct {
		LogData string
	}

	// 这里可以加offset/length参数
	err = daemon.Supervisor.ReadProcessStdoutLog(nil,
		&supervisord.ProcessLogReadInfo{Name: req.Name, Offset: int(readOffset), Length: int(readLength)}, &reply)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(map[string]interface{}{
		"logData": reply.LogData,
		"logSize": size,
	}))
}

func DaemonProcessLogClean(c fiber.Ctx) error {
	type LogReq struct {
		Name string `json:"name"`
	}
	req, err := e.BodyToStruct[LogReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Name == "" {
		return c.JSON(e.Fail(fmt.Errorf("process name cannot be empty")))
	}
	name := req.Name
	// 获取进程信息以找到日志文件路径
	var procInfo struct{ ProcInfo types.ProcessInfo }
	err = daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: name}, &procInfo)
	if err != nil {
		return c.JSON(e.Fail(fmt.Errorf("get process info failed: %v", err)))
	}
	stdoutLogfile := procInfo.ProcInfo.StdoutLogfile
	stderrLogfile := procInfo.ProcInfo.StderrLogfile

	// 清空日志文件内容
	if err := os.WriteFile(stdoutLogfile, []byte{}, 0644); err != nil {
		return c.JSON(e.Fail(fmt.Errorf("failed to clean stdout log file: %v", err)))
	}
	if err := os.WriteFile(stderrLogfile, []byte{}, 0644); err != nil {
		return c.JSON(e.Fail(fmt.Errorf("failed to clean stderr log file: %v", err)))
	}
	return c.JSON(e.Succ(nil))
}

type Names struct {
	Names []string `json:"names"`
}

// 批量操作
func DaemonStartBatchProcess(c fiber.Ctx) error {
	req, err := e.BodyToStruct[Names](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	results := make(map[string]bool)
	for _, name := range req.Names {
		var reply struct{ Success bool }
		err := daemon.Supervisor.StartProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply)
		results[name] = err == nil && reply.Success
	}
	return c.JSON(e.Succ(results))
}

func DaemonStopBatchProcess(c fiber.Ctx) error {
	req, err := e.BodyToStruct[Names](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	results := make(map[string]bool)
	for _, name := range req.Names {
		var reply struct{ Success bool }
		err := daemon.Supervisor.StopProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply)
		results[name] = err == nil && reply.Success
	}
	return c.JSON(e.Succ(results))
}

func DaemonReloadBatchProcess(c fiber.Ctx) error {
	req, err := e.BodyToStruct[Names](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	results := make(map[string]bool)
	for _, name := range req.Names {
		var stopReply struct{ Success bool }
		_ = daemon.Supervisor.StopProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &stopReply)
		var startReply struct{ Success bool }
		err := daemon.Supervisor.StartProcess(nil, &supervisord.StartProcessArgs{Name: name, Wait: true}, &startReply)
		results[name] = err == nil && startReply.Success
	}
	return c.JSON(e.Succ(results))
}

func DaemonConfigFileLoad(c fiber.Ctx) error {
	// 读取文件路径
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
	return c.JSON(e.Succ(nil))
}

func DaemonConfigList(c fiber.Ctx) error {
	daemonService := service.NewDaemonConfigManager()
	configs, err := daemonService.GetConfig()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(configs))
}

func DaemonConfigAdd(c fiber.Ctx) error {
	req, err := e.BodyToStruct[service.ProcCfg](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	// 写入到配置文件
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
