package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 渲染出与 stepBuild 里同形状的 docker shim，用假的真实二进制验证注入行为
func newShimDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	fakeDocker := filepath.Join(dir, "real-docker")
	if err := os.WriteFile(fakeDocker, []byte("#!/bin/sh\nfor a in \"$@\"; do echo \"$a\"; done\n"), 0755); err != nil {
		t.Fatal(err)
	}
	shim := "#!/bin/sh\n" + runEnvInjectSnippet + "\nexec \"$REAL_DOCKER_BIN\" \"$@\"\n"
	shimPath := filepath.Join(dir, "docker")
	if err := os.WriteFile(shimPath, []byte(shim), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runShim(t *testing.T, dir string, version string, args ...string) []string {
	t.Helper()
	cmd := exec.Command(filepath.Join(dir, "docker"), args...)
	cmd.Env = append(os.Environ(), "REAL_DOCKER_BIN="+filepath.Join(dir, "real-docker"), "PIPELINE_VERSION="+version)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim 执行失败: %v, out=%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func TestRunEnvInjectSnippet(t *testing.T) {
	dir := newShimDir(t)
	const wantGopanel = "GOPANEL_PIPELINE_VERSION=2.3.4"
	const wantPipeline = "PIPELINE_VERSION=2.3.4"

	t.Run("run 子命令注入到最前", func(t *testing.T) {
		got := runShim(t, dir, "2.3.4", "run", "--rm", "-p", "3000:3000", "myimg", "npm", "start")
		want := []string{"run", "-e", wantGopanel, "-e", wantPipeline, "--rm", "-p", "3000:3000", "myimg", "npm", "start"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("got %v\nwant %v", got, want)
		}
	})

	t.Run("container run 也注入", func(t *testing.T) {
		got := runShim(t, dir, "2.3.4", "container", "run", "myimg")
		want := []string{"container", "run", "-e", wantGopanel, "-e", wantPipeline, "myimg"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("got %v\nwant %v", got, want)
		}
	})

	t.Run("create 也注入", func(t *testing.T) {
		got := runShim(t, dir, "2.3.4", "create", "myimg")
		if len(got) < 2 || got[1] != "-e" {
			t.Fatalf("create 没注入: %v", got)
		}
	})

	t.Run("build/compose 不动", func(t *testing.T) {
		for _, args := range [][]string{{"build", "-t", "x", "."}, {"compose", "up", "-d"}, {"ps"}} {
			got := runShim(t, dir, "2.3.4", args...)
			if strings.Join(got, "|") != strings.Join(args, "|") {
				t.Fatalf("%v 被改写成了 %v", args, got)
			}
		}
	})

	t.Run("用户自己的 -e 排在后面（docker 取后者）", func(t *testing.T) {
		got := runShim(t, dir, "2.3.4", "run", "-e", "PIPELINE_VERSION=user", "myimg")
		ourIdx, userIdx := -1, -1
		for i, v := range got {
			if v == wantPipeline {
				ourIdx = i
			}
			if v == "PIPELINE_VERSION=user" {
				userIdx = i
			}
		}
		if ourIdx == -1 || userIdx == -1 || userIdx < ourIdx {
			t.Fatalf("用户值必须排在注入值之后: %v", got)
		}
	})

	t.Run("没有版本号时不注入", func(t *testing.T) {
		got := runShim(t, dir, "", "run", "myimg")
		want := []string{"run", "myimg"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("got %v want %v", got, want)
		}
	})
}
