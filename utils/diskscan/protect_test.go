package diskscan

import "testing"

func TestIsProtected(t *testing.T) {
	baseDir := "/opt/gopanel"

	mustProtect := []string{
		"/", "/etc", "/etc/passwd", "/bin", "/bin/sh", "/usr/bin/ssh",
		"/boot/vmlinuz", "/lib/x86_64-linux-gnu/libc.so.6", "/proc/1/mem",
		"/sys/kernel", "/dev/sda", "/var", "/home", "/root", "/usr",
		"/opt/gopanel", "relative/path", "",
	}
	for _, p := range mustProtect {
		if !IsProtected(p, baseDir) {
			t.Errorf("%q 必须受保护", p)
		}
	}

	mustAllow := []string{
		"/var/log/nginx/access.log",
		"/var/lib/docker/overlay2/x/diff/big.bin",
		"/home/ubuntu/dump.sql",
		"/tmp/build.tar.gz",
		"/opt/gopanel/tmp/cache.bin",
		"/opt/app/releases/old.tar",
		"/data/backup/2026.sql",
	}
	for _, p := range mustAllow {
		if IsProtected(p, baseDir) {
			t.Errorf("%q 不应被保护（这正是要清理的目标）", p)
		}
	}
}

func TestIsProtectedTrailingSlashAndDotDot(t *testing.T) {
	// filepath.Clean 会把这些归一化，避免用 /etc/ 或 /var/log/../../etc 绕过
	for _, p := range []string{"/etc/", "/var/log/../../etc/passwd", "/usr/bin/../bin/sh"} {
		if !IsProtected(p, "/opt/gopanel") {
			t.Errorf("%q 归一化后应受保护", p)
		}
	}
}
