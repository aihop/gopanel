//go:build !windows

package diskscan

import (
	"os"
	"syscall"
)

// deviceOf 返回路径所在文件系统的设备号，用于「不跨文件系统」判断。
// 用 Lstat 而不是 Stat：目标是路径本身，不跟随符号链接。
func deviceOf(path string) (uint64, bool) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true
}
