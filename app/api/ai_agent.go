package api

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/token"
	"github.com/creack/pty"
)

type WsMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

var previewURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+|(?:localhost|127\.0\.0\.1|0\.0\.0\.0):\d+[^\s"'<>]*`)

func AIAgentWsSSH(wsConn *websocket.Conn) {
	defer wsConn.Close()
	workDir := ""
	authClaims, ok := wsConn.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || authClaims == nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Unauthorized."))
		return
	}
	userID := authClaims.UserId
	if authClaims.Role == constant.UserRoleSubAdmin {
		workDir = strings.TrimSpace(authClaims.FileBaseDir)
		if workDir == "" {
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Sub-admin workspace is not configured."))
			return
		}
	}
	aiRepo := repo.NewAITaskRepo()
	sessionRepo := repo.NewAIDevSessionRepo()
	workDir, reqProjectID, currentTask, currentSession, err := loadAIAgentSessionState(wsConn, aiRepo, sessionRepo, workDir, authClaims)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	workDir, err = normalizeAIAgentAuthorizedWorkDir(workDir, userID, authClaims)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	cols, _ := strconv.Atoi(wsConn.Query("cols", "80"))
	rows, _ := strconv.Atoi(wsConn.Query("rows", "24"))
	containerName, err := ensureAIAgentWorkspaceContainer(wsConn, workDir, authClaims)
	if err != nil {
		return
	}
	agentName := wsConn.Query("agent")
	if currentTask != nil && currentTask.AgentName != "" {
		agentName = currentTask.AgentName
	} else if currentSession != nil && currentSession.AgentName != "" {
		agentName = currentSession.AgentName
	}
	autoStartAI := strings.TrimSpace(agentName) != ""
	agentName, err = normalizeAIAgentName(agentName)
	if err != nil {
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	welcomeCmd := "echo -e '\\033[32m欢迎回到 GoPanel AI 持久化沙箱。\\033[0m';"
	if autoStartAI {
		welcomeCmd += fmt.Sprintf("echo -e '💡 \\033[33m已为您自动拉起 %s 智能体。输入 \"exit\" 可退回普通 Shell。\\033[0m';", agentName)
		welcomeCmd += "echo -e '\\033[36m[CX-AI-HOOK:START-INTERACTIVE]\\033[0m';"
	} else {
		welcomeCmd += "echo -e '💡 \\033[33m提示: 直接输入 \"@trae\" 或 \"@ai\" 即可唤起智能体！\\033[0m';"
	}
	execArgs := []string{"exec", "-it", "-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor", "-e", fmt.Sprintf("COLUMNS=%d", cols), "-e", fmt.Sprintf("LINES=%d", rows), containerName, "sh", "-c", welcomeCmd + " /bin/sh"}
	cmd, err := docker.RuntimeCommand(context.Background(), execArgs...)
	if err != nil {
		global.LOG.Errorf("Failed to create exec command: %v", err)
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Failed to start AI Agent terminal."))
		return
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		global.LOG.Errorf("Failed to start pty: %v", err)
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Failed to start AI Agent terminal."))
		return
	}
	defer func() {
		_ = ptmx.Close()
	}()
	defer func() {
		_ = cmd.Process.Kill()
	}()
	inAIChatMode := false
	var aiInputBuffer strings.Builder
	pendingInstructionsLoaded := false
	var wsWriteMu sync.Mutex
	sendWsMsg := func(data string) {
		responseMsg := WsMsg{Type: "cmd", Data: data}
		jsonMsg, _ := json.Marshal(responseMsg)
		wsWriteMu.Lock()
		defer wsWriteMu.Unlock()
		_ = wsConn.WriteMessage(websocket.TextMessage, jsonMsg)
	}
	sendWsRaw := func(data string) {
		wsWriteMu.Lock()
		defer wsWriteMu.Unlock()
		_ = wsConn.WriteMessage(websocket.TextMessage, []byte(data))
	}
	aiExecQueue := make(chan aiExecRequest, 16)
	done := make(chan struct{})
	defer close(done)
	enqueueAIExecution := func(req aiExecRequest) {
		select {
		case <-done:
			return
		case aiExecQueue <- req:
		default:
			sendWsMsg("\r\n\033[33m[系统] 当前 AI 执行队列繁忙，请稍后再试。\033[0m\r\n")
		}
	}
	go func() {
		for {
			select {
			case <-done:
				return
			case req, ok := <-aiExecQueue:
				if !ok {
					return
				}
				if strings.TrimSpace(req.input) == "" {
					continue
				}
				if req.task != nil {
					req.task.Status = "running"
					_ = aiRepo.UpdateTask(req.task)
				}
				if req.instruction != nil {
					req.instruction.Status = "running"
					_ = sessionRepo.UpdateInstruction(req.instruction)
				}
				if currentSession != nil {
					now := time.Now()
					currentSession.Status = "active"
					currentSession.CurrentStage = "executing"
					currentSession.LastInstructionAt = &now
					if req.task != nil {
						currentSession.LastTaskID = req.task.ID
					}
					_ = sessionRepo.UpdateSession(currentSession)
					createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
						SessionID: currentSession.ID,
						InstructionID: func() uint {
							if req.instruction != nil {
								return req.instruction.ID
							}
							return 0
						}(),
						TaskID: func() uint {
							if req.task != nil {
								return req.task.ID
							}
							return 0
						}(),
						EventType: "execution_started",
						Stage:     "executing",
						Title:     "开始执行开发任务",
						Content:   buildTimelineContent(req.input),
						Status:    "running",
					})
				}
				sendWsMsg("\033[36m[AI Agent] 正在思考并执行...\033[0m\r\n")
				cmdArgs, err := buildAIAgentExecArgs(containerName, agentName, req.input)
				out := []byte{}
				if err == nil {
					execCmd, cmdErr := docker.RuntimeCommand(context.Background(), cmdArgs...)
					if cmdErr != nil {
						err = cmdErr
					} else {
						out, err = execCmd.CombinedOutput()
					}
				}
				if err != nil {
					global.LOG.Errorf("AI execution error: %v, out: %s", err, string(out))
					if len(out) == 0 {
						out = []byte(fmt.Sprintf("执行错误: %v", err))
					}
				}
				if req.task != nil {
					_ = aiRepo.CreateMessage(&model.AIMessage{TaskID: req.task.ID, Role: "agent", Content: string(out)})
				}
				previews, previewErr := upsertAIPreviews(sessionRepo, currentSession, req.task, req.instruction, string(out))
				if previewErr != nil {
					global.LOG.Errorf("Failed to upsert AI previews: %v", previewErr)
				}
				execFailed := err != nil
				if currentSession != nil {
					resultStage := "completed"
					resultTitle := "开发任务已完成"
					resultStatus := "success"
					if execFailed {
						resultStage = "failed"
						resultTitle = "开发任务执行失败"
						resultStatus = "error"
					} else if len(previews) > 0 {
						resultStage = "preview_ready"
						resultTitle = "开发预览已生成"
					}
					createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
						SessionID: currentSession.ID,
						InstructionID: func() uint {
							if req.instruction != nil {
								return req.instruction.ID
							}
							return 0
						}(),
						TaskID: func() uint {
							if req.task != nil {
								return req.task.ID
							}
							return 0
						}(),
						EventType: "execution_result",
						Stage:     resultStage,
						Title:     resultTitle,
						Content:   summarizeAIRecentOutput(string(out)),
						Status:    resultStatus,
					})
					for _, preview := range previews {
						if preview == nil {
							continue
						}
						createAITimelineEvent(sessionRepo, &model.AITimelineEvent{
							SessionID: currentSession.ID,
							InstructionID: func() uint {
								if req.instruction != nil {
									return req.instruction.ID
								}
								return 0
							}(),
							TaskID:    preview.TaskID,
							EventType: "preview_ready",
							Stage:     "preview_ready",
							Title:     "预览已生成",
							Content:   buildTimelineContent(preview.URL),
							Status:    "success",
						})
					}
				}
				if req.task != nil {
					if execFailed {
						req.task.Status = "failed"
					} else {
						req.task.Status = "completed"
					}
					_ = aiRepo.UpdateTask(req.task)
				}
				if req.instruction != nil {
					if execFailed {
						req.instruction.Status = "failed"
					} else {
						req.instruction.Status = "completed"
					}
					_ = sessionRepo.UpdateInstruction(req.instruction)
				}
				if currentSession != nil {
					if execFailed {
						currentSession.CurrentStage = "failed"
					} else if len(previews) > 0 {
						currentSession.CurrentStage = "preview_ready"
					} else {
						currentSession.CurrentStage = "completed"
					}
					_ = sessionRepo.UpdateSession(currentSession)
				}
				formattedOut := strings.ReplaceAll(string(out), "\n", "\r\n")
				if !strings.HasSuffix(formattedOut, "\r\n") {
					formattedOut += "\r\n"
				}
				sendWsMsg(formattedOut)
				sendWsMsg("\033[32m[AI Agent] > \033[0m")
			}
		}
	}()
	startAIAgentPTYForwarder(
		ptmx,
		sendWsMsg,
		enqueueAIExecution,
		sessionRepo,
		currentSession,
		&currentTask,
		authClaims,
		userID,
		&inAIChatMode,
		&pendingInstructionsLoaded,
	)
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
			switch msg.Type {
			case "cmd":
				if inAIChatMode {
					if msg.Data == "\x7f" || msg.Data == "\b" {
						if aiInputBuffer.Len() > 0 {
							str := aiInputBuffer.String()
							aiInputBuffer.Reset()
							aiInputBuffer.WriteString(str[:len(str)-1])
							sendWsMsg("\b \b")
						}
						continue
					}
					if msg.Data == "\r" || msg.Data == "\n" {
						sendWsMsg("\r\n")
						userInput := strings.TrimSpace(aiInputBuffer.String())
						aiInputBuffer.Reset()
						if userInput == "exit" || userInput == "quit" {
							inAIChatMode = false
							sendWsMsg("\033[33m[系统] 已退出 AI 交互模式，恢复 Shell 环境。\033[0m\r\n")
							_, _ = ptmx.Write([]byte("\r"))
							continue
						}
						if userInput != "" {
							if currentTask == nil {
								title := userInput
								if len([]rune(title)) > 20 {
									title = string([]rune(title)[:20]) + "..."
								}
								if agentName == "" {
									agentName = "trae"
								}
								newTask := &model.AITask{UserID: userID, ProjectID: uint(reqProjectID), Title: title, AgentName: agentName, WorkDir: workDir, Status: "active"}
								if err := aiRepo.CreateTask(newTask); err == nil {
									currentTask = newTask
									if currentSession != nil {
										currentSession.LastTaskID = currentTask.ID
										currentSession.CurrentStage = "interactive"
										_ = sessionRepo.UpdateSession(currentSession)
									}
									sendWsRaw(fmt.Sprintf(`{"type":"meta","task_id":%d}`, currentTask.ID))
								}
							}
							if currentTask != nil {
								_ = aiRepo.CreateMessage(&model.AIMessage{TaskID: currentTask.ID, Role: "user", Content: userInput})
							}
							enqueueAIExecution(aiExecRequest{input: userInput, task: currentTask})
							continue
						}
						sendWsMsg("\033[32m[AI Agent] > \033[0m")
					} else {
						aiInputBuffer.WriteString(msg.Data)
						sendWsMsg(msg.Data)
					}
				} else {
					_, _ = ptmx.Write([]byte(msg.Data))
				}
			case "resize":
				type resizeData struct {
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				var r resizeData
				if err := json.Unmarshal([]byte( // 处理终端大小调整
					msg.Data), &r); err == nil {
					_ = pty.Setsize(ptmx, &pty.Winsize{Rows: r.Rows, Cols: r.Cols})
				}
			case "ping":
				_ = wsConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			}
		}
	}
	dt := time.Now().Add(time.Second)
	_ = wsConn.WriteControl(websocket.CloseMessage, nil, dt)
}
