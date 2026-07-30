package service

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/aihop/gopanel/utils/gpc"
)

const (
	ControlPlaneHealthy          = "healthy"
	ControlPlaneMissing          = "missing"
	ControlPlaneServiceStopped   = "service_stopped"
	ControlPlaneSocketMissing    = "socket_missing"
	ControlPlanePermissionDenied = "permission_denied"
	ControlPlaneConfigMismatch   = "config_mismatch"
)

type ControlPlaneComponentStatus struct {
	Name       string   `json:"name"`
	State      string   `json:"state"`
	Healthy    bool     `json:"healthy"`
	Installed  bool     `json:"installed"`
	Reachable  bool     `json:"reachable"`
	SocketPath string   `json:"socketPath,omitempty"`
	Version    string   `json:"version,omitempty"`
	Error      string   `json:"error,omitempty"`
	Commands   []string `json:"commands,omitempty"`
}

type ControlPlaneStatus struct {
	Healthy        bool                        `json:"healthy"`
	AutoRepairable bool                        `json:"autoRepairable"`
	RequiresSudo   bool                        `json:"requiresSudo"`
	NextAction     string                      `json:"nextAction"`
	CheckedAt      int64                       `json:"checkedAt"`
	GPC            ControlPlaneComponentStatus `json:"gpc"`
	Agent          ControlPlaneComponentStatus `json:"agent"`
}

type controlPlaneAgentInfo struct {
	Version string `json:"version"`
}

// DiagnoseControlPlane returns both the privileged helper and host agent state.
// GPC is always dialed even when the panel runs as root because AgentEnsure uses it.
func DiagnoseControlPlane(ctx context.Context) ControlPlaneStatus {
	if ctx == nil {
		ctx = context.Background()
	}
	type componentResult struct {
		name   string
		status ControlPlaneComponentStatus
	}
	results := make(chan componentResult, 2)
	go func() { results <- componentResult{name: "gpc", status: diagnoseGpcComponent(ctx)} }()
	go func() { results <- componentResult{name: "agent", status: diagnoseAgentComponent(ctx)} }()
	var gpcStatus, agentStatus ControlPlaneComponentStatus
	for range 2 {
		result := <-results
		if result.name == "gpc" {
			gpcStatus = result.status
		} else {
			agentStatus = result.status
		}
	}
	status := ControlPlaneStatus{
		CheckedAt: time.Now().UnixMilli(),
		GPC:       gpcStatus,
		Agent:     agentStatus,
	}
	status.Healthy = status.GPC.Healthy && status.Agent.Healthy
	status.AutoRepairable = status.GPC.Healthy && !status.Agent.Healthy
	status.RequiresSudo = !status.GPC.Healthy && len(status.GPC.Commands) > 0
	switch {
	case status.Healthy:
		status.NextAction = "none"
	case status.AutoRepairable:
		status.NextAction = "repair_agent"
	default:
		status.NextAction = "repair_gpc"
	}
	return status
}

func diagnoseGpcComponent(ctx context.Context) ControlPlaneComponentStatus {
	socketPath := gpc.SocketPath()
	installed := fileExists(gpcBinaryPath)
	dialErr := dialUnixSocket(ctx, socketPath)
	state := classifyControlPlaneState(installed, socketPath, dialErr)

	expectedSocket := expectedGpcSocketPath()
	if dialErr != nil && expectedSocket != "" && filepath.Clean(socketPath) != filepath.Clean(expectedSocket) {
		if expectedErr := dialUnixSocket(ctx, expectedSocket); expectedErr == nil {
			state = ControlPlaneConfigMismatch
		}
	}

	component := ControlPlaneComponentStatus{
		Name:       "gpc",
		State:      state,
		Healthy:    state == ControlPlaneHealthy,
		Installed:  installed,
		Reachable:  dialErr == nil && state == ControlPlaneHealthy,
		SocketPath: socketPath,
	}
	if dialErr != nil {
		component.Error = dialErr.Error()
		component.Commands = gpcRecoveryCommands(runtime.GOOS, state, fileExists(gpcDarwinPlist))
	}
	return component
}

func diagnoseAgentComponent(ctx context.Context) ControlPlaneComponentStatus {
	socketPath := gpagent.SocketPath()
	baseDir := strings.TrimSpace(global.CONF.System.BaseDir)
	installed := fileExists(filepath.Join(baseDir, "gp-agent"))
	component := ControlPlaneComponentStatus{
		Name:       "gp-agent",
		Installed:  installed,
		SocketPath: socketPath,
	}
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	response, err := gpagent.Do(requestCtx, "AGENT_STATUS", nil)
	component.State = classifyControlPlaneState(installed, socketPath, err)
	component.Healthy = component.State == ControlPlaneHealthy
	component.Reachable = component.Healthy
	if err != nil {
		component.Error = err.Error()
		return component
	}
	info, decodeErr := gpagent.DecodeOutput[controlPlaneAgentInfo](response)
	if decodeErr != nil {
		component.State = ControlPlaneServiceStopped
		component.Healthy = false
		component.Reachable = true
		component.Error = decodeErr.Error()
		return component
	}
	component.Version = info.Version
	return component
}

func classifyControlPlaneState(installed bool, socketPath string, dialErr error) string {
	if dialErr == nil {
		return ControlPlaneHealthy
	}
	if !installed {
		return ControlPlaneMissing
	}
	if errors.Is(dialErr, os.ErrPermission) || strings.Contains(strings.ToLower(dialErr.Error()), "permission denied") {
		return ControlPlanePermissionDenied
	}
	if _, err := os.Stat(socketPath); errors.Is(err, os.ErrNotExist) {
		return ControlPlaneSocketMissing
	}
	return ControlPlaneServiceStopped
}

func dialUnixSocket(ctx context.Context, socketPath string) error {
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(requestCtx, "unix", socketPath)
	if err == nil {
		_ = conn.Close()
	}
	return err
}

func expectedGpcSocketPath() string {
	baseDir := strings.TrimSpace(global.CONF.System.BaseDir)
	if baseDir == "" {
		return ""
	}
	return filepath.Join(filepath.Clean(baseDir), "gpc.sock")
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func gpcRecoveryCommands(goos, state string, darwinPlistExists bool) []string {
	if state == ControlPlaneConfigMismatch {
		return []string{"bash <(curl -fsSL https://gopanel.run)"}
	}
	if state == ControlPlaneMissing || (goos == "darwin" && !darwinPlistExists) {
		return []string{"bash <(curl -fsSL https://gopanel.run)"}
	}
	if goos == "darwin" {
		return []string{
			"sudo /usr/libexec/PlistBuddy -c 'Delete :UserName' " + gpcDarwinPlist + " 2>/dev/null; sudo /usr/libexec/PlistBuddy -c 'Add :UserName string root' " + gpcDarwinPlist + " && sudo chown root:wheel " + gpcDarwinPlist + " && sudo chmod 644 " + gpcDarwinPlist + " && sudo launchctl bootout system " + gpcDarwinPlist + " 2>/dev/null; sudo launchctl bootstrap system " + gpcDarwinPlist,
		}
	}
	return []string{"sudo systemctl enable --now gpc.service && sudo systemctl restart gpc.service"}
}
