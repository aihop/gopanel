//go:build !windows

package helper

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestServer(t *testing.T, baseDir string) *Server {
	t.Helper()
	return NewServer(Config{BaseDir: baseDir, FileRoots: []string{baseDir}})
}

func writeSized(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}
}

func scanDir(t *testing.T, s *Server, dir string, minSize int64) diskScanResult {
	t.Helper()
	out, err := s.actionDiskScan(context.Background(), map[string]interface{}{
		"roots":       []interface{}{dir},
		"minSize":     float64(minSize),
		"crossDevice": true, // t.TempDir 在部分环境是独立挂载点，测试里不做设备限制
	})
	if err != nil {
		t.Fatalf("DISK_SCAN 失败: %v", err)
	}
	var res diskScanResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	return res
}

func TestDiskScanFindsBigFiles(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "big.bin")
	writeSized(t, big, 20<<20)
	writeSized(t, filepath.Join(dir, "small.bin"), 1024)

	res := scanDir(t, newTestServer(t, dir), dir, 10<<20)
	if res.ScanID == "" {
		t.Fatal("应返回 scanId")
	}
	if len(res.Files) != 1 || res.Files[0].Path != big {
		t.Fatalf("应只命中 big.bin: %+v", res.Files)
	}
	if res.ScannedFiles != 2 {
		t.Fatalf("ScannedFiles want 2, got %d", res.ScannedFiles)
	}
}

func TestRemoveRequiresValidScanID(t *testing.T) {
	dir := t.TempDir()
	// 放在 FileRoots 之外，确保只能靠 scanId 授权
	outside := t.TempDir()
	target := filepath.Join(outside, "big.bin")
	writeSized(t, target, 20<<20)

	s := newTestServer(t, dir)

	t.Run("无 scanId 时受 FileRoots 限制", func(t *testing.T) {
		_, err := s.actionFileRemove(context.Background(), map[string]interface{}{"path": target})
		if err == nil {
			t.Fatal("roots 之外的路径不带 scanId 不应能删")
		}
	})

	t.Run("伪造的 scanId 无效", func(t *testing.T) {
		_, err := s.actionFileRemove(context.Background(), map[string]interface{}{
			"path": target, "scanId": "deadbeefdeadbeefdeadbeefdeadbeef",
		})
		if err != errPathNotInScan {
			t.Fatalf("want errPathNotInScan, got %v", err)
		}
	})

	t.Run("扫描授权后可删", func(t *testing.T) {
		res := scanDir(t, s, outside, 10<<20)
		_, err := s.actionFileRemove(context.Background(), map[string]interface{}{
			"path": target, "scanId": res.ScanID,
		})
		if err != nil {
			t.Fatalf("授权后应可删: %v", err)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatal("文件应已被删除")
		}
	})
}

func TestScanIDCannotDeleteSmallFile(t *testing.T) {
	// 攻击场景：用 minSize=1 扫一遍，试图把小文件也纳入可删集合。
	// diskGrantMinSize 是硬门槛，无视请求里的 minSize。
	dir := t.TempDir()
	small := filepath.Join(dir, "small.conf")
	writeSized(t, small, 2048)
	s := newTestServer(t, dir)

	res := scanDir(t, s, dir, 1)
	_, err := s.actionFileRemove(context.Background(), map[string]interface{}{
		"path": small, "scanId": res.ScanID,
	})
	if err == nil {
		t.Fatal("小文件不应能通过 scanId 授权删除")
	}
	if _, statErr := os.Lstat(small); statErr != nil {
		t.Fatal("文件不应被删除")
	}
}

func TestScanIDCannotDeleteProtectedPath(t *testing.T) {
	s := newTestServer(t, "/opt/gopanel")
	// 直接把保护路径塞进 store，模拟“扫描结果里混进了受保护路径”的最坏情况
	scanID, err := diskScanStore.save([]string{"/etc/passwd", "/bin/sh", "/opt/gopanel"})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/etc/passwd", "/bin/sh", "/opt/gopanel", "/"} {
		if _, err := s.checkScanGrantedPath(scanID, p); err != errPathProtected {
			t.Errorf("%s 必须被保护名单拦下，got %v", p, err)
		}
	}
}

func TestScanIDCannotDeleteViaSymlinkSwap(t *testing.T) {
	// 攻击场景：扫描时是普通大文件，删除前把它换成指向受保护文件的软链
	dir := t.TempDir()
	target := filepath.Join(dir, "big.bin")
	writeSized(t, target, 20<<20)
	s := newTestServer(t, dir)
	res := scanDir(t, s, dir, 10<<20)

	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", target); err != nil {
		t.Skipf("环境不支持符号链接: %v", err)
	}
	_, err := s.actionFileRemove(context.Background(), map[string]interface{}{
		"path": target, "scanId": res.ScanID,
	})
	if err == nil {
		t.Fatal("换成软链后不应能删")
	}
	if err != errPathProtected && err != errNotRegularFile {
		t.Fatalf("应被保护名单或普通文件校验拦下，got %v", err)
	}
}

func TestScanIDIsSingleUsePerPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "big.bin")
	writeSized(t, target, 20<<20)
	s := newTestServer(t, dir)
	res := scanDir(t, s, dir, 10<<20)

	if _, err := s.actionFileRemove(context.Background(), map[string]interface{}{
		"path": target, "scanId": res.ScanID,
	}); err != nil {
		t.Fatal(err)
	}
	// 同一路径重建后不能用旧 scanId 再删一次
	writeSized(t, target, 20<<20)
	if _, err := s.actionFileRemove(context.Background(), map[string]interface{}{
		"path": target, "scanId": res.ScanID,
	}); err != errPathNotInScan {
		t.Fatalf("删除后该路径应已从授权集合移除，got %v", err)
	}
}

func TestFileTruncate(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "app.log")
	writeSized(t, target, 20<<20)
	s := newTestServer(t, dir)
	res := scanDir(t, s, dir, 10<<20)

	if _, err := s.actionFileTruncate(context.Background(), map[string]interface{}{
		"path": target, "scanId": res.ScanID,
	}); err != nil {
		t.Fatalf("truncate 失败: %v", err)
	}
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatal("truncate 后文件必须还在（inode 保留，正在写它的进程不受影响）")
	}
	if fi.Size() != 0 {
		t.Fatalf("size want 0, got %d", fi.Size())
	}
}

func TestFileTruncateRejectsProtected(t *testing.T) {
	s := newTestServer(t, "/opt/gopanel")
	if _, err := s.actionFileTruncate(context.Background(), map[string]interface{}{
		"path": "/etc/passwd",
	}); err == nil {
		t.Fatal("不应能 truncate 受保护路径")
	}
}
