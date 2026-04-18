//go:build !windows

package service

import "syscall"

func (s *HostMaintenanceService) RelieveCPU(level int) (int, error) {
	if level < 5 {
		level = 10
	}
	if level > 19 {
		level = 19
	}
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, 0, level); err != nil {
		return level, err
	}
	return level, nil
}

