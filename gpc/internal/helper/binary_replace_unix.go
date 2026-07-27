//go:build !windows

package helper

import (
	"io"
	"os"
	"path/filepath"
)

// replaceBinary 原子替换一个可能正在运行的可执行文件。
//
// 不能像普通文件那样直接 O_TRUNC 写目标：内核对正在执行的二进制加了写保护，
// 打开就返回 ETXTBSY（text file busy）。gp-agent 自更新失败正是这个原因：
//
//	gpc install error: open /opt/gopanel/gp-agent: text file busy
//
// 正确做法是「同目录写临时文件 + rename」：rename 只替换目录项且是原子的，
// 正在运行的老进程继续持有旧 inode 不受影响，新进程启动时拿到新文件。
// 临时文件必须落在目标同一目录——跨文件系统 rename 会直接失败。
//
// 注意：替换完成后仍需重启服务才会用上新二进制，rename 本身不会影响已运行的进程。
func replaceBinary(src, dst string, mode os.FileMode) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(dir, ".gp-bin-new-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := io.Copy(tmp, in); err != nil {
		cleanup()
		return err
	}
	// 先落盘再改名：进程若在这中间被杀，不会留下一个内容不全的可执行文件
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	// chmod 要在 rename 之前做，否则会出现「文件已就位但还不可执行」的窗口
	if err := os.Chmod(tmpPath, mode); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
