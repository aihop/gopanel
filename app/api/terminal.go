package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/cmd"
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
	pidMap := loadMapFromDockerTop(containerID)
	slave, err := terminal.NewCommand(initCmd)
	if wshandleError(wsConn, err) {
		return
	}
	defer killBash(containerID, strings.ReplaceAll(strings.Join(initCmd, " "), fmt.Sprintf("exec -it %s ", containerID), ""), pidMap)
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

func wshandleError(ws *websocket.Conn, err error) bool {
	if err != nil {
		global.LOG.Errorf("handler ws faled:, err: %v", err)
		dt := time.Now().Add(time.Second)
		if ctlerr := ws.WriteControl(websocket.CloseMessage, []byte(err.Error()), dt); ctlerr != nil {
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
	pidMap := make(map[string]string)
	sudo := cmd.SudoHandleCmd()

	stdout, err := cmd.Execf("%s docker top %s -eo pid,command ", sudo, containerID)
	if err != nil {
		return pidMap
	}
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pidMap[parts[0]] = strings.Join(parts[1:], " ")
	}
	return pidMap
}

func killBash(containerID, comm string, pidMap map[string]string) {
	sudo := cmd.SudoHandleCmd()
	newPidMap := loadMapFromDockerTop(containerID)
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

// var upGrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024 * 1024 * 10,
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }
