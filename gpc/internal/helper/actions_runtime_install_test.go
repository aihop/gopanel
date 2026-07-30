//go:build !windows

package helper

import "testing"

func TestValidateRuntimeInstallKind(t *testing.T) {
	for _, runtimeKind := range []string{"docker", "podman", " Docker "} {
		if _, err := validateRuntimeInstallKind(map[string]interface{}{"runtime": runtimeKind}); err != nil {
			t.Fatalf("runtime %q should be accepted: %v", runtimeKind, err)
		}
	}
	for _, runtimeKind := range []string{"", "containerd", "docker; reboot"} {
		if _, err := validateRuntimeInstallKind(map[string]interface{}{"runtime": runtimeKind}); err == nil {
			t.Fatalf("runtime %q should be rejected", runtimeKind)
		}
	}
}

func TestRuntimeInstallHasDedicatedLock(t *testing.T) {
	if lockKey := lockKeyForAction("CONTAINER_RUNTIME_INSTALL"); lockKey != "container_runtime_install" {
		t.Fatalf("unexpected lock key: %q", lockKey)
	}
}
