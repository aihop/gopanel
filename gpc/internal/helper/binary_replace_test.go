//go:build !windows

package helper

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// startRunningBinary 在 dst 处放一个真实可执行文件并跑起来，
// 用来复现「替换正在运行的二进制」这个场景。
//
// 必须现编译，不能直接拷 /bin/sleep：macOS 的代码签名会让复制出来的系统二进制
// 一启动就被杀掉，进程根本没跑起来，测试会得出「这个平台允许覆盖」的错误结论。
func startRunningBinary(t *testing.T, dst string) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("没有 go 工具链，无法编译用于测试的二进制")
	}
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "main.go")
	program := "package main\n\nimport \"time\"\n\nfunc main() { time.Sleep(60 * time.Second) }\n"
	if err := os.WriteFile(srcFile, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "build", "-o", dst, srcFile).CombinedOutput(); err != nil {
		t.Skipf("编译测试二进制失败: %v\n%s", err, out)
	}

	cmd := exec.Command(dst)
	if err := cmd.Start(); err != nil {
		t.Skipf("无法启动测试二进制: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	// 等内核真正把它标记为「正在执行」，否则 ETXTBSY 复现不稳定
	time.Sleep(300 * time.Millisecond)
	// 必须确认它还活着：进程若已退出，后面「能否覆盖写」的结论就没有意义
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Skipf("测试二进制没能保持运行（%v），无法验证 ETXTBSY", err)
	}
}

// 复现 gp-agent 自更新的报错：直接覆盖写正在运行的二进制会 text file busy
func TestCopyFileFailsOnRunningBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "gp-agent")
	startRunningBinary(t, dst)

	newSrc := filepath.Join(dir, "new-bin")
	if err := os.WriteFile(newSrc, []byte("#!/bin/sh\necho new\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := copyFile(newSrc, dst, 0o755)
	if err == nil {
		t.Skip("该平台允许覆盖运行中的二进制，本用例不适用")
	}
	t.Logf("copyFile 如期失败: %v", err)
}

// replaceBinary 必须能在同样的场景下成功
func TestReplaceBinaryWorksOnRunningBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "gp-agent")
	startRunningBinary(t, dst)

	newSrc := filepath.Join(dir, "new-bin")
	want := []byte("#!/bin/sh\necho new-version\n")
	if err := os.WriteFile(newSrc, want, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(newSrc, dst, 0o755); err != nil {
		t.Fatalf("替换正在运行的二进制应当成功: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("目标内容没被替换:\n%s", got)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Fatalf("权限应为 0755，实际 %v", fi.Mode().Perm())
	}

	// 老进程还在跑，说明 rename 没有影响到它
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if len(e.Name()) > 8 && e.Name()[:8] == ".gp-bin-" {
			t.Errorf("残留了临时文件: %s", e.Name())
		}
	}
}

func TestReplaceBinaryCreatesMissingTarget(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "sub", "deep", "gp-agent")
	src := filepath.Join(dir, "src-bin")
	if err := os.WriteFile(src, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceBinary(src, dst, 0o755); err != nil {
		t.Fatalf("目标目录不存在时应自动创建: %v", err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Fatalf("权限应为 0755，实际 %v", fi.Mode().Perm())
	}
}

func TestReplaceBinaryFailsCleanlyOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "gp-agent")
	if err := replaceBinary(filepath.Join(dir, "nope"), dst, 0o755); err == nil {
		t.Fatal("源文件不存在应报错")
	}
	// 失败时不能留下半截文件或临时文件
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("失败后目录应保持干净，实际残留: %v", names)
	}
}
