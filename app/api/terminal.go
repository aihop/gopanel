package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/terminal"
	"github.com/aihop/gopanel/utils/token"
)

func ContainerWsSSH(wsConn *websocket.Conn) {
	defer wsConn.Close()

	if claims, ok := wsConn.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		if wshandleError(wsConn, errors.New("permission denied: sub_admin cannot use terminal")) {
			return
		}
	}

	if global.CONF.System.IsDemo {
		if wshandleError(wsConn, errors.New("demo server, prohibit this operation!")) {
			return
		}
	}
	cols, err := strconv.Atoi(wsConn.Query("cols", "80"))
	if wshandleError(wsConn, err) {
		return
	}
	rows, err := strconv.Atoi(wsConn.Query("rows", "40"))
	if wshandleError(wsConn, err) {
		return
	}
	source := wsConn.Query("source")
	runtimeHost := wsConn.Query("runtimeHost")
	var containerID string
	var initCmd []string
	switch source {
	case "container":
		containerID, initCmd, err = loadContainerInitCmd(wsConn)
	default:
		if wshandleError(wsConn, fmt.Errorf("not support such source %s", source)) {
			return
		}
	}
	if wshandleError(wsConn, err) {
		return
	}
	finalHost := strings.TrimSpace(runtimeHost)
	if finalHost == "" && docker.IsPodmanRuntime(context.Background()) {
		finalHost = resolvePodmanContainerHost(containerID)
	}
	if wshandleError(wsConn, ensureContainerTerminalReady(context.Background(), containerID, finalHost)) {
		return
	}

	pidMap := loadMapFromRuntimeTop(containerID, finalHost)
	slave, err := terminal.NewCommandWithRuntimeHost(initCmd, finalHost)
	if wshandleError(wsConn, normalizeContainerTerminalStartError(err)) {
		return
	}
	defer killBash(containerID, strings.ReplaceAll(strings.Join(initCmd, " "), fmt.Sprintf("exec -it %s ", containerID), ""), pidMap, finalHost)
	defer slave.Close()

	tty, err := terminal.NewLocalWsSession(cols, rows, wsConn, slave, false)
	if wshandleError(wsConn, err) {
		return
	}

	quitChan := make(chan bool, 3)
	tty.Start(quitChan)
	go slave.Wait(quitChan)

	<-quitChan

	global.LOG.Info("websocket finished")
	dt := time.Now().Add(time.Second)
	_ = wsConn.WriteControl(websocket.CloseMessage, nil, dt)

}

func loadContainerInitCmd(c *websocket.Conn) (string, []string, error) {
	containerID := c.Query("containerid")
	command := c.Query("command")
	user := c.Query("user")
	if cmd.CheckIllegal(user, containerID, command) {
		return "", nil, fmt.Errorf("the command contains illegal characters. command: %s, user: %s, containerID: %s", command, user, containerID)
	}
	if len(command) == 0 || len(containerID) == 0 {
		return "", nil, fmt.Errorf("error param of command: %s or containerID: %s", command, containerID)
	}
	commands := []string{"exec", "-it", containerID, command}
	if len(user) != 0 {
		commands = []string{"exec", "-it", "-u", user, containerID, command}
	}

	return containerID, commands, nil
}

type containerTerminalInspector func(context.Context, string, string) ([]byte, error)

func ensureContainerTerminalReady(ctx context.Context, containerID, runtimeHost string) error {
	return ensureContainerTerminalReadyWith(ctx, containerID, runtimeHost, inspectContainerTerminalState)
}

func ensureContainerTerminalReadyWith(ctx context.Context, containerID, runtimeHost string, inspect containerTerminalInspector) error {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := inspect(checkCtx, containerID, runtimeHost)
	if checkCtx.Err() != nil {
		err = checkCtx.Err()
	}
	return containerTerminalStateError(output, err)
}

func inspectContainerTerminalState(ctx context.Context, containerID, runtimeHost string) ([]byte, error) {
	c, err := docker.RuntimeCommandWithHost(ctx, runtimeHost, "inspect", "-f", "{{.State.Running}}", containerID)
	if err != nil {
		return nil, err
	}
	return c.CombinedOutput()
}

func containerTerminalStateError(output []byte, commandErr error) error {
	detail := strings.ToLower(strings.TrimSpace(string(output)))
	if commandErr != nil {
		detail = strings.TrimSpace(detail + " " + strings.ToLower(commandErr.Error()))
	}
	if commandErr == nil {
		switch detail {
		case "true":
			return nil
		case "false":
			return errors.New("容器当前未运行，请启动容器后重试")
		}
	}
	if errors.Is(commandErr, context.DeadlineExceeded) || errors.Is(commandErr, context.Canceled) {
		return errors.New("连接容器运行时超时，请稍后重试")
	}
	if containsContainerTerminalError(detail, "no such object", "no such container", "not found", "does not exist") {
		return errors.New("容器不存在或运行时已变化，请刷新列表后重试")
	}
	if containsContainerTerminalError(detail, "container state improper", "can only create exec sessions on running containers", "is not running", "container stopped") {
		return errors.New("容器当前未运行，请启动容器后重试")
	}
	return errors.New("无法读取容器状态，请确认容器运行时可用后重试")
}

func normalizeContainerTerminalStartError(err error) error {
	if err == nil {
		return nil
	}
	detail := strings.ToLower(err.Error())
	if containsContainerTerminalError(detail, "container state improper", "can only create exec sessions on running containers", "is not running", "container stopped") {
		return errors.New("容器当前未运行，请启动容器后重试")
	}
	if containsContainerTerminalError(detail, "no such object", "no such container", "not found", "does not exist") {
		return errors.New("容器不存在或运行时已变化，请刷新列表后重试")
	}
	return errors.New("启动容器终端失败，请确认容器和运行时状态后重试")
}

func containsContainerTerminalError(detail string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(detail, fragment) {
			return true
		}
	}
	return false
}

func wshandleError(ws *websocket.Conn, err error) bool {
	if err != nil {
		global.LOG.Errorf("handler ws faled:, err: %v", err)
		dt := time.Now().Add(time.Second)
		closeMessage := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
		if ctlerr := ws.WriteControl(websocket.CloseMessage, closeMessage, dt); ctlerr != nil {
			wsData, err := json.Marshal(terminal.WsMsg{
				Type: terminal.WsMsgCmd,
				Data: base64.StdEncoding.EncodeToString([]byte(err.Error())),
			})
			if err != nil {
				_ = ws.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"cmd\",\"data\":\"failed to encoding to json\"}"))
			} else {
				_ = ws.WriteMessage(websocket.TextMessage, wsData)
			}
		}
		return true
	}
	return false
}

func loadMapFromDockerTop(containerID string) map[string]string {
	return loadMapFromRuntimeTop(containerID, "")
}

func loadMapFromRuntimeTop(containerID string, runtimeHost string) map[string]string {
	pidMap := make(map[string]string)
	c, err := docker.RuntimeCommandWithHost(context.Background(), runtimeHost, "top", containerID, "-eo", "pid,command")
	if err != nil {
		return pidMap
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return pidMap
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pidMap[parts[0]] = strings.Join(parts[1:], " ")
	}
	return pidMap
}

func killBash(containerID, comm string, pidMap map[string]string, runtimeHost string) {
	sudo := cmd.SudoHandleCmd()
	newPidMap := loadMapFromRuntimeTop(containerID, runtimeHost)
	for pid, command := range newPidMap {
		isOld := false
		for pid2 := range pidMap {
			if pid == pid2 {
				isOld = true
				break
			}
		}
		if !isOld && command == comm {
			_, _ = cmd.Execf("%s kill -9 %s", sudo, pid)
		}
	}
}

func resolvePodmanContainerHost(containerID string) string {
	id := strings.TrimSpace(containerID)
	if id == "" {
		return ""
	}
	c := exec.Command("podman", "container", "exists", id)
	if err := c.Run(); err == nil {
		return ""
	}
	host := "unix:///run/podman/podman.sock"
	c2 := exec.Command("podman", "--url", host, "container", "exists", id)
	if err := c2.Run(); err == nil {
		return host
	}
	return ""
}

// var upGrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024 * 1024 * 10,
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }
