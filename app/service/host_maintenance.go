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

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/i18n"
	"github.com/aihop/gopanel/pkg/shell"
	"github.com/gofiber/fiber/v3"
)

type HostMaintenanceService struct{}

func NewHostMaintenance() *HostMaintenanceService {
	return &HostMaintenanceService{}
}

func (s *HostMaintenanceService) ClearMemoryCaches(c fiber.Ctx, mode int) (stdout string, needPrivilege bool, message string, err error) {
	runtime.GC()
	debug.FreeOSMemory()

	if mode != 1 && mode != 2 && mode != 3 {
		mode = 3
	}

	switch runtime.GOOS {
	case "linux":
		if _, err = os.Stat("/proc/sys/vm/drop_caches"); err != nil {
			return "", false, i18n.GetMsgFromCtx(c, constant.ErrHostMemGoPanelRecycledLinuxNoDrop), nil
		}
		if f, openErr := os.OpenFile("/proc/sys/vm/drop_caches", os.O_WRONLY, 0); openErr != nil {
			return "", true, i18n.GetMsgFromCtx(c, constant.ErrHostMemRootRequiredLinux), nil
		} else {
			_ = f.Close()
		}
		stdout, err = shell.ExecfWithTimeout(10*time.Second, fmt.Sprintf("sync; echo %d > /proc/sys/vm/drop_caches", mode))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "permission") {
				return stdout, true, i18n.GetMsgFromCtx(c, constant.ErrHostMemRootRequiredLinux), nil
			}
			return stdout, false, "", err
		}
		return stdout, false, i18n.GetMsgFromCtx(c, constant.ErrHostMemCacheCleanedLinux), nil
	case "darwin":
		if _, err = exec.LookPath("purge"); err != nil {
			return "", false, i18n.GetMsgFromCtx(c, constant.ErrHostMemGoPanelRecycledDarwinNoPurge), nil
		}
		stdout, err = shell.ExecfWithTimeout(20*time.Second, "purge")
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "permission") || strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
				return stdout, true, i18n.GetMsgFromCtx(c, constant.ErrHostMemRootRequiredDarwin), nil
			}
			return stdout, false, "", err
		}
		return stdout, false, i18n.GetMsgFromCtx(c, constant.ErrHostMemCacheCleanedDarwin), nil
	case "windows":
		return "", false, i18n.GetMsgFromCtx(c, constant.ErrHostMemGoPanelRecycledWindows), nil
	default:
		return "", false, i18n.GetMsgFromCtx(c, constant.ErrHostMemGoPanelRecycledUnsupported), nil
	}
}
