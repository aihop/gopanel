package diskscan

import (
	"os"
	"path/filepath"
	"strings"
)

// protectedRoots 这些路径本身及其直接语义不允许被磁盘清理功能删除。
// 注意语义：目录本身不可删；目录“里面”的文件按 protectedPrefixes 单独判断。
var protectedRoots = []string{
	"/", "/bin", "/sbin", "/lib", "/lib32", "/lib64", "/libx32",
	"/usr", "/etc", "/boot", "/root", "/home", "/var", "/opt", "/srv",
	"/proc", "/sys", "/dev", "/run",
}

// protectedPrefixes 这些子树里的任何东西都不允许删——删了系统就起不来了。
// /var 和 /home 这类不整体禁止：/var/log 下的大日志正是这个功能要清理的目标。
var protectedPrefixes = []string{
	"/bin/", "/sbin/", "/lib/", "/lib32/", "/lib64/", "/libx32/",
	"/usr/bin/", "/usr/sbin/", "/usr/lib/", "/usr/lib64/", "/usr/libexec/",
	"/etc/", "/boot/", "/proc/", "/sys/", "/dev/", "/run/",
}

// IsProtected 判断路径是否受保护、禁止删除。
// baseDir 是面板安装目录：面板自身目录整体不可删，否则一次误操作就把面板删了；
// 但 baseDir 下的 tmp/日志/备份等子目录是允许清理的，所以只保护 baseDir 本身。
//
// 这份名单在 gpc 侧有一份等价实现（gpc/internal/helper/disk_protect_unix.go）。
// 两边都要有：面板以 root 运行时直接本地删，gpc 是 rootless 场景下最后一道关卡。
// 改动其中一边时必须同步另一边。
func IsProtected(path string, baseDir string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." {
		return true
	}
	if !filepath.IsAbs(clean) {
		return true
	}
	for _, r := range protectedRoots {
		if clean == r {
			return true
		}
	}
	for _, p := range protectedPrefixes {
		if strings.HasPrefix(clean, p) {
			return true
		}
	}
	if b := filepath.Clean(strings.TrimSpace(baseDir)); b != "" && b != "." {
		if clean == b {
			return true
		}
	}
	return false
}

// IsRegularFile 只允许操作普通文件：目录要单独确认，
// 设备/socket/管道删了没有意义还可能弄坏系统。
func IsRegularFile(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}
