//go:build darwin

package diskscan

import "syscall"

// mntRdonly = MNT_RDONLY。macOS 的 / 是封印过的只读系统卷，
// 上面的文件任何权限都删不掉，必须在结果里标出来。
const mntRdonly = 0x00000001

func isReadOnlyFS(path string) bool {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return false
	}
	return st.Flags&mntRdonly != 0
}
