package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func (u *ContainerService) ContainerLogClean(req *dto.OperationWithName) error {
	if cmd.CheckIllegal(req.Name) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	ctx := context.Background()
	if docker.IsPodmanRuntime(ctx) {
		if runtime.GOOS == "darwin" {
			return errors.New("podman on darwin does not support cleaning container log files (logs are stored inside podman machine); please restart/recreate the container to clear logs")
		}
		host := strings.TrimSpace(req.RuntimeHost)
		if host == "" {
			host, _ = resolveLinuxPodmanContainerHost(ctx, req.Name)
		}
		logPath, err := podmanContainerLogPath(ctx, req.Name, host)
		if err != nil {
			return err
		}
		if logPath == "" {
			return errors.New("container log path is empty")
		}
		return truncateContainerLogFiles(logPath)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	containerItem, err := client.ContainerInspect(ctx, req.Name)
	if err != nil {
		return err
	}
	if containerItem.LogPath == "" {
		return errors.New("container log path is empty")
	}
	return truncateContainerLogFiles(containerItem.LogPath)
}

func (u *ContainerService) ContainerLogs(wsConn *websocket.Conn, containerType, containerID, since, tail, runtimeHost string, follow bool) error {
	defer func() { wsConn.Close() }()
	if cmd.CheckIllegal(containerID, since, tail) {
		return buserr.New(constant.ErrCmdIllegal)
	}
	if containerType == "container" {
		if handled, err := u.streamContainerLogsViaGPC(wsConn, containerID, since, tail, runtimeHost, follow); handled {
			return err
		}
	}
	cmdExec, err := buildContainerLogCommand(context.Background(), containerType, containerID, since, tail, runtimeHost, follow)
	if err != nil {
		return err
	}
	if !follow {
		cmdExec.Stderr = cmdExec.Stdout
		stdout, _ := cmdExec.CombinedOutput()
		stdout = bytes.ToValidUTF8(stdout, []byte("?"))
		if err := wsConn.WriteMessage(websocket.TextMessage, stdout); err != nil {
			global.LOG.Errorf("send message with log to ws failed, err: %v", err)
		}
		return nil
	}

	stdout, err := cmdExec.StdoutPipe()
	if err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return err
	}
	cmdExec.Stderr = cmdExec.Stdout
	if err := cmdExec.Start(); err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return err
	}
	exitCh := make(chan struct{})
	go func() {
		_, wsData, _ := wsConn.ReadMessage()
		if string(wsData) == "close conn" {
			_ = cmdExec.Process.Signal(syscall.SIGTERM)
			exitCh <- struct{}{}
		}
	}()

	go func() {
		buffer := make([]byte, 1024)
		pending := make([]byte, 0, utf8.UTFMax)
		for {
			select {
			case <-exitCh:
				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err == io.EOF {
						if len(pending) > 0 {
							finalChunk := bytes.ToValidUTF8(pending, []byte("?"))
							if len(finalChunk) > 0 {
								_ = wsConn.WriteMessage(websocket.TextMessage, finalChunk)
							}
						}
						return
					}
					global.LOG.Errorf("read bytes from log failed, err: %v", err)
					return
				}
				chunk := append(pending, buffer[:n]...)
				safeChunk, rest := splitUTF8LogChunk(chunk)
				pending = pending[:0]
				if len(rest) > 0 {
					pending = append(pending, rest...)
				}
				if len(safeChunk) == 0 {
					continue
				}
				if err = wsConn.WriteMessage(websocket.TextMessage, safeChunk); err != nil {
					global.LOG.Errorf("send message with log to ws failed, err: %v", err)
					return
				}
			}
		}
	}()
	_ = cmdExec.Wait()
	return nil
}

func splitUTF8LogChunk(chunk []byte) ([]byte, []byte) {
	if len(chunk) == 0 {
		return nil, nil
	}
	if utf8.Valid(chunk) {
		return chunk, nil
	}
	maxTail := utf8.UTFMax - 1
	if maxTail > len(chunk) {
		maxTail = len(chunk)
	}
	for tail := 1; tail <= maxTail; tail++ {
		prefix := chunk[:len(chunk)-tail]
		suffix := chunk[len(chunk)-tail:]
		if utf8.Valid(prefix) && !utf8.FullRune(suffix) {
			return prefix, suffix
		}
	}
	return bytes.ToValidUTF8(chunk, []byte("?")), nil
}

func (u *ContainerService) DownloadContainerLogs(containerType, containerID, since, tail, runtimeHost string) (string, error) {
	if cmd.CheckIllegal(containerID, since, tail) {
		return "", buserr.New(constant.ErrCmdIllegal)
	}
	if containerType == "container" {
		if filePath, handled, err := u.downloadContainerLogsViaGPC(containerID, since, tail, runtimeHost); handled {
			return filePath, err
		}
	}
	cmdExec, err := buildContainerLogCommand(context.Background(), containerType, containerID, since, tail, runtimeHost, false)
	if err != nil {
		return "", err
	}

	stdout, err := cmdExec.StdoutPipe()
	if err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return "", err
	}
	cmdExec.Stderr = cmdExec.Stdout
	if err := cmdExec.Start(); err != nil {
		_ = cmdExec.Process.Signal(syscall.SIGTERM)
		return "", err
	}

	tempFile, err := os.CreateTemp("", "cmd_output_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	errCh := make(chan error)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if _, err := tempFile.WriteString(line + "\n"); err != nil {
				errCh <- err
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case err := <-errCh:
		if err != nil {
			global.LOG.Errorf("Error: %v", err)
		}
	case <-time.After(3 * time.Second):
		global.LOG.Errorf("Timeout reached")
	}
	return tempFile.Name(), nil
}

func buildContainerLogCommand(ctx context.Context, containerType, containerID, since, tail, runtimeHost string, follow bool) (*exec.Cmd, error) {
	if containerType == "compose" {
		commandArg := []string{"-f", containerID, "logs"}
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		if follow {
			commandArg = append(commandArg, "-f")
		}
		return compose.Command(ctx, commandArg...)
	}

	isPodman := docker.IsPodmanRuntime(ctx)
	host := strings.TrimSpace(runtimeHost)
	if host == "" && isPodman {
		host, _ = resolveLinuxPodmanContainerHost(ctx, containerID)
	}
	if isPodman {
		if cmdExec, err := buildPodmanContainerLogCommand(ctx, containerID, since, tail, host, follow); err == nil && cmdExec != nil {
			return cmdExec, nil
		}
	}

	commandArg := make([]string, 0, 12)
	commandArg = append(commandArg, "logs")
	if tail != "0" {
		commandArg = append(commandArg, "--tail", tail)
	}
	if since != "all" {
		commandArg = append(commandArg, "--since", since)
	}
	if follow {
		commandArg = append(commandArg, "-f")
	}
	commandArg = append(commandArg, containerID)
	return docker.RuntimeCommandWithHost(ctx, host, commandArg...)
}

type podmanContainerLogMeta struct {
	ID        string
	Name      string
	LogDriver string
}

type gpcJournalLogBatch struct {
	Lines  []string `json:"lines"`
	Cursor string   `json:"cursor"`
}

func buildPodmanContainerLogCommand(ctx context.Context, containerID, since, tail, runtimeHost string, follow bool) (*exec.Cmd, error) {
	meta, err := inspectPodmanContainerLogMeta(containerID, runtimeHost)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(meta.LogDriver, "journald") {
		commandArg := make([]string, 0, 12)
		commandArg = append(commandArg, "logs")
		if tail != "0" {
			commandArg = append(commandArg, "--tail", tail)
		}
		if since != "all" {
			commandArg = append(commandArg, "--since", since)
		}
		if follow {
			commandArg = append(commandArg, "-f")
		}
		commandArg = append(commandArg, containerID)
		return docker.RuntimeCommandWithHost(ctx, runtimeHost, commandArg...)
	}

	cmdExec, ok, err := buildJournaldContainerLogCommand(ctx, meta, since, tail, runtimeHost, follow)
	if ok || err != nil {
		return cmdExec, err
	}
	commandArg := make([]string, 0, 12)
	commandArg = append(commandArg, "logs")
	if tail != "0" {
		commandArg = append(commandArg, "--tail", tail)
	}
	if since != "all" {
		commandArg = append(commandArg, "--since", since)
	}
	if follow {
		commandArg = append(commandArg, "-f")
	}
	commandArg = append(commandArg, containerID)
	return docker.RuntimeCommandWithHost(ctx, runtimeHost, commandArg...)
}

func (u *ContainerService) streamContainerLogsViaGPC(wsConn *websocket.Conn, containerID, since, tail, runtimeHost string, follow bool) (bool, error) {
	ctx := context.Background()
	if !docker.IsPodmanRuntime(ctx) {
		return false, nil
	}
	meta, err := inspectPodmanContainerLogMeta(containerID, runtimeHost)
	if err != nil {
		return false, nil
	}
	if !strings.EqualFold(meta.LogDriver, "journald") {
		return false, nil
	}
	if !follow {
		batch, err := fetchPodmanJournalLogsViaGPC(meta, since, tail, runtimeHost, "")
		if err != nil {
			return true, err
		}
		if len(batch.Lines) > 0 {
			payload := bytes.ToValidUTF8([]byte(strings.Join(batch.Lines, "\n")), []byte("?"))
			if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
				global.LOG.Errorf("send gpc journal logs to ws failed, err: %v", err)
			}
		}
		return true, nil
	}

	stopCh := make(chan struct{})
	go func() {
		defer close(stopCh)
		_, wsData, _ := wsConn.ReadMessage()
		if string(wsData) == "close conn" {
			return
		}
	}()

	cursor := ""
	firstRound := true
	for {
		select {
		case <-stopCh:
			return true, nil
		default:
		}

		currentTail := tail
		currentSince := since
		if !firstRound {
			currentTail = "0"
			currentSince = "all"
		}
		batch, err := fetchPodmanJournalLogsViaGPC(meta, currentSince, currentTail, runtimeHost, cursor)
		if err != nil {
			return true, err
		}
		if batch.Cursor != "" {
			cursor = batch.Cursor
		}
		for _, line := range batch.Lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			payload := bytes.ToValidUTF8([]byte(line), []byte("?"))
			if err := wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
				global.LOG.Errorf("send gpc journal log line to ws failed, err: %v", err)
				return true, nil
			}
		}
		firstRound = false

		select {
		case <-stopCh:
			return true, nil
		case <-time.After(1500 * time.Millisecond):
		}
	}
}

func (u *ContainerService) downloadContainerLogsViaGPC(containerID, since, tail, runtimeHost string) (string, bool, error) {
	ctx := context.Background()
	if !docker.IsPodmanRuntime(ctx) {
		return "", false, nil
	}
	meta, err := inspectPodmanContainerLogMeta(containerID, runtimeHost)
	if err != nil {
		return "", false, nil
	}
	if !strings.EqualFold(meta.LogDriver, "journald") {
		return "", false, nil
	}
	batch, err := fetchPodmanJournalLogsViaGPC(meta, since, tail, runtimeHost, "")
	if err != nil {
		return "", true, err
	}
	tempFile, err := os.CreateTemp("", "container_journal_*.txt")
	if err != nil {
		return "", true, err
	}
	defer tempFile.Close()
	if len(batch.Lines) > 0 {
		if _, err := tempFile.WriteString(strings.Join(batch.Lines, "\n") + "\n"); err != nil {
			return "", true, err
		}
	}
	return tempFile.Name(), true, nil
}

func fetchPodmanJournalLogsViaGPC(meta podmanContainerLogMeta, since, tail, runtimeHost, afterCursor string) (gpcJournalLogBatch, error) {
	params := map[string]interface{}{
		"container_id":   strings.TrimSpace(meta.ID),
		"container_name": strings.TrimSpace(meta.Name),
		"since":          since,
		"tail":           tail,
		"after_cursor":   afterCursor,
		"runtime_host":   runtimeHost,
	}
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
		return podmanContainerLogMeta{
			ID:        strings.TrimSpace(item.ID),
			Name:      strings.TrimSpace(strings.TrimPrefix(item.Name, "/")),
			LogDriver: strings.TrimSpace(item.HostConfig.LogConfig.Type),
		}
	}
	var list []podmanInspectItem
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return parse(list[0]), nil
	}
	var single podmanInspectItem
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return parse(single), nil
	}
	return podmanContainerLogMeta{}, errors.New("failed to parse podman inspect output for log metadata")
}

type journalctlCommandSpec struct {
	command string
	args    []string
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
			specs = append(specs, journalctlCommandSpec{
				command: "sudo",
				args:    append([]string{"-n", "-u", "#" + rootlessUID, "journalctl"}, baseArgs(true)...),
			})
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
	return strings.Contains(msg, "insufficient permissions") ||
		strings.Contains(msg, "no journal files were opened") ||
		strings.Contains(msg, "you are currently not seeing messages from the system") ||
		strings.Contains(msg, "failed to open journals") ||
		strings.Contains(msg, "sudo:") ||
		strings.Contains(msg, "a password is required")
}

func buildJournalPermissionDeniedError(runtimeHost string) error {
	if uid := rootlessPodmanUID(runtimeHost); uid != "" {
		return fmt.Errorf("当前 GoPanel 进程无权读取 rootless Podman 用户 %s 的 journald 日志；请让面板进程具备 `sudo -n -u '#%s' journalctl --user` 权限，或将面板用户加入 `adm/systemd-journal`，否则建议继续使用 k8s-file 日志驱动", uid, uid)
	}
	return errors.New("当前 GoPanel 进程无权读取 journald 日志；请将面板用户加入 `adm/systemd-journal`，或改用 k8s-file 日志驱动")
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
			return "", errors.New("podman container is using journald log driver; log files cannot be truncated directly, please restart/recreate the container to clear logs")
		}
		return "", errors.New("container log path is empty")
	}

	var list []podmanContainerInspect
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		return resolve(list[0])
	}
	var single podmanContainerInspect
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return resolve(single)
	}
	return "", errors.New("failed to parse podman inspect output for log path")
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

func (u *ContainerService) LoadContainerLogs(req *dto.OperationWithNameAndType) string {
	filePath := ""
	if req.Type == "compose-detail" {
		ctx := context.Background()
		options := container.ListOptions{All: true}
		isPodman := docker.IsPodmanRuntime(ctx)
		if !isPodman {
			options.Filters = filters.NewArgs()
			options.Filters.Add("label", fmt.Sprintf("%s=%s", composeProjectLabel, req.Name))
		}
		var (
			containers []types.Container
			err        error
		)
		if isPodman {
			containers, err = docker.ListContainersMerged(ctx, options)
		} else {
			cli, err := docker.NewDockerClient()
			if err != nil {
				return ""
			}
			defer cli.Close()
			containers, err = cli.ContainerList(ctx, options)
		}
		if err != nil {
			return ""
		}
		for _, container := range containers {
			if isPodman {
				name, ok := firstLabel(container.Labels, composeProjectLabel, podmanComposeProjectLabel)
				if !ok || name != req.Name {
					continue
				}
			}
			config, _ := firstLabel(container.Labels, composeConfigLabel, podmanComposeConfigLabel)
			workdir, _ := firstLabel(container.Labels, composeWorkdirLabel, podmanComposeWorkdirLabel)
			if len(config) != 0 && len(workdir) != 0 && strings.Contains(config, workdir) {
				filePath = config
				break
			}
			filePath = workdir
			break
		}
		if len(containers) == 0 {
			composeItem, _ := repo.NewIComposeTemplateRepo().GetRecord(repo.NewCommonRepo().WithByName(req.Name))
			filePath = composeItem.Path
		}
	}
	if _, err := os.Stat(filePath); err != nil {
		return ""
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(content)
}
