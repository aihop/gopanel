package diskscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnnotateRemovable(t *testing.T) {
	dir := t.TempDir()
	mine := filepath.Join(dir, "mine.bin")
	if err := os.WriteFile(mine, make([]byte, 10), 0644); err != nil {
		t.Fatal(err)
	}

	files := []FileItem{
		{Path: mine},
		{Path: "/etc/passwd"},
		{Path: "/var/lib/docker/overlay2/x/diff/a.bin", IsContainer: true},
		{Path: "/var/log/journal/abc123/system@0005.journal"},
	}
	AnnotateRemovable(files, "/opt/gopanel", os.Geteuid(), false)

	if !files[0].Removable {
		t.Errorf("自己创建的文件应可清理: %+v", files[0])
	}
	if files[1].Removable || files[1].Reason == "" {
		t.Errorf("/etc/passwd 必须标为不可清理: %+v", files[1])
	}
	if files[2].Removable || files[2].Reason == "" {
		t.Errorf("容器层文件必须标为不可清理: %+v", files[2])
	}
	if files[3].Removable || files[3].Reason == "" {
		t.Errorf("journald 内部文件必须标为不可清理: %+v", files[3])
	}
}

func TestIsJournalInternal(t *testing.T) {
	yes := []string{
		"/var/log/journal/abc/system.journal",
		"/var/log/journal/abc/user-1000@0006.journal~",
		"/run/log/journal/x/system.journal",
	}
	no := []string{
		"/var/log/nginx/access.log",
		"/var/log/syslog",
		"/home/x/journal.txt",
	}
	for _, p := range yes {
		if !IsJournalInternal(p) {
			t.Errorf("%s 应识别为 journald 内部文件", p)
		}
	}
	for _, p := range no {
		if IsJournalInternal(p) {
			t.Errorf("%s 不应识别为 journald 内部文件", p)
		}
	}
}

// macOS: / 是只读封印卷，上面的文件任何权限都删不掉，必须标出来
func TestAnnotateReadOnlyVolume(t *testing.T) {
	const p = "/System/Library/CoreServices/SystemVersion.plist"
	if _, err := os.Lstat(p); err != nil {
		t.Skip("非 macOS 或路径不存在")
	}
	files := []FileItem{{Path: p}}
	AnnotateRemovable(files, "/opt/gopanel", os.Geteuid(), true) // 即便能提权也删不掉
	if files[0].Removable {
		t.Fatal("只读卷上的文件必须标为不可清理")
	}
	t.Logf("原因: %s", files[0].Reason)
}

// 非 root 运行且无 gpc 时，属主不是自己的文件删不掉
func TestAnnotateForeignOwner(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 运行，属主检查不适用")
	}
	const p = "/usr/local/bin"
	if _, err := os.Lstat(p); err != nil {
		t.Skip("路径不存在")
	}
	uid, ok := ownerUID(p)
	if !ok || uid == os.Geteuid() {
		t.Skip("该路径属主就是当前用户，用例不适用")
	}
	files := []FileItem{{Path: p}}
	AnnotateRemovable(files, "/opt/gopanel", os.Geteuid(), false)
	if files[0].Removable {
		t.Fatal("属主不是当前用户且无法提权时应标为不可清理")
	}
	t.Logf("原因: %s", files[0].Reason)
}
