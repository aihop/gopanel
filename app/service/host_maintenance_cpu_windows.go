//go:build windows

package service

import (
	"golang.org/x/sys/windows"
)

func (s *HostMaintenanceService) RelieveCPU(level int) (int, error) {
	h := windows.CurrentProcess()
	if err := windows.SetPriorityClass(h, windows.BELOW_NORMAL_PRIORITY_CLASS); err != nil {
		return level, err
	}
	return level, nil
}

