package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
)

func TestBuildWebsiteImageRuntimeSpecKeepsWebsiteDeploymentMinimal(t *testing.T) {
	inspect := image.InspectResponse{Config: &container.Config{
		Env:        []string{"PORT=8080", "APP_ENV=production"},
		WorkingDir: "/srv/app",
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
	}}

	request := websiteImageDeployRequest{Alias: " demo ", PreviousContainerID: " previous "}
	spec := buildWebsiteImageRuntimeSpec(request, inspect)

	if spec.Alias != "demo" || spec.ContainerPort != "8080" || spec.WorkingDir != "/srv/app" {
		t.Fatalf("unexpected website runtime spec: %+v", spec)
	}
	if spec.PublishedHostPort != "0" || spec.PreviousContainerID != "previous" {
		t.Fatalf("unexpected website port switch policy: %+v", spec)
	}
	if len(spec.Cmd) != 0 || len(spec.Binds) != 0 || len(spec.Mounts) != 0 || spec.NetworkMode != "" || len(spec.ExtraNetworks) != 0 {
		t.Fatalf("website spec contains runner-only settings: %+v", spec)
	}
	if len(spec.Env) != 2 || spec.Env[1] != "APP_ENV=production" {
		t.Fatalf("image environment was not preserved: %v", spec.Env)
	}
}

func TestBuildPipelineRunnerRuntimeSpecOwnsRunnerSettings(t *testing.T) {
	codeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(codeRoot, "package.json"), []byte(`{"scripts":{"start":"node server.js"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	rc := parseRunnerConfig(map[string]interface{}{
		"hostPort":      "3101",
		"runnerUser":    "1000:1000",
		"extraNetworks": []interface{}{runnerNetworkName, "backend"},
	})
	inspect := image.InspectResponse{Config: &container.Config{Env: []string{"PATH=/usr/bin"}}}

	request := pipelineRunnerDeployRequest{
		Alias:               " pipeline-demo ",
		CodeRoot:            codeRoot,
		PipelineKey:         "demo",
		PipelineVersion:     "1.2.3",
		PreviousContainerID: " old ",
	}
	spec, err := buildPipelineRunnerRuntimeSpec(request, rc, inspect)
	if err != nil {
		t.Fatal(err)
	}

	if spec.Alias != "pipeline-demo" || spec.ContainerPort != "3000" || spec.PublishedHostPort != "3101" {
		t.Fatalf("unexpected runner identity or ports: %+v", spec)
	}
	if spec.NetworkMode != container.NetworkMode(runnerNetworkName) || len(spec.ExtraNetworks) != 1 || spec.ExtraNetworks[0] != "backend" {
		t.Fatalf("unexpected runner networks: mode=%q extra=%v", spec.NetworkMode, spec.ExtraNetworks)
	}
	if len(spec.Binds) != 1 || spec.Binds[0] != codeRoot+":"+runnerWorkspaceMountPath+":ro" {
		t.Fatalf("unexpected runner source bind: %v", spec.Binds)
	}
	if len(spec.Mounts) != 1 || spec.User != "1000:1000" || spec.PreviousContainerID != "old" {
		t.Fatalf("runner isolation or switch settings missing: %+v", spec)
	}
	if len(spec.Cmd) != 3 || spec.Cmd[0] != "sh" || !strings.Contains(spec.Cmd[2], "node .output/server/index.mjs") {
		t.Fatalf("runner startup command missing: %v", spec.Cmd)
	}
	if !spec.WaitForReady {
		t.Fatal("Runner must keep restart disabled until HTTP readiness succeeds")
	}
	envs := envSliceToMap(spec.Env)
	if envs["PATH"] != "/usr/bin" || envs["PORT"] != "3000" || envs["PIPELINE_VERSION"] != "1.2.3" {
		t.Fatalf("unexpected runner environment: %v", envs)
	}
}

func envSliceToMap(envs []string) map[string]string {
	result := make(map[string]string, len(envs))
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func TestRuntimeRestartPolicyChangesAfterReadiness(t *testing.T) {
	if got := initialRuntimeRestartPolicy(true).Name; got != "no" {
		t.Fatalf("initial Runner restart policy = %q, want no", got)
	}
	if got := initialRuntimeRestartPolicy(false).Name; got != "always" {
		t.Fatalf("regular runtime restart policy = %q, want always", got)
	}
	if got := readyRuntimeUpdateConfig().RestartPolicy.Name; got != "always" {
		t.Fatalf("ready Runner restart policy = %q, want always", got)
	}
}

func TestRunnerReadyTimeoutMatchesMode(t *testing.T) {
	buildRun := resolveRunnerReadyTimeout(parseRunnerConfig(map[string]interface{}{"mode": "build_run"}))
	if buildRun != runnerBuildReadyTimeout {
		t.Fatalf("build_run ready timeout = %s, want %s", buildRun, runnerBuildReadyTimeout)
	}
	startOnly := resolveRunnerReadyTimeout(parseRunnerConfig(map[string]interface{}{"mode": "start"}))
	if startOnly != runnerStartReadyTimeout {
		t.Fatalf("start-only ready timeout = %s, want %s", startOnly, runnerStartReadyTimeout)
	}
	if buildRun <= startOnly {
		t.Fatal("build_run must get a larger budget than start-only")
	}
}

func TestResolveRuntimeReadyTimeoutPrecedence(t *testing.T) {
	if got := resolveRuntimeReadyTimeout(0); got != defaultRuntimeReadyTimeout {
		t.Fatalf("zero spec timeout = %s, want the default %s", got, defaultRuntimeReadyTimeout)
	}
	if got := resolveRuntimeReadyTimeout(30 * time.Minute); got != 30*time.Minute {
		t.Fatalf("spec timeout ignored, got %s", got)
	}
	t.Setenv(runtimeReadyTimeoutEnv, "90m")
	if got := resolveRuntimeReadyTimeout(30 * time.Minute); got != 90*time.Minute {
		t.Fatalf("env override must win over the spec, got %s", got)
	}
	// 写错的值不能把预算悄悄清零，否则 Runner 一启动就判超时。
	t.Setenv(runtimeReadyTimeoutEnv, "not-a-duration")
	if got := resolveRuntimeReadyTimeout(30 * time.Minute); got != 30*time.Minute {
		t.Fatalf("invalid env must fall back to the spec, got %s", got)
	}
}
