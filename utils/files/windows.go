//go:build windows
// +build windows

package files

import "os"

// Windows 下返回默认值，UID/GID 在 Windows 上通常无意义
func GetUidGid(fi os.FileInfo) (int, int) {
	// 若需要可从 syscall 或 x/sys/windows 获取更多信息
	_ = fi
	return -1, -1
}
