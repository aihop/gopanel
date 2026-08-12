package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCodeResidueDirectoryAcceptsManagedNamesOnly(t *testing.T) {
	accepted := map[string]uint{
		"session_1":         1,
		"session_108":       108,
		"delivery_42":       42,
		"delivery_42_multi": 42,
	}
	for name, expected := range accepted {
		id, ok := parseCodeResidueDirectory(name)
		if !ok || id != expected {
			t.Fatalf("%q should parse to %d, got %d (ok=%v)", name, expected, id, ok)
		}
	}
	for _, name := range []string{
		"session_", "session_abc", "session_0", "delivery_", "delivery_0_multi",
		"..", ".", "node_modules", "session_1x", "sessions_1", "_multi",
	} {
		if id, ok := parseCodeResidueDirectory(name); ok {
			t.Fatalf("%q should be rejected, parsed as %d", name, id)
		}
	}
}

// 孤儿清理没有会话记录可依托，边界只能靠这个函数自己把。
// 少一道校验，一个被污染的路径就能变成任意目录删除。
func TestRemoveCodeResidueDirectoriesRefusesPathsOutsideManagedRoot(t *testing.T) {
	withAIProjectBaseDir(t)
	const userID = 9
	root := aiProjectWorktreeRoot(userID)
	if err := os.MkdirAll(root, 0750); err != nil {
		t.Fatal(err)
	}
	outsider := filepath.Join(t.TempDir(), "precious")
	if err := os.MkdirAll(outsider, 0750); err != nil {
		t.Fatal(err)
	}

	refused := map[string]string{
		"管理目录之外":  outsider,
		"用 .. 逃逸": filepath.Join(root, "session_1", "..", "..", "escape"),
		"命名不受管理":  filepath.Join(root, "random_dir"),
		"管理目录本身":  root,
	}
	for label, target := range refused {
		if err := removeCodeResidueDirectories(userID, []string{target}); err == nil {
			t.Fatalf("%s 应被拒绝：%s", label, target)
		}
	}
	if _, err := os.Stat(outsider); err != nil {
		t.Fatalf("管理目录之外的目录被删除了：%v", err)
	}

	managed := filepath.Join(root, "session_1")
	if err := os.MkdirAll(managed, 0750); err != nil {
		t.Fatal(err)
	}
	if err := removeCodeResidueDirectories(userID, []string{managed}); err != nil {
		t.Fatalf("受管目录应可清理：%v", err)
	}
	if _, err := os.Stat(managed); !os.IsNotExist(err) {
		t.Fatalf("受管目录未被清理：%v", err)
	}
}

// 符号链接指向哪里不受管理目录约束，跟着删等于把边界让了出去。
func TestRemoveCodeResidueDirectoriesRefusesSymlink(t *testing.T) {
	withAIProjectBaseDir(t)
	const userID = 11
	root := aiProjectWorktreeRoot(userID)
	if err := os.MkdirAll(root, 0750); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(target, 0750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "session_5")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := removeCodeResidueDirectories(userID, []string{link}); err == nil {
		t.Fatal("符号链接应被拒绝")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("符号链接的目标被删除了：%v", err)
	}
}

func TestScanCodeWorktreeResiduesGroupsDirectoriesBySession(t *testing.T) {
	withAIProjectBaseDir(t)
	const userID = 13
	root := aiProjectWorktreeRoot(userID)
	for _, name := range []string{
		"session_70", "delivery_70", "delivery_70_multi", "session_71", "unrelated",
	} {
		if err := os.MkdirAll(filepath.Join(root, name), 0750); err != nil {
			t.Fatal(err)
		}
	}
	residues, err := scanCodeWorktreeResidues(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(residues) != 2 {
		t.Fatalf("同一会话的三个目录应聚成一条，实际 %d 条：%#v", len(residues), residues)
	}
	if residues[0].SessionID != 71 || residues[1].SessionID != 70 {
		t.Fatalf("应按会话号倒序：%d, %d", residues[0].SessionID, residues[1].SessionID)
	}
	if len(residues[1].Directories) != 3 {
		t.Fatalf("会话 70 应聚合三个目录：%#v", residues[1].Directories)
	}
	for _, residue := range residues {
		for _, directory := range residue.Directories {
			if strings.Contains(directory, "unrelated") {
				t.Fatalf("不受管理的目录被纳入了残留：%s", directory)
			}
		}
	}
}

func TestScanCodeWorktreeResiduesReturnsEmptyWhenRootMissing(t *testing.T) {
	withAIProjectBaseDir(t)
	residues, err := scanCodeWorktreeResidues(99)
	if err != nil {
		t.Fatalf("管理目录尚未创建时不该报错：%v", err)
	}
	if len(residues) != 0 {
		t.Fatalf("应返回空列表：%#v", residues)
	}
}
