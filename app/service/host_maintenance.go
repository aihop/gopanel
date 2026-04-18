package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aihop/gopanel/pkg/shell"
)

type HostMaintenanceService struct{}

func NewHostMaintenance() *HostMaintenanceService {
	return &HostMaintenanceService{}
}

func (s *HostMaintenanceService) ClearMemoryCaches(mode int) (stdout string, needPrivilege bool, message string, err error) {
	runtime.GC()
	debug.FreeOSMemory()

	if mode != 1 && mode != 2 && mode != 3 {
		mode = 3
	}

	switch runtime.GOOS {
	case "linux":
		if _, err = os.Stat("/proc/sys/vm/drop_caches"); err != nil {
			return "", false, "已执行 GoPanel 进程内存回收，但系统不支持 drop_caches", nil
		}
		if f, openErr := os.OpenFile("/proc/sys/vm/drop_caches", os.O_WRONLY, 0); openErr != nil {
			return "", true, "权限不足：清理内核缓存需要 root 权限（请用 sudo 或让 systemd 服务以 root 运行）", nil
		} else {
			_ = f.Close()
		}
		stdout, err = shell.ExecfWithTimeout(10*time.Second, fmt.Sprintf("sync; echo %d > /proc/sys/vm/drop_caches", mode))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "permission") {
				return stdout, true, "权限不足：清理内核缓存需要 root 权限（请用 sudo 或让 systemd 服务以 root 运行）", nil
			}
			return stdout, false, "", err
		}
		return stdout, false, "已执行清理内核缓存（drop_caches）", nil
	case "darwin":
		if _, err = exec.LookPath("purge"); err != nil {
			return "", false, "已执行 GoPanel 进程内存回收（macOS 未检测到 purge 命令）", nil
		}
		stdout, err = shell.ExecfWithTimeout(20*time.Second, "purge")
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "permission") || strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
				return stdout, true, "权限不足：执行 purge 可能需要管理员权限，请使用 sudo 运行 GoPanel 或手动执行 purge", nil
			}
			return stdout, false, "", err
		}
		return stdout, false, "已执行 purge（系统缓存回收）", nil
	case "windows":
		return "", false, "已执行 GoPanel 进程内存回收（Windows 系统级缓存清理不在此处自动执行）", nil
	default:
		return "", false, "已执行 GoPanel 进程内存回收（当前系统不支持系统级缓存清理）", nil
	}
}
