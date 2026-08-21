package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/gpc"
)

func fetchPodmanJournalLogsViaGPC(meta podmanContainerLogMeta, since, tail, runtimeHost, afterCursor string) (gpcJournalLogBatch, error) {
	params := map[string]interface{}{"container_id": strings.TrimSpace(meta.ID), "container_name": strings.TrimSpace(meta.Name), "since": since, "tail": tail, "after_cursor": afterCursor, "runtime_host": runtimeHost}
	resp, err := gpc.Do(context.Background(), "PODMAN_CONTAINER_JOURNAL_LOGS", params)
	if err != nil {
		return gpcJournalLogBatch{}, err
	}
	var batch gpcJournalLogBatch
	if err := json.Unmarshal([]byte(resp.Output), &batch); err != nil {
		return gpcJournalLogBatch{}, err
	}
	if batch.Lines == nil {
		batch.Lines = []string{}
	}
	return batch, nil
}
func inspectPodmanContainerLogMeta(containerID string, runtimeHost string) (podmanContainerLogMeta, error) {
	raw, err := inspectPodman(&dto.InspectReq{ID: containerID, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return podmanContainerLogMeta{}, err
	}
	type podmanLogConfig struct {
		Type string `json:"Type"`
	}
	type podmanHostConfig struct {
		LogConfig podmanLogConfig `json:"LogConfig"`
	}
	type podmanInspectItem struct {
		ID         string           `json:"Id"`
		Name       string           `json:"Name"`
		HostConfig podmanHostConfig `json:"HostConfig"`
	}
	parse := func(item podmanInspectItem) podmanContainerLogMeta {
		return podmanContainerLogMeta{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(strings.TrimPrefix(item.Name, "/")), LogDriver: strings.TrimSpace(item.HostConfig.LogConfig.Type)}
	}
	var list []podmanInspectItem
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return parse(list[0]), nil
	}
	var single podmanInspectItem
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse(single), nil
	}
	return podmanContainerLogMeta{}, buserr.New("ErrContainerLogMetadataParse")
}
func buildJournaldContainerLogCommand(ctx context.Context, meta podmanContainerLogMeta, since, tail, runtimeHost string, follow bool) (*exec.Cmd, bool, error) {
	if runtime.GOOS != "linux" {
		return nil, false, nil
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		return nil, false, nil
	}
	candidates := journaldCommandCandidates(meta, since, tail, runtimeHost, follow)
	if len(candidates) == 0 {
		return nil, false, nil
	}
	probeSince := since
	if strings.TrimSpace(probeSince) == "all" {
		probeSince = ""
	}
	permissionDenied := false
	for _, spec := range candidates {
		probeCmd := exec.CommandContext(ctx, spec.command, buildJournaldProbeArgs(spec.args, probeSince)...)
		out, err := probeCmd.CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return exec.CommandContext(ctx, spec.command, spec.args...), true, nil
		}
		if isJournalPermissionError(string(out), err) {
			permissionDenied = true
		}
	}
	if permissionDenied {
		return nil, true, buildJournalPermissionDeniedError(runtimeHost)
	}
	return exec.CommandContext(ctx, candidates[0].command, candidates[0].args...), true, nil
}
func journaldCommandCandidates(meta podmanContainerLogMeta, since, tail, runtimeHost string, follow bool) []journalctlCommandSpec {
	filter := "CONTAINER_ID_FULL=" + firstNonEmpty(strings.TrimSpace(meta.ID), strings.TrimSpace(meta.Name))
	if filter == "CONTAINER_ID_FULL=" {
		return nil
	}
	rootlessUID := rootlessPodmanUID(runtimeHost)
	currentUID := ""
	if currentUser, err := user.Current(); err == nil {
		currentUID = strings.TrimSpace(currentUser.Uid)
	}
	baseArgs := func(includeUser bool) []string {
		args := make([]string, 0, 12)
		if includeUser {
			args = append(args, "--user")
		}
		args = append(args, "--no-pager", "-q", "-o", "short-iso")
		if normalizedSince := normalizeJournalctlSince(since); normalizedSince != "" {
			args = append(args, "--since", normalizedSince)
		}
		if tail != "0" {
			args = append(args, "-n", tail)
		}
		if follow {
			args = append(args, "-f")
		}
		args = append(args, filter)
		return args
	}
	var specs []journalctlCommandSpec
	if rootlessUID != "" && rootlessUID != currentUID {
		if _, err := exec.LookPath("sudo"); err == nil {
			specs = append(specs, journalctlCommandSpec{command: "sudo", args: append([]string{"-n", "-u", "#" + rootlessUID, "journalctl"}, baseArgs(true)...)})
		}
		specs = append(specs, journalctlCommandSpec{command: "journalctl", args: baseArgs(false)})
		specs = append(specs, journalctlCommandSpec{command: "journalctl", args: baseArgs(true)})
		return specs
	}
	specs = append(specs, journalctlCommandSpec{command: "journalctl", args: baseArgs(true)})
	specs = append(specs, journalctlCommandSpec{command: "journalctl", args: baseArgs(false)})
	return specs
}
func buildJournaldProbeArgs(args []string, since string) []string {
	out := make([]string, 0, len(args)+2)
	for i := 0; i < len(args); i++ {
		if args[i] == "-f" {
			continue
		}
		out = append(out, args[i])
	}
	hasTail := false
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "-n" {
			out[i+1] = "1"
			hasTail = true
			break
		}
	}
	if !hasTail {
		filter := out[len(out)-1]
		out = append(out[:len(out)-1], "-n", "1", filter)
	}
	if normalizedSince := normalizeJournalctlSince(since); normalizedSince != "" {
		hasSince := false
		for i := 0; i < len(out)-1; i++ {
			if out[i] == "--since" {
				hasSince = true
				break
			}
		}
		if !hasSince {
			filter := out[len(out)-1]
			out = append(out[:len(out)-1], "--since", normalizedSince, filter)
		}
	}
	return out
}
func normalizeJournalctlSince(since string) string {
	s := strings.TrimSpace(since)
	if s == "" || s == "all" {
		return ""
	}
	if dur, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-dur).Format(time.RFC3339)
	}
	return s
}
func rootlessPodmanUID(runtimeHost string) string {
	host := strings.TrimSpace(runtimeHost)
	if !strings.HasPrefix(host, "unix:///run/user/") {
		return ""
	}
	rest := strings.TrimPrefix(host, "unix:///run/user/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func isJournalPermissionError(output string, err error) bool {
	msg := strings.ToLower(strings.TrimSpace(output))
	if msg == "" && err != nil {
		msg = strings.ToLower(strings.TrimSpace(err.Error()))
	}
	return strings.Contains(msg, "insufficient permissions") || strings.Contains(msg, "no journal files were opened") || strings.Contains(msg, "you are currently not seeing messages from the system") || strings.Contains(msg, "failed to open journals") || strings.Contains(msg, "sudo:") || strings.Contains(msg, "a password is required")
}
func buildJournalPermissionDeniedError(runtimeHost string) error {
	if uid := rootlessPodmanUID(runtimeHost); uid != "" {
		return buserr.WithMap(constant.ErrContainerJournalUserPermission, map[string]interface{}{"uid": uid})
	}
	return buserr.New(constant.ErrContainerJournalPermission)
}
func podmanContainerLogPath(ctx context.Context, containerName string, runtimeHost string) (string, error) {
	_ = ctx
	raw, err := inspectPodman(&dto.InspectReq{ID: containerName, Type: "container", RuntimeHost: runtimeHost})
	if err != nil {
		return "", err
	}
	type podmanLogConfig struct {
		Type string `json:"Type"`
		Path string `json:"Path"`
	}
	type podmanHostConfig struct {
		LogPath   string          `json:"LogPath"`
		LogConfig podmanLogConfig `json:"LogConfig"`
	}
	type podmanContainerInspect struct {
		LogPath    string           `json:"LogPath"`
		HostConfig podmanHostConfig `json:"HostConfig"`
	}
	resolve := func(item podmanContainerInspect) (string, error) {
		if path := strings.TrimSpace(item.LogPath); path != "" {
			return path, nil
		}
		if path := strings.TrimSpace(item.HostConfig.LogPath); path != "" {
			return path, nil
		}
		if path := strings.TrimSpace(item.HostConfig.LogConfig.Path); path != "" {
			return path, nil
		}
		if strings.EqualFold(strings.TrimSpace(item.HostConfig.LogConfig.Type), "journald") {
			return "", buserr.New("ErrContainerLogJournaldDriver")
		}
		return "", buserr.New(constant.ErrContainerLogPathEmpty)
	}
	var list []podmanContainerInspect
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return resolve(list[0])
	}
	var single podmanContainerInspect
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return resolve(single)
	}
	return "", buserr.New("ErrContainerLogPathParse")
}
func truncateContainerLogFiles(logPath string) error {
	file, err := os.OpenFile(logPath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	if err = file.Truncate(0); err != nil {
		return err
	}
	files, _ := filepath.Glob(fmt.Sprintf("%s.*", logPath))
	for _, file := range files {
		_ = os.Remove(file)
	}
	return nil
}
