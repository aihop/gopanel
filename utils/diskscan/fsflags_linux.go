//go:build linux

package diskscan

import "syscall"

// stRdonly = ST_RDONLY，只读挂载（squashfs、只读 bind mount 等）
const stRdonly = 0x0001

func isReadOnlyFS(path string) bool {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return false
	}
	return st.Flags&stRdonly != 0
}
