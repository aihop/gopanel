package api

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/token"
)

const aiAgentWorkspaceImage = "node:18-alpine"

func loadAIAgentSessionState(
	wsConn *websocket.Conn,
	aiRepo repo.IAITaskRepo,
	sessionRepo repo.IAIDevSessionRepo,
	workDir string,
	claims *token.CustomClaims,
) (string, int, *model.AITask, *model.AIDevSession, error) {
	reqTaskID, _ := strconv.Atoi(wsConn.Query("task_id", "0"))
	reqSessionID, _ := strconv.Atoi(wsConn.Query("session_id", "0"))
	reqProjectID, _ := strconv.Atoi(wsConn.Query("project_id", "0"))

	var currentTask *model.AITask
	var currentSession *model.AIDevSession
	if reqSessionID > 0 {
		session, err := sessionRepo.GetSessionByID(uint(reqSessionID))
		if err != nil {
			return "", 0, nil, nil, fmt.Errorf("开发会话不存在")
		}
		if session.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
			return "", 0, nil, nil, fmt.Errorf("无权访问该开发会话")
		}
		currentSession = session
		workDir = session.WorkDir
		if reqProjectID == 0 {
			reqProjectID = int(session.ProjectID)
		}
		if session.LastTaskID > 0 {
			if task, taskErr := aiRepo.GetTaskByID(session.LastTaskID); taskErr == nil {
				currentTask = task
			}
		}
	}
	if reqTaskID > 0 {
		if task, err := aiRepo.GetTaskByID(uint(reqTaskID)); err == nil {
			if task.UserID != claims.UserId && claims.Role != constant.UserRoleSuper {
				return "", 0, nil, nil, fmt.Errorf("无权访问该 AI 任务")
			}
			currentTask = task
			workDir = task.WorkDir
			if task.SessionID > 0 && currentSession == nil {
				session, sessionErr := sessionRepo.GetSessionByID(task.SessionID)
				if sessionErr == nil && (session.UserID == claims.UserId || claims.Role == constant.UserRoleSuper) {
					currentSession = session
				}
			}
		}
	} else if reqCwd := wsConn.Query("cwd"); reqCwd != "" {
		workDir = filepath.Clean(reqCwd)
	}
	if claims.Role == constant.UserRoleSubAdmin {
		if err := service.ValidatePathWithinBase(claims.FileBaseDir, workDir); err != nil {
			return "", 0, nil, nil, err
		}
	}
	return workDir, reqProjectID, currentTask, currentSession, nil
}

func normalizeAIAgentAuthorizedWorkDir(workDir string, userID uint, claims *token.CustomClaims) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("unauthorized")
	}
	if claims.Role != constant.UserRoleSubAdmin {
		return normalizeAIAgentWorkDir(workDir, userID), nil
	}
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if err := service.ValidatePathWithinBase(claims.FileBaseDir, workDir); err != nil {
		return "", err
	}
	return workDir, nil
}

func ensureAIAgentWorkspaceContainer(wsConn *websocket.Conn, workDir string, claims *token.CustomClaims) (string, error) {
	containerKey := fmt.Sprintf("%d:%s", claims.UserId, workDir)
	containerName := fmt.Sprintf("cx_agent_%x", md5.Sum([]byte(containerKey)))
	isRunning, exists, err := docker.InspectContainerRunning(context.Background(), containerName)
	if err != nil {
		global.LOG.Errorf("Failed to inspect workspace container %s: %v", containerName, err)
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("检查持久化沙箱失败: %s\r\n", err.Error())))
		return "", err
	}

	if !exists {
		runArgs := []string{
			"run", "-d", "--name", containerName,
			"-v", fmt.Sprintf("%s:/workspace", workDir),
			"-w", "/workspace",
		}

		if claims.Role == constant.UserRoleAdmin || claims.Role == constant.UserRoleSuper {
			hostHome, _ := os.UserHomeDir()
			if hostHome == "" {
				hostHome = "/root"
			}
			credentialsPaths := []string{
				filepath.Join(hostHome, ".ssh"),
				filepath.Join(hostHome, ".trae"),
				filepath.Join(hostHome, ".aws"),
				filepath.Join(hostHome, ".npmrc"),
				filepath.Join(hostHome, ".gitconfig"),
			}
			for _, path := range credentialsPaths {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					containerPath := "/root/" + filepath.Base(path)
					runArgs = append(runArgs, "-v", fmt.Sprintf("%s:%s:ro", path, containerPath))
				}
			}
		}

		runArgs = append(runArgs, aiAgentWorkspaceImage, "sh", "-c", buildAIAgentWorkspaceDaemonCommand())
		runCmd, cmdErr := docker.RuntimeCommand(context.Background(), runArgs...)
		if cmdErr != nil {
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("创建持久化沙箱失败: %s\r\n", cmdErr.Error())))
			return "", cmdErr
		}
		if output, runErr := runCmd.CombinedOutput(); runErr != nil {
			global.LOG.Errorf("Failed to create workspace container %s: %v", containerName, runErr)
			message := fmt.Sprintf("创建持久化沙箱失败: %s\r\n", formatExecOutput(output, runErr.Error()))
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(message))
			return "", runErr
		}
		return containerName, nil
	}

	if isRunning {
		return containerName, nil
	}

	startCmd, cmdErr := docker.RuntimeCommand(context.Background(), "start", containerName)
	if cmdErr != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("唤醒持久化沙箱失败: %s\r\n", cmdErr.Error())))
		return "", cmdErr
	}
	if output, startErr := startCmd.CombinedOutput(); startErr != nil {
		global.LOG.Errorf("Failed to start workspace container %s: %v", containerName, startErr)
		message := fmt.Sprintf("唤醒持久化沙箱失败: %s\r\n", formatExecOutput(output, startErr.Error()))
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(message))
		return "", startErr
	}
	return containerName, nil
}

func buildAIAgentWorkspaceDaemonCommand() string {
	return `
		mkdir -p /usr/local/bin
		cat << 'EOF' > /usr/local/bin/@智能体
#!/bin/sh
# 这是一个特殊的信标脚本。
# 当用户在 PTY 中输入此命令时，它会向终端输出特定的标识符，
# 从而触发 Go 后端的状态机拦截，接管后续的会话。

if [ -z "$*" ]; then
    # 无参数，触发交互模式
    echo -e "\033[36m[CX-AI-HOOK:START-INTERACTIVE]\033[0m"
else
    # 有参数，触发单次指令模式
    echo -e "\033[36m[CX-AI-HOOK:ONE-SHOT] $*\033[0m"
fi
EOF
		chmod +x /usr/local/bin/@智能体
		cp /usr/local/bin/@智能体 /usr/local/bin/@ai
		cp /usr/local/bin/@智能体 /usr/local/bin/ai
		cp /usr/local/bin/@智能体 /usr/local/bin/@trae

			# 保持后台常驻
		tail -f /dev/null
		`
}
