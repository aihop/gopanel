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

// ownerUID 返回文件属主 uid。面板以普通用户运行时，属主不是自己的文件删不掉，
// 得在结果里提前标出来，而不是等用户点了删除才报错。
func ownerUID(path string) (int, bool) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}
