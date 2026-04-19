//go:build !linux

package monitor

import "runtime"

func allowedMountpoints() func(mount string) bool {
	return func(mount string) bool {
		if mount == "" {
			return false
		}
		if runtime.GOOS == "windows" {
			if len(mount) < 3 {
				return false
			}
			if mount[1] != ':' {
				return false
			}
		}
		return true
	}
}

