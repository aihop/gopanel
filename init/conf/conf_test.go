package conf

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNormalizeGpcSocketPathMigratesMacOSLegacyDefaults(t *testing.T) {
	baseDir := "/Users/test/.gopanel"
	for _, legacyPath := range []string{
		"/run/gopanel/gpc.sock",
		"/var/run/gopanel/gpc.sock",
		"/private/var/run/gopanel/gpc.sock",
		"/opt/gopanel/gpc.sock",
		"/opt/gopanel/runtime/gpc.sock",
	} {
		if got := normalizeGpcSocketPath("darwin", baseDir, legacyPath); got != baseDir+"/gpc.sock" {
			t.Fatalf("normalizeGpcSocketPath(%q) = %q", legacyPath, got)
		}
	}
}

func TestMigrateGpcSocketConfigUpdatesPersistedValue(t *testing.T) {
	config := viper.New()
	config.Set("system.base_dir", "/Users/test/.gopanel")
	config.Set("system.gpc_socket_path", "/run/gopanel/gpc.sock")

	if !migrateGpcSocketConfig(config, "darwin") {
		t.Fatal("legacy macOS socket path was not migrated")
	}
	if got := config.GetString("system.gpc_socket_path"); got != "/Users/test/.gopanel/gpc.sock" {
		t.Fatalf("persisted socket path = %q", got)
	}
	if migrateGpcSocketConfig(config, "darwin") {
		t.Fatal("already migrated socket path changed again")
	}
}

func TestNormalizeGpcSocketPathPreservesCustomAndLinuxPaths(t *testing.T) {
	const customPath = "/custom/run/gpc.sock"
	if got := normalizeGpcSocketPath("darwin", "/Users/test/.gopanel", customPath); got != customPath {
		t.Fatalf("custom macOS path changed to %q", got)
	}
	if got := normalizeGpcSocketPath("linux", "/opt/gopanel", "/run/gopanel/gpc.sock"); got != "/run/gopanel/gpc.sock" {
		t.Fatalf("linux path changed to %q", got)
	}
}

func TestRebaseInstanceSocketPathFollowsRuntimeBaseDir(t *testing.T) {
	configuredBaseDir := "/Users/test/.gopanel"
	runtimeBaseDir := "/Users/test/.gopanel-dev"

	if got := rebaseInstanceSocketPath(configuredBaseDir, runtimeBaseDir, configuredBaseDir+"/gpc.sock"); got != runtimeBaseDir+"/gpc.sock" {
		t.Fatalf("gpc socket path was not rebased: %q", got)
	}
	if got := rebaseInstanceSocketPath(configuredBaseDir, runtimeBaseDir, configuredBaseDir+"/agent/run/gp-agent.sock"); got != runtimeBaseDir+"/agent/run/gp-agent.sock" {
		t.Fatalf("gp-agent socket path was not rebased: %q", got)
	}
}

func TestRebaseInstanceSocketPathPreservesCustomPath(t *testing.T) {
	const customPath = "/custom/run/gpc.sock"
	if got := rebaseInstanceSocketPath("/Users/test/.gopanel", "/Users/test/.gopanel-dev", customPath); got != customPath {
		t.Fatalf("custom socket path changed to %q", got)
	}
}
