//go:build !windows

package helper

import (
	"os"
	"path/filepath"
	"strings"
)

// 这份保护名单与面板侧 utils/diskscan/protect.go 等价，两边都要有：
// 面板以 root 运行时直接本地删（面板侧那份生效），rootless 时走 gpc（这份生效）。
// gpc 这份是最后一道关卡——即使有人绕过面板直接连 socket，也必须被它挡住。
// 改动其中一边时必须同步另一边。
var diskProtectedRoots = []string{
	"/", "/bin", "/sbin", "/lib", "/lib32", "/lib64", "/libx32",
	"/usr", "/etc", "/boot", "/root", "/home", "/var", "/opt", "/srv",
	"/proc", "/sys", "/dev", "/run",
}

var diskProtectedPrefixes = []string{
	"/bin/", "/sbin/", "/lib/", "/lib32/", "/lib64/", "/libx32/",
	"/usr/bin/", "/usr/sbin/", "/usr/lib/", "/usr/lib64/", "/usr/libexec/",
	"/etc/", "/boot/", "/proc/", "/sys/", "/dev/", "/run/",
}

// isDiskProtectedPath 受保护路径一律拒绝，且不受任何 scanId 授权豁免。
func (s *Server) isDiskProtectedPath(path string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." || !filepath.IsAbs(clean) {
		return true
	}
	for _, r := range diskProtectedRoots {
		if clean == r {
			return true
		}
	}
	for _, p := range diskProtectedPrefixes {
		if strings.HasPrefix(clean, p) {
			return true
		}
	}
	if b := filepath.Clean(strings.TrimSpace(s.cfg.BaseDir)); b != "" && b != "." {
		if clean == b {
			return true
		}
	}
	return false
}

// checkScanGrantedPath 校验一次「由扫描结果授权」的写操作。
//
// 这是整个磁盘清理功能的安全内核：不是把整个文件系统对 gpc 解锁，
// 而是只允许操作「确实出现在某次扫描结果里的大文件」。
// 即使调用方被攻破，能造成的最坏后果也只是删掉几个大文件，
// 而 /etc/passwd、/bin/sh 这类关键文件只有几 KB，根本进不了大文件列表。
//
// 四道校验缺一不可：
//  1. scanId 有效且未过期
//  2. 路径确实在该次扫描结果中（EvalSymlinks 后比对，防软链替换）
//  3. 体积不低于硬门槛（无视请求里传的 minSize）
//  4. 不在保护名单里，且是普通文件
func (s *Server) checkScanGrantedPath(scanID string, path string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || !filepath.IsAbs(clean) {
		return "", errPathNotAbsolute
	}
	if s.isDiskProtectedPath(clean) {
		return "", errPathProtected
	}
	// 先解析软链再比对：否则可以先扫出 /tmp/big.bin，再把它换成指向 /etc/shadow 的软链
	resolved := clean
	if r, err := filepath.EvalSymlinks(clean); err == nil {
		resolved = r
	}
	if s.isDiskProtectedPath(resolved) {
		return "", errPathProtected
	}
	if !diskScanStore.contains(scanID, resolved) && !diskScanStore.contains(scanID, clean) {
		return "", errPathNotInScan
	}
	fi, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if !fi.Mode().IsRegular() {
		return "", errNotRegularFile
	}
	if fi.Size() < diskGrantMinSize {
		return "", errFileTooSmall
	}
	return clean, nil
}
