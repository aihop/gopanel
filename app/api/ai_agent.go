package api

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/token"
	"github.com/creack/pty"
)

// WsMsg is the terminal message format
type WsMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func formatExecOutput(output []byte, fallback string) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return fallback
	}
	return text
}

func defaultAIAgentWorkDir(userID uint) string {
	hostHome, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(hostHome) == "" {
		hostHome = "/tmp"
	}
	if userID == 0 {
		return filepath.Join(hostHome, ".gopanel", "ai_agent", "workspace", "default")
	}
	return filepath.Join(hostHome, ".gopanel", "ai_agent", "workspace", fmt.Sprintf("user_%d", userID))
}

func normalizeAIAgentWorkDir(workDir string, userID uint) string {
	workDir = filepath.Clean(strings.TrimSpace(workDir))
	if workDir == "" || workDir == "." {
		return defaultAIAgentWorkDir(userID)
	}

	if runtime.GOOS == "darwin" {
		if workDir == "/" || workDir == "/root" || !strings.HasPrefix(workDir, "/Users/") {
			return defaultAIAgentWorkDir(userID)
		}
	}

	return workDir
}

func AIAgentWsSSH(wsConn *websocket.Conn) {
	defer wsConn.Close()

	// 1. 权限与沙箱控制
	workDir := ""
	var userID uint

	if claims, ok := wsConn.Locals(constant.AppAuthName).(*token.CustomClaims); ok {
		userID = claims.UserId
		if claims.Role == constant.UserRoleSubAdmin {
			// 普通管理员沙箱限制：只能在 FileBaseDir 及其子目录下活动
			if claims.FileBaseDir != "" {
				baseDir := filepath.Clean(claims.FileBaseDir)
				if !strings.HasPrefix(workDir, baseDir) {
					// 越权访问，强制限制在 baseDir
					workDir = baseDir
				}
			} else {
				workDir = "/"
			}
		}
	}

	// 检查是否是通过 task_id 恢复历史任务
	reqTaskID, _ := strconv.Atoi(wsConn.Query("task_id", "0"))
	reqProjectID, _ := strconv.Atoi(wsConn.Query("project_id", "0"))
	var currentTask *model.AITask
	aiRepo := repo.NewAITaskRepo()

	if reqTaskID > 0 {
		if task, err := aiRepo.GetTaskByID(uint(reqTaskID)); err == nil {
			currentTask = task
			workDir = task.WorkDir
		}
	} else {
		// 从前端获取用户指定的工作目录
		reqCwd := wsConn.Query("cwd")
		if reqCwd != "" {
			workDir = filepath.Clean(reqCwd)
		}
	}

	workDir = normalizeAIAgentWorkDir(workDir, userID)

	// 确保目录存在
	_ = os.MkdirAll(workDir, 0755)

	cols, _ := strconv.Atoi(wsConn.Query("cols", "80"))
	rows, _ := strconv.Atoi(wsConn.Query("rows", "24"))

	// 2. 持久化容器 (Workspace Container) 架构
	dockerImage := "node:18-alpine" // 默认镜像

	// 为这个工作区生成一个唯一的容器名
	containerName := fmt.Sprintf("cx_agent_%x", md5.Sum([]byte(workDir)))

	// 检查该容器是否存在且正在运行
	inspectCmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName)
	inspectOut, err := inspectCmd.Output()

	if err != nil {
		// 容器不存在，需要新建一个后台守护容器 (Daemon)
		// 注意这里使用 -d (后台运行) 且没有 --rm，最后执行 tail -f /dev/null 保证它永远不退出
		runArgs := []string{
			"run", "-d", "--name", containerName,
			"-v", fmt.Sprintf("%s:/workspace", workDir),
			"-w", "/workspace",
		}

		// 智能检测并追加只读凭证挂载
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
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				// 映射到容器的 /root 目录下
				containerPath := "/root/" + filepath.Base(path)
				runArgs = append(runArgs, "-v", fmt.Sprintf("%s:%s:ro", path, containerPath))
			}
		}

		// 准备在容器内部注入快捷脚本（极简模式，通知后端接管）
		daemonCmd := `
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

		runArgs = append(runArgs, dockerImage, "sh", "-c", daemonCmd)

		if output, err := exec.Command("docker", runArgs...).CombinedOutput(); err != nil {
			global.LOG.Errorf("Failed to create workspace container %s: %v", containerName, err)
			message := fmt.Sprintf("创建持久化沙箱失败: %s\r\n", formatExecOutput(output, err.Error()))
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte(message))
			return
		}
	} else {
		// 容器存在，检查是否停机
		isRunning := strings.TrimSpace(string(inspectOut)) == "true"
		if !isRunning {
			// 如果容器停机了（如服务器重启过），唤醒它
			if output, err := exec.Command("docker", "start", containerName).CombinedOutput(); err != nil {
				global.LOG.Errorf("Failed to start workspace container %s: %v", containerName, err)
				message := fmt.Sprintf("唤醒持久化沙箱失败: %s\r\n", formatExecOutput(output, err.Error()))
				_ = wsConn.WriteMessage(websocket.TextMessage, []byte(message))
				return
			}
		}
	}

	// 3. 通过 docker exec 接入这个持久化容器
	// 检查是否需要在启动时默认拉起指定的智能体
	agentName := wsConn.Query("agent") // 例如前端可以传 ?agent=trae

	if currentTask != nil && currentTask.AgentName != "" {
		agentName = currentTask.AgentName
	}

	autoStartAI := agentName != ""

	welcomeCmd := "echo -e '\\033[32m欢迎回到 GoPanel AI 持久化沙箱。\\033[0m';"
	if autoStartAI {
		welcomeCmd += fmt.Sprintf("echo -e '💡 \\033[33m已为您自动拉起 %s 智能体。输入 \"exit\" 可退回普通 Shell。\\033[0m';", agentName)
		// 发送一个隐形的信标，让 Go 后端的读取协程瞬间将状态机切入 AI 模式
		welcomeCmd += "echo -e '\\033[36m[CX-AI-HOOK:START-INTERACTIVE]\\033[0m';"
	} else {
		welcomeCmd += "echo -e '💡 \\033[33m提示: 直接输入 \"@trae\" 或 \"@ai\" 即可唤起智能体！\\033[0m';"
	}

	execArgs := []string{
		"docker", "exec", "-it",
		"-e", "TERM=xterm-256color",
		"-e", "COLORTERM=truecolor",
		"-e", fmt.Sprintf("COLUMNS=%d", cols),
		"-e", fmt.Sprintf("LINES=%d", rows),
		containerName,
		"sh", "-c", welcomeCmd + " /bin/sh",
	}

	cmd := exec.Command(execArgs[0], execArgs[1:]...)

	// 3. 启动 PTY (伪终端)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		global.LOG.Errorf("Failed to start pty: %v", err)
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Failed to start AI Agent terminal."))
		return
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = cmd.Process.Kill() }()

	// 在函数开始附近定义状态机变量，以便在读写 goroutine 中共享
	inAIChatMode := false
	var aiInputBuffer strings.Builder

	// 4. 读取 PTY 输出，推送到前端 WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					global.LOG.Errorf("Error reading from pty: %v", err)
				}
				break
			}

			outputStr := string(buf[:n])

			// 如果读取到的数据为空，或者只有不可见的信标且被替换为空后，不需要发给前端
			if outputStr == "" {
				continue
			}

			// 检测信标：是否需要切换到 AI 交互模式
			if strings.Contains(outputStr, "[CX-AI-HOOK:START-INTERACTIVE]") {
				inAIChatMode = true

				// 给前端发送进入聊天模式的友好提示
				welcomeMsg := "\r\n\033[32m[系统] GoPanel AI 引擎已接管终端。\033[0m\r\n" +
					"\033[36m现在您可以直接用自然语言与 AI 对话了。(输入 'exit' 退出聊天)\033[0m\r\n\033[32m[AI Agent] > \033[0m"

				// 去除信标本身的输出，替换为欢迎信息
				outputStr = strings.ReplaceAll(outputStr, "\033[36m[CX-AI-HOOK:START-INTERACTIVE]\033[0m", welcomeMsg)
				outputStr = strings.ReplaceAll(outputStr, "[CX-AI-HOOK:START-INTERACTIVE]", "")
			}

			// 处理单次指令的信标 (暂不处理复杂的单次交互拦截，直接打印即可)
			if strings.Contains(outputStr, "[CX-AI-HOOK:ONE-SHOT]") {
				outputStr = strings.ReplaceAll(outputStr, "[CX-AI-HOOK:ONE-SHOT]", "[系统] 已拦截单次 AI 任务：")
			}

			// 如果进入了 AI 会话模式，我们需要过滤掉底层 Shell 的命令提示符回显 (如 /workspace # )
			if inAIChatMode {
				// 常见的 alpine shell 提示符特征
				if strings.Contains(outputStr, "/workspace #") || strings.Contains(outputStr, "/ #") {
					// 过滤掉提示符
					outputStr = strings.ReplaceAll(outputStr, "/workspace # ", "")
					outputStr = strings.ReplaceAll(outputStr, "/workspace #", "")
					outputStr = strings.ReplaceAll(outputStr, "/ # ", "")
					outputStr = strings.ReplaceAll(outputStr, "/ #", "")
				}

				// 如果在 AI 模式下，底层 Linux 又发来了一个纯回车换行，我们也忽略它，
				// 因为我们在 WebSocket 接收端已经自己处理了 AI 聊天的换行。
				if strings.TrimSpace(outputStr) == "" {
					continue
				}
			}

			// 替换信标后如果字符串变成空了（比如原本只有信标被替换成了空字符），也不发送
			if strings.TrimSpace(outputStr) == "" {
				// 有些时候底层会传一些纯 ANSI 控制符或空白符
				continue
			}

			// 如果处理到最后，数据为空，绝不向前端发空包
			if outputStr == "" {
				continue
			}

			msg := WsMsg{
				Type: "cmd",
				Data: outputStr,
			}
			jsonMsg, _ := json.Marshal(msg)
			if err := wsConn.WriteMessage(websocket.TextMessage, jsonMsg); err != nil {
				break
			}
		}
	}()

	// 5. 接收前端 WebSocket 输入，写入 PTY
	for {
		messageType, p, err := wsConn.ReadMessage()
		if err != nil {
			global.LOG.Infof("WebSocket closed or error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			var msg WsMsg
			if err := json.Unmarshal(p, &msg); err != nil {
				continue
			}

			// 辅助函数：安全地将字符串打包为 WebSocket JSON 发送
			sendWsMsg := func(data string) {
				responseMsg := WsMsg{
					Type: "cmd",
					Data: data,
				}
				jsonMsg, _ := json.Marshal(responseMsg)
				_ = wsConn.WriteMessage(websocket.TextMessage, jsonMsg)
			}

			switch msg.Type {
			case "cmd":
				// 前端敲击键盘的字符
				if inAIChatMode {
					// --- AI 聊天拦截模式 ---
					// 处理退格键 (Backspace/Delete)
					if msg.Data == "\x7f" || msg.Data == "\b" {
						if aiInputBuffer.Len() > 0 {
							// 截断最后一个字符
							str := aiInputBuffer.String()
							aiInputBuffer.Reset()
							aiInputBuffer.WriteString(str[:len(str)-1])
							// 在前端终端擦除字符：退格 -> 空格 -> 退格
							sendWsMsg("\b \b")
						}
						continue
					}

					// 处理回车键 (Enter)
					if msg.Data == "\r" || msg.Data == "\n" {
						sendWsMsg("\r\n")

						userInput := strings.TrimSpace(aiInputBuffer.String())
						aiInputBuffer.Reset()

						if userInput == "exit" || userInput == "quit" {
							inAIChatMode = false
							sendWsMsg("\033[33m[系统] 已退出 AI 交互模式，恢复 Shell 环境。\033[0m\r\n")
							// 为了让终端重新显示 Shell 的提示符 (如 /workspace # )，向 PTY 发送一个回车
							_, _ = ptmx.Write([]byte("\r"))
							continue
						}

						if userInput != "" {
							// 持久化逻辑：如果这是当前会话的第一条消息，自动创建 Task
							if currentTask == nil {
								// 生成任务标题（取前20个字符）
								title := userInput
								if len([]rune(title)) > 20 {
									title = string([]rune(title)[:20]) + "..."
								}

								// 优先取 URL 参数，没有则给个默认值
								if agentName == "" {
									agentName = "trae"
								}

								newTask := &model.AITask{
									UserID:    userID,
									ProjectID: uint(reqProjectID),
									Title:     title,
									AgentName: agentName,
									WorkDir:   workDir,
									Status:    "active",
								}
								if err := aiRepo.CreateTask(newTask); err == nil {
									currentTask = newTask
									// 可以通过 WebSocket 发送特殊事件，通知前端更新 URL (加上 ?task_id=)
									_ = wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"meta","task_id":%d}`, currentTask.ID)))
								}
							}

							// 持久化用户提问
							if currentTask != nil {
								_ = aiRepo.CreateMessage(&model.AIMessage{
									TaskID:  currentTask.ID,
									Role:    "user",
									Content: userInput,
								})
							}

							// 调用真实的容器内部智能体 (例如 trae)
							// 这里通过 docker exec 在当前容器中单次执行命令，并捕获输出
							sendWsMsg("\033[36m[AI Agent] 正在思考并执行...\033[0m\r\n")

							go func(input string, task *model.AITask) {
								// 使用 docker exec -i (不带 -t，因为这里只是捕获纯文本输出)
								cmdArgs := []string{
									"docker", "exec", "-i", containerName,
									"sh", "-c",
								}

								// 构造执行逻辑：先检查工具是否存在，存在则执行，不存在则友好提示
								// 注意：这里使用的是单次运行模式 (One-shot)
								shellCmd := fmt.Sprintf(`
								if command -v trae >/dev/null 2>&1; then
									trae --message "%s"
								elif command -v aider >/dev/null 2>&1; then
									aider --message "%s"
								else
									echo -e "\033[33m[系统提示] 当前沙箱环境尚未安装 trae 或 aider 工具。\033[0m"
									echo -e "您可以先输入 'exit' 退回普通终端，然后执行 \033[32mnpm install -g trae\033[0m 进行安装。"
								fi
								`, strings.ReplaceAll(input, `"`, `\"`), strings.ReplaceAll(input, `"`, `\"`))

								cmdArgs = append(cmdArgs, shellCmd)

								execCmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
								out, err := execCmd.CombinedOutput()

								// 将输出流式（或一次性）推回给前端终端
								if err != nil {
									global.LOG.Errorf("AI execution error: %v, out: %s", err, string(out))
									if len(out) == 0 {
										out = []byte(fmt.Sprintf("执行错误: %v", err))
									}
								}

								// 持久化 AI 的回复
								if task != nil {
									_ = aiRepo.CreateMessage(&model.AIMessage{
										TaskID:  task.ID,
										Role:    "agent",
										Content: string(out),
									})
								}

								// 处理输出的换行符，适应 xterm.js (\n 替换为 \r\n)
								formattedOut := strings.ReplaceAll(string(out), "\n", "\r\n")
								if !strings.HasSuffix(formattedOut, "\r\n") {
									formattedOut += "\r\n"
								}

								sendWsMsg(formattedOut)

								// 重新打印提示符
								sendWsMsg("\033[32m[AI Agent] > \033[0m")
							}(userInput, currentTask)

							// 这里的 return/continue 是在主循环里，所以不能阻塞，上面的逻辑已经用 go func 包装
							continue
						}

						// 如果输入为空，继续显示聊天提示符
						sendWsMsg("\033[32m[AI Agent] > \033[0m")
					} else {
						// 普通字符，存入 Buffer 并回显给前端
						aiInputBuffer.WriteString(msg.Data)
						sendWsMsg(msg.Data)
					}

				} else {
					// --- 正常 Shell 模式 ---
					_, _ = ptmx.Write([]byte(msg.Data))
				}
			case "resize":
				// 处理终端大小调整
				type resizeData struct {
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				var r resizeData
				if err := json.Unmarshal([]byte(msg.Data), &r); err == nil {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: r.Rows,
						Cols: r.Cols,
					})
				}
			case "ping":
				// 保持连接
				_ = wsConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			}
		}
	}

	// 优雅关闭
	dt := time.Now().Add(time.Second)
	_ = wsConn.WriteControl(websocket.CloseMessage, nil, dt)
}
