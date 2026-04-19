//go:build linux

package monitor

import "strings"

func allowedMountpoints() func(mount string) bool {
	return func(mount string) bool {
		if mount == "" {
			return false
		}
		if strings.HasPrefix(mount, "/proc") ||
			strings.HasPrefix(mount, "/sys") ||
			strings.HasPrefix(mount, "/dev") ||
			strings.HasPrefix(mount, "/run") {
			return false
		}
		return true
	}
}

