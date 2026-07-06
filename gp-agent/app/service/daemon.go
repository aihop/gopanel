package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aihop/gopanel/gp-agent/init/daemon"
	"github.com/aihop/gopanel/gp-agent/init/daemon/supervisord"
	"github.com/ochinchina/supervisord/types"
)

func DaemonStatusJSON() (string, error) {
	if daemon.Supervisor == nil {
		return "", errors.New("daemon not initialized")
	}
	var reply struct{ StateInfo supervisord.StateInfo }
	if err := daemon.Supervisor.GetState(nil, &struct{}{}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply.StateInfo)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonStart(ctx context.Context) (string, error) {
	_ = ctx
	if daemon.Supervisor == nil {
		return "", errors.New("daemon not initialized")
	}
	var reply struct{ RPCTaskResults []supervisord.RPCTaskResult }
	if err := daemon.Supervisor.StartAllProcesses(nil, &struct {
		Wait bool `default:"true"`
	}{Wait: true}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply.RPCTaskResults)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonStop(ctx context.Context) (string, error) {
	_ = ctx
	if daemon.Supervisor == nil {
		return "", errors.New("daemon not initialized")
	}
	var reply struct{ RPCTaskResults []supervisord.RPCTaskResult }
	if err := daemon.Supervisor.StopAllProcesses(nil, &struct {
		Wait bool `default:"true"`
	}{Wait: true}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply.RPCTaskResults)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonReload(ctx context.Context) (string, error) {
	_ = ctx
	if daemon.Supervisor == nil {
		return "", errors.New("daemon not initialized")
	}
	added, changed, removed, err := daemon.Supervisor.Reload(true)
	if err != nil {
		return "", err
	}
	out := map[string][]string{"added": added, "changed": changed, "removed": removed}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonAppListJSON() (string, error) {
	if daemon.Supervisor == nil {
		return "", errors.New("daemon not initialized")
	}
	var reply struct{ AllProcessInfo []types.ProcessInfo }
	if err := daemon.Supervisor.GetAllProcessInfo(nil, &struct{}{}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply.AllProcessInfo)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonAppStart(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	name := getString(params, "name")
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	var reply struct{ Success bool }
	if err := daemon.Supervisor.StartProcess((*http.Request)(nil), &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonAppStop(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	name := getString(params, "name")
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	var reply struct{ Success bool }
	if err := daemon.Supervisor.StopProcess((*http.Request)(nil), &supervisord.StartProcessArgs{Name: name, Wait: true}, &reply); err != nil {
		return "", err
	}
	b, err := json.Marshal(reply)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonAppRestart(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	name := getString(params, "name")
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	var stopReply struct{ Success bool }
	_ = daemon.Supervisor.StopProcess((*http.Request)(nil), &supervisord.StartProcessArgs{Name: name, Wait: true}, &stopReply)
	var startReply struct{ Success bool }
	if err := daemon.Supervisor.StartProcess((*http.Request)(nil), &supervisord.StartProcessArgs{Name: name, Wait: true}, &startReply); err != nil {
		return "", err
	}
	out := map[string]bool{"stopped": stopReply.Success, "started": startReply.Success}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DaemonAppLogJSON(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	name := getString(params, "name")
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	offset := getInt(params, "offset", 0)
	length := getInt(params, "length", 10240)
	var reply struct{ LogData string }
	if err := daemon.Supervisor.ReadProcessStdoutLog(nil, &supervisord.ProcessLogReadInfo{Name: name, Offset: offset, Length: length}, &reply); err != nil {
		return "", err
	}
	var logSize int64
	var procInfo struct{ ProcInfo types.ProcessInfo }
	if err := daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: name}, &procInfo); err == nil {
		if fi, err := os.Stat(procInfo.ProcInfo.StdoutLogfile); err == nil {
			logSize = fi.Size()
		}
	}
	b, err := json.Marshal(map[string]interface{}{"logData": reply.LogData, "offset": offset, "length": length, "logSize": logSize, "at": time.Now().UnixMilli()})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

func getInt(m map[string]interface{}, key string, def int) int {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return def
		}
		return n
	default:
		return def
	}
}

// DaemonAppLogClear 清除应用日志
func DaemonAppLogClear(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	name := getString(params, "name")
	if name == "" {
		return "", errors.New("invalid params: name is empty")
	}
	// 获取进程信息以找到日志文件路径
	var procInfo struct{ ProcInfo types.ProcessInfo }
	err := daemon.Supervisor.GetProcessInfo(nil, &struct{ Name string }{Name: name}, &procInfo)
	if err != nil {
		return "", err
	}
	stdoutLogfile := procInfo.ProcInfo.StdoutLogfile
	stderrLogfile := procInfo.ProcInfo.StderrLogfile
	// 清空日志文件内容
	if err := os.WriteFile(stdoutLogfile, []byte{}, 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(stderrLogfile, []byte{}, 0644); err != nil {
		return "", err
	}
	return "", nil
}
