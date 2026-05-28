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
