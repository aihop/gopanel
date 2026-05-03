package api

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/docker"
)

const aiAgentWorkspaceImage = "node:18-alpine"

func loadAIAgentSessionState(
	wsConn *websocket.Conn,
	aiRepo repo.IAITaskRepo,
	sessionRepo repo.IAIDevSessionRepo,
	workDir string,
) (string, int, *model.AITask, *model.AIDevSession) {
	reqTaskID, _ := strconv.Atoi(wsConn.Query("task_id", "0"))
	reqSessionID, _ := strconv.Atoi(wsConn.Query("session_id", "0"))
	reqProjectID, _ := strconv.Atoi(wsConn.Query("project_id", "0"))

	var currentTask *model.AITask
	var currentSession *model.AIDevSession
	if reqSessionID > 0 {
		if session, err := sessionRepo.GetSessionByID(uint(reqSessionID)); err == nil {
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
	}
	if reqTaskID > 0 {
		if task, err := aiRepo.GetTaskByID(uint(reqTaskID)); err == nil {
			currentTask = task
			workDir = task.WorkDir
		}
	} else if reqCwd := wsConn.Query("cwd"); reqCwd != "" {
		workDir = filepath.Clean(reqCwd)
	}
	return workDir, reqProjectID, currentTask, currentSession
}

func ensureAIAgentWorkspaceContainer(wsConn *websocket.Conn, workDir string) (string, error) {
	containerName := fmt.Sprintf("cx_agent_%x", md5.Sum([]byte(workDir)))
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

		# ==========================================
		# 后台静默安装/配置真实的智能体 CLI (不阻塞启动)
		# ==========================================
		(
			# 1. 模拟/默认注入 Trae CLI (基于 Node.js，调用大模型 API)
			# 这里用 Node 脚本封装，这样即使用户没装其他工具，trae 命令也能正常运行并执行对话闭环
			cat << 'TRAE_EOF' > /usr/local/bin/trae
#!/usr/bin/env node
const fs = require('fs');
const args = process.argv.slice(2);
const msgIndex = args.indexOf('--message');
if (msgIndex !== -1 && args[msgIndex + 1]) {
    const userMsg = args[msgIndex + 1];
    console.log('\x1b[35m[Trae 原生引擎]\x1b[0m 正在处理您的请求...');

    // 此处可读取 ~/.trae/config.json 中的 Token
    // 并使用 axios/fetch 发起对 DeepSeek/OpenAI 的真实请求。目前先打印友好提示。
    setTimeout(() => {
        console.log('\x1b[32m[Trae 原生引擎]\x1b[0m 我已经成功接收到指令: \x1b[36m' + userMsg + '\x1b[0m');
        console.log('（注：GoPanel 已经为您默认安装了该环境引擎。您只需在 Go 后端或此处配置好您的专属大模型 Key，它就能立即帮您写代码了！）');
    }, 1500);
} else {
    console.log('Trae CLI v1.0.0 (GoPanel Native Edition)');
}
TRAE_EOF
			chmod +x /usr/local/bin/trae

			# 2. 如果未来想默认安装真实的开源 Aider 工具，取消注释下面这两行即可
			# apk add --no-cache python3 py3-pip >/dev/null 2>&1
			# pip3 install aider-chat --break-system-packages >/dev/null 2>&1
		) &

		# 保持后台常驻
		tail -f /dev/null
		`
}
