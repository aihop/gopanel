package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCodeExecutorCommandFindsNVMInstall(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", "/usr/bin:/bin")
	binDir := filepath.Join(homeDir, ".nvm", "versions", "node", "v22.0.0", "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	commandPath := filepath.Join(binDir, "gopanel-test-codex")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}

	resolvedPath, env, err := resolveCodeExecutorCommand("gopanel-test-codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolvedPath != commandPath {
		t.Fatalf("resolved path = %q, want %q", resolvedPath, commandPath)
	}
	pathValue := environmentValue(env, "PATH")
	if !strings.HasPrefix(pathValue, binDir+string(os.PathListSeparator)) {
		t.Fatalf("executor PATH should start with %q: %q", binDir, pathValue)
	}
}

func TestResolveCodeExecutorRunsNPMShimWithManagedNode(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", "/usr/bin:/bin")
	binDir := filepath.Join(homeDir, ".nvm", "versions", "node", "v22.0.0", "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "node"), []byte("#!/bin/sh\necho codex-cli 1.2.3\n"), 0700); err != nil {
		t.Fatal(err)
	}
	commandName := "gopanel-test-npm-codex"
	if err := os.WriteFile(filepath.Join(binDir, commandName), []byte("#!/usr/bin/env node\n"), 0700); err != nil {
		t.Fatal(err)
	}

	commandPath, env, err := resolveCodeExecutorCommand(commandName)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(commandPath, "--version")
	command.Env = env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run npm Codex shim: %v: %s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != "codex-cli 1.2.3" {
		t.Fatalf("npm Codex shim output = %q", got)
	}
}

func TestCodeExecutorSearchDirsIncludePackageManagerLocations(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", "/usr/bin:/bin")
	dirs := codeExecutorSearchDirs()
	for _, expected := range []string{
		filepath.Join(homeDir, ".npm-global", "bin"),
		filepath.Join(homeDir, ".volta", "bin"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/home/linuxbrew/.linuxbrew/bin",
	} {
		if !containsString(dirs, expected) {
			t.Fatalf("missing executor search directory %q: %#v", expected, dirs)
		}
	}
}

func environmentValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
