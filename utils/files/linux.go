//go:build !windows
// +build !windows

package files

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// 从 os.FileInfo 安全提取 uid/gid（Unix 实现）
func GetUidGid(fi os.FileInfo) (int, int) {
	if fi == nil {
		return -1, -1
	}
	// 优先尝试 x/sys/unix 的 Stat_t
	if st, ok := fi.Sys().(*unix.Stat_t); ok && st != nil {
		return int(st.Uid), int(st.Gid)
	}
	// 兼容性：有时会返回 syscall.Stat_t（例如某些 macOS/go 组合）
	if st2, ok := fi.Sys().(*syscall.Stat_t); ok && st2 != nil {
		return int(st2.Uid), int(st2.Gid)
	}
	return -1, -1
}
