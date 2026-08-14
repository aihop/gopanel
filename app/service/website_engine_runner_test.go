package service

import (
	"strings"
	"testing"
)

func TestParseRunnerConfigDetectsCustomWorkingDir(t *testing.T) {
	t.Run("default working dir", func(t *testing.T) {
		rc := parseRunnerConfig(nil)
		if rc.WorkingDir != "/var/www/app" {
			t.Fatalf("expected default workingDir, got %q", rc.WorkingDir)
		}
		if rc.HasCustomWorkingDir {
			t.Fatalf("expected default config to not mark custom workingDir")
		}
	})

	t.Run("custom working dir", func(t *testing.T) {
		rc := parseRunnerConfig(map[string]interface{}{
			"workingDir": "/srv/app",
		})
		if rc.WorkingDir != "/srv/app" {
			t.Fatalf("expected custom workingDir, got %q", rc.WorkingDir)
		}
		if !rc.HasCustomWorkingDir {
			t.Fatalf("expected config to mark custom workingDir")
		}
	})
}

func TestMergeRunnerEnvsInjectsVersionAliases(t *testing.T) {
	toMap := func(envs []string) map[string]string {
		m := make(map[string]string, len(envs))
		for _, e := range envs {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				m[parts[0]] = parts[1]
			}
		}
		return m
	}

	t.Run("三个别名都注入", func(t *testing.T) {
		rc := parseRunnerConfig(nil)
		got := toMap(mergeRunnerEnvs([]string{"PATH=/usr/bin"}, rc, "3000", "1.4.2"))
		for _, key := range []string{"GOPANEL_PIPELINE_VERSION", "PIPELINE_VERSION", "VERSION"} {
			if got[key] != "1.4.2" {
				t.Fatalf("%s = %q, want 1.4.2", key, got[key])
			}
		}
		if got["PORT"] != "3000" || got["HOST"] != "0.0.0.0" || got["PATH"] != "/usr/bin" {
			t.Fatalf("原有 env 处理被破坏: %v", got)
		}
	})

	t.Run("VERSION 已存在时不覆盖", func(t *testing.T) {
		rc := parseRunnerConfig(map[string]interface{}{
			"env": map[string]interface{}{"VERSION": "user-defined"},
		})
		got := toMap(mergeRunnerEnvs([]string{"VERSION=from-image"}, rc, "3000", "1.4.2"))
		if got["VERSION"] != "user-defined" {
			t.Fatalf("VERSION 被覆盖了: %q", got["VERSION"])
		}
		// 保留名仍然以流水线版本为准
		if got["PIPELINE_VERSION"] != "1.4.2" || got["GOPANEL_PIPELINE_VERSION"] != "1.4.2" {
			t.Fatalf("保留名没写对: %v", got)
		}
	})

	t.Run("版本为空时不注入", func(t *testing.T) {
		rc := parseRunnerConfig(nil)
		got := toMap(mergeRunnerEnvs(nil, rc, "3000", "  "))
		for _, key := range []string{"GOPANEL_PIPELINE_VERSION", "PIPELINE_VERSION", "VERSION"} {
			if _, ok := got[key]; ok {
				t.Fatalf("版本为空却注入了 %s", key)
			}
		}
	})
}

func TestResolveRunnerSourceMountDir(t *testing.T) {
	defaultRC := parseRunnerConfig(nil)
	if got := resolveRunnerSourceMountDir(defaultRC, defaultRC.WorkingDir); got != runnerWorkspaceMountPath {
		t.Fatalf("expected default source mount dir %q, got %q", runnerWorkspaceMountPath, got)
	}

	customRC := parseRunnerConfig(map[string]interface{}{
		"workingDir": "/srv/app",
	})
	if got := resolveRunnerSourceMountDir(customRC, customRC.WorkingDir); got != "/srv/app" {
		t.Fatalf("expected custom source mount dir to match workingDir, got %q", got)
	}
}

func TestBuildRunnerScriptSkipsSyncWhenSourceEqualsWorkingDir(t *testing.T) {
	rc := parseRunnerConfig(map[string]interface{}{
		"workingDir": "/srv/app",
	})
	script := buildRunnerScript(rc, "/srv/app")

	if !strings.Contains(script, "source mount already targets working dir, skip sync") {
		t.Fatalf("expected direct-mount script to skip sync, script=%s", script)
	}
	if strings.Contains(script, "syncing source into working dir") {
		t.Fatalf("expected direct-mount script to avoid sync branch, script=%s", script)
	}
}

func TestBuildRunnerScriptBuildsDefaultNodeProject(t *testing.T) {
	script := buildRunnerScript(parseRunnerConfig(nil), runnerWorkspaceMountPath)
	for _, expected := range []string{"package.json detected, rebuilding app", "NPM_CONFIG_INCLUDE=dev", "npm ci --include=dev", "npm run build"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("expected default Runner script to contain %q, script=%s", expected, script)
		}
	}
}

func TestBuildRunnerScriptIncludesBuildDependenciesForCustomCommand(t *testing.T) {
	rc := parseRunnerConfig(map[string]interface{}{
		"buildCommand": "npm ci --legacy-peer-deps && npm run build",
	})
	script := buildRunnerScript(rc, runnerWorkspaceMountPath)
	for _, expected := range []string{"NPM_CONFIG_INCLUDE=dev", "NPM_CONFIG_PRODUCTION=false", "YARN_PRODUCTION=false", rc.BuildCommand} {
		if !strings.Contains(script, expected) {
			t.Fatalf("expected custom Runner script to contain %q, script=%s", expected, script)
		}
	}
}

func TestBuildRunnerScriptKeepsPersistentPathsOutOfCleanup(t *testing.T) {
	rc := parseRunnerConfig(map[string]interface{}{
		"persistentPaths": []interface{}{".data", "storage"},
	})
	script := buildRunnerScript(rc, runnerWorkspaceMountPath)

	if strings.Contains(script, "/var/www/app/.data") {
		t.Fatalf("expected cleanup script to skip persistent .data, script=%s", script)
	}
	if strings.Contains(script, "/var/www/app/storage") {
		t.Fatalf("expected cleanup script to skip persistent storage, script=%s", script)
	}
	if !strings.Contains(script, "/var/www/app/.git") {
		t.Fatalf("expected cleanup script to keep transient dirs, script=%s", script)
	}
}
