package diskscan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScanTopN(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "big1.bin"), 5000)
	writeFile(t, filepath.Join(root, "big2.bin"), 3000)
	writeFile(t, filepath.Join(root, "sub", "big3.bin"), 4000)
	writeFile(t, filepath.Join(root, "small.txt"), 10)

	res, err := Scan(context.Background(), Options{Roots: []string{root}, MinSize: 1000, TopN: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 {
		t.Fatalf("TopN=2 应只留 2 条，got %d: %+v", len(res.Files), res.Files)
	}
	if res.Files[0].Size != 5000 || res.Files[1].Size != 4000 {
		t.Fatalf("应按大小倒序留最大的两个: %+v", res.Files)
	}
	// small.txt 低于 MinSize 不进结果，但仍要计入扫描统计
	if res.ScannedFiles != 4 {
		t.Fatalf("ScannedFiles want 4, got %d", res.ScannedFiles)
	}
	if res.ScannedBytes != 5000+3000+4000+10 {
		t.Fatalf("ScannedBytes 不对: %d", res.ScannedBytes)
	}
}

func TestScanDirAggregation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "1.bin"), 1000)
	writeFile(t, filepath.Join(root, "a", "2.bin"), 2000)
	writeFile(t, filepath.Join(root, "b", "1.bin"), 500)

	res, err := Scan(context.Background(), Options{Roots: []string{root}, MinSize: 1, TopDirs: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dirs) == 0 {
		t.Fatal("应有目录聚合结果")
	}
	got := res.Dirs[0]
	if got.Path != filepath.Join(root, "a") || got.Size != 3000 || got.Count != 2 {
		t.Fatalf("目录聚合不对: %+v", got)
	}
}

func TestScanSkipsSymlinkAndIrregular(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.bin")
	writeFile(t, target, 2000)
	if err := os.Symlink(target, filepath.Join(root, "link.bin")); err != nil {
		t.Skipf("环境不支持符号链接: %v", err)
	}

	res, err := Scan(context.Background(), Options{Roots: []string{root}, MinSize: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ScannedFiles != 1 {
		t.Fatalf("符号链接不应被统计，got %d: %+v", res.ScannedFiles, res.Files)
	}
	for _, f := range res.Files {
		if filepath.Base(f.Path) == "link.bin" {
			t.Fatal("符号链接不应出现在结果里")
		}
	}
}

func TestScanSkipDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "a.bin"), 2000)
	writeFile(t, filepath.Join(root, "drop", "b.bin"), 3000)

	res, err := Scan(context.Background(), Options{
		Roots:    []string{root},
		MinSize:  1,
		SkipDirs: []string{filepath.Join(root, "drop")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Files {
		if filepath.Base(f.Path) == "b.bin" {
			t.Fatalf("被跳过目录里的文件不应出现: %+v", res.Files)
		}
	}
	if len(res.Files) != 1 {
		t.Fatalf("want 1, got %d", len(res.Files))
	}
}

func TestScanCancel(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 200; i++ {
		writeFile(t, filepath.Join(root, "d", string(rune('a'+i%26)), "f.bin"), 100)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, Options{Roots: []string{root}, MinSize: 1}, nil); err == nil {
		t.Fatal("已取消的 context 应返回错误")
	}
}

func TestScanProgress(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.bin"), 1000)
	called := 0
	var last Progress
	_, err := Scan(context.Background(), Options{Roots: []string{root}, MinSize: 1}, func(p Progress) {
		called++
		last = p
	})
	if err != nil {
		t.Fatal(err)
	}
	if called == 0 {
		t.Fatal("应至少回调一次（收尾强制上报）")
	}
	if last.ScannedFiles != 1 {
		t.Fatalf("最终进度应为 1 个文件，got %+v", last)
	}
}

func TestCategorize(t *testing.T) {
	cases := map[string]string{
		"/var/log/nginx/access.log":              "log",
		"/var/lib/docker/overlay2/abc/diff/x.so": "container",
		"/var/cache/apt/archives/foo.deb":        "cache",
		"/root/backup/db.sql":                    "backup",
		"/tmp/build/x.tmp":                       "temp",
		"/home/u/release.tar.gz":                 "archive",
		"/var/crash/core.1234":                   "coredump",
		"/opt/app/server":                        "other",
	}
	for path, want := range cases {
		if got := Categorize(path); got != want {
			t.Errorf("Categorize(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestAlwaysSkipPseudoFS(t *testing.T) {
	// 不真去扫 /proc，只验证判定逻辑
	for _, p := range []string{"/proc", "/proc/1/fd", "/sys/kernel", "/dev/shm", "/run/user/0"} {
		if !isSkipped(p, alwaysSkipDirs) {
			t.Errorf("%s 应被跳过", p)
		}
	}
	if isSkipped("/procession/data", alwaysSkipDirs) {
		t.Error("/procession 不应被误判成 /proc")
	}
}
