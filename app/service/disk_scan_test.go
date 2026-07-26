package service

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/zlog"
	"github.com/aihop/gopanel/utils/diskscan"
)

func setupDiskTest(t *testing.T) string {
	t.Helper()
	global.LOG = zlog.New(io.Discard, zlog.DebugLevel, &zlog.TextFormatter{})
	base := t.TempDir()
	oldBase := global.CONF.System.BaseDir
	global.CONF.System.BaseDir = base
	t.Cleanup(func() { global.CONF.System.BaseDir = oldBase })
	return base
}

// 直接构造一个已完成的任务，避免测试依赖真实扫描耗时
func injectTask(t *testing.T, files map[string]int64) string {
	t.Helper()
	task := &DiskScanTask{
		ID:        "test-" + t.Name(),
		Status:    DiskScanStatusSuccess,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
		paths:     map[string]int64{},
		Result:    &diskscan.Result{},
		cancel:    func() {},
	}
	for p, s := range files {
		task.paths[p] = s
		task.Result.Files = append(task.Result.Files, diskscan.FileItem{Path: p, Size: s})
	}
	diskScanMgr.mu.Lock()
	diskScanMgr.tasks[task.ID] = task
	diskScanMgr.mu.Unlock()
	t.Cleanup(func() {
		diskScanMgr.mu.Lock()
		delete(diskScanMgr.tasks, task.ID)
		diskScanMgr.mu.Unlock()
	})
	return task.ID
}

func TestCleanRejectsPathNotInScan(t *testing.T) {
	setupDiskTest(t)
	dir := t.TempDir()
	known := filepath.Join(dir, "known.bin")
	unknown := filepath.Join(dir, "unknown.bin")
	if err := os.WriteFile(known, make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unknown, make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	taskID := injectTask(t, map[string]int64{known: 100})

	res, err := CleanDiskPaths(taskID, []string{unknown}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].OK {
		t.Fatalf("不在扫描结果里的路径必须被拒绝: %+v", res)
	}
	if _, statErr := os.Lstat(unknown); statErr != nil {
		t.Fatal("文件不应被删除")
	}
}

func TestCleanRejectsProtectedPath(t *testing.T) {
	base := setupDiskTest(t)
	// 即便被塞进扫描结果，保护名单也必须拦下
	taskID := injectTask(t, map[string]int64{"/etc/passwd": 100, base: 100})

	res, err := CleanDiskPaths(taskID, []string{"/etc/passwd", base}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.OK {
			t.Fatalf("%s 必须被保护名单拦下", r.Path)
		}
	}
	if _, statErr := os.Lstat("/etc/passwd"); statErr != nil {
		t.Fatal("/etc/passwd 居然被动了")
	}
}

func TestCleanDeletesAndTruncates(t *testing.T) {
	setupDiskTest(t)
	dir := t.TempDir()
	del := filepath.Join(dir, "del.bin")
	trunc := filepath.Join(dir, "app.log")
	if err := os.WriteFile(del, make([]byte, 2048), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trunc, make([]byte, 4096), 0644); err != nil {
		t.Fatal(err)
	}
	taskID := injectTask(t, map[string]int64{del: 2048, trunc: 4096})

	res, err := CleanDiskPaths(taskID, []string{del}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].OK || res[0].Freed != 2048 {
		t.Fatalf("删除应成功并统计释放量: %+v", res)
	}
	if _, statErr := os.Lstat(del); !os.IsNotExist(statErr) {
		t.Fatal("文件应已删除")
	}

	res, err = CleanDiskPaths(taskID, []string{trunc}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].OK {
		t.Fatalf("truncate 应成功: %+v", res)
	}
	fi, statErr := os.Lstat(trunc)
	if statErr != nil {
		t.Fatal("truncate 后文件必须保留（inode 不变，写它的进程不受影响）")
	}
	if fi.Size() != 0 {
		t.Fatalf("size want 0, got %d", fi.Size())
	}
}

func TestCleanRejectsContainerStoreOnDelete(t *testing.T) {
	setupDiskTest(t)
	p := "/var/lib/docker/overlay2/abc/diff/layer.bin"
	taskID := injectTask(t, map[string]int64{p: 100})
	res, err := CleanDiskPaths(taskID, []string{p}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].OK {
		t.Fatal("容器存储目录的文件不应直接删除")
	}
	if res[0].Message == "" {
		t.Fatal("应给出引导到容器页面的提示")
	}
}

func TestCleanRejectsExpiredOrUnknownTask(t *testing.T) {
	setupDiskTest(t)
	if _, err := CleanDiskPaths("no-such-task", []string{"/tmp/x"}, false); err == nil {
		t.Fatal("未知任务必须报错")
	}
}

func TestOnlyOneScanAtATime(t *testing.T) {
	setupDiskTest(t)
	diskScanMgr.mu.Lock()
	diskScanMgr.tasks["busy"] = &DiskScanTask{
		ID: "busy", Status: DiskScanStatusRunning,
		ExpiresAt: time.Now().Add(time.Minute), cancel: func() {},
	}
	diskScanMgr.running = "busy"
	diskScanMgr.mu.Unlock()
	t.Cleanup(func() {
		diskScanMgr.mu.Lock()
		delete(diskScanMgr.tasks, "busy")
		diskScanMgr.running = ""
		diskScanMgr.mu.Unlock()
	})

	if _, err := StartDiskScan(DiskScanRequest{Roots: []string{t.TempDir()}}); err == nil {
		t.Fatal("已有任务在跑时不应允许再启动")
	}
}

// 端到端：真实扫描 → 拿结果 → 删除，验证 taskId 全链路
func TestScanThenCleanEndToEnd(t *testing.T) {
	setupDiskTest(t)
	dir := t.TempDir()
	big := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(big, make([]byte, 300*1024), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tiny.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	task, err := StartDiskScan(DiskScanRequest{
		Roots: []string{dir}, MinSize: 100 * 1024, TopN: 10, CrossDevice: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		diskScanMgr.mu.Lock()
		delete(diskScanMgr.tasks, task.ID)
		diskScanMgr.running = ""
		diskScanMgr.mu.Unlock()
	})

	var done *DiskScanTask
	for i := 0; i < 100; i++ {
		snap, ok := GetDiskScanTask(task.ID)
		if ok && snap.Status != DiskScanStatusRunning {
			done = snap
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if done == nil {
		t.Fatal("扫描未在预期时间内结束")
	}
	if done.Status != DiskScanStatusSuccess {
		t.Fatalf("扫描应成功: status=%s err=%s", done.Status, done.Error)
	}
	if len(done.Result.Files) != 1 || done.Result.Files[0].Path != big {
		t.Fatalf("应只命中 big.bin: %+v", done.Result.Files)
	}

	res, err := CleanDiskPaths(task.ID, []string{big}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].OK {
		t.Fatalf("删除应成功: %+v", res)
	}
	if _, err := os.Lstat(big); !os.IsNotExist(err) {
		t.Fatal("文件应已删除")
	}
}
