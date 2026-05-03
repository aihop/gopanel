package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
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
	defer func() {
		wsConn.Close()
	}()
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

type journalctlCommandSpec struct {
	command string
	args    []string
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
