package api

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
)

type aiExecRequest struct {
	input       string
	task        *model.AITask
	instruction *model.AIInstruction
}

func startAIAgentPTYForwarder(
	ptmx *os.File,
	sendWsMsg func(string),
	enqueueAIExecution func(aiExecRequest),
	sessionRepo repo.IAIDevSessionRepo,
	currentSession *model.AIDevSession,
	currentTaskRef **model.AITask,
	authClaims *token.CustomClaims,
	userID uint,
	inAIChatMode *bool,
	pendingInstructionsLoaded *bool,
) {
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
			if outputStr == "" {
				continue
			}

			if strings.Contains(outputStr, "[CX-AI-HOOK:START-INTERACTIVE]") {
				*inAIChatMode = true
				welcomeMsg := "\r\n\033[32m[系统] GoPanel AI 引擎已接管终端。\033[0m\r\n" +
					"\033[36m现在您可以直接用自然语言与 AI 对话了。(输入 'exit' 退出聊天)\033[0m\r\n\033[32m[AI Agent] > \033[0m"
				outputStr = strings.ReplaceAll(outputStr, "\033[36m[CX-AI-HOOK:START-INTERACTIVE]\033[0m", welcomeMsg)
				outputStr = strings.ReplaceAll(outputStr, "[CX-AI-HOOK:START-INTERACTIVE]", "")

				if currentSession != nil && !*pendingInstructionsLoaded {
					*pendingInstructionsLoaded = true
					go func(session *model.AIDevSession) {
						instructions, err := sessionRepo.GetPendingInstructionsBySessionID(session.ID)
						if err != nil {
							sendWsMsg(fmt.Sprintf("\r\n\033[33m[系统] 读取待执行指令失败: %s\033[0m\r\n", err.Error()))
							return
						}
						if len(instructions) == 0 {
							return
						}

						if *currentTaskRef == nil {
							claims := authClaims
							if claims == nil {
								claims = &token.CustomClaims{UserId: userID}
							}
							if task, taskErr := ensureSessionTask(session, claims, instructions[0].Content); taskErr == nil {
								*currentTaskRef = task
							}
						}
						if *currentTaskRef == nil {
							sendWsMsg("\r\n\033[33m[系统] 当前会话未能恢复任务，暂时无法自动执行待处理指令。\033[0m\r\n")
							return
						}

						sendWsMsg(fmt.Sprintf("\r\n\033[36m[系统] 检测到 %d 条待执行开发指令，开始自动执行。\033[0m\r\n", len(instructions)))
						for _, instruction := range instructions {
							enqueueAIExecution(aiExecRequest{
								input:       instruction.Content,
								task:        *currentTaskRef,
								instruction: instruction,
							})
						}
					}(currentSession)
				}
			}

			if strings.Contains(outputStr, "[CX-AI-HOOK:ONE-SHOT]") {
				outputStr = strings.ReplaceAll(outputStr, "[CX-AI-HOOK:ONE-SHOT]", "[系统] 已拦截单次 AI 任务：")
			}

			if *inAIChatMode {
				if strings.Contains(outputStr, "/workspace #") || strings.Contains(outputStr, "/ #") {
					outputStr = strings.ReplaceAll(outputStr, "/workspace # ", "")
					outputStr = strings.ReplaceAll(outputStr, "/workspace #", "")
					outputStr = strings.ReplaceAll(outputStr, "/ # ", "")
					outputStr = strings.ReplaceAll(outputStr, "/ #", "")
				}
				if strings.TrimSpace(outputStr) == "" {
					continue
				}
			}

			if strings.TrimSpace(outputStr) == "" || outputStr == "" {
				continue
			}

			sendWsMsg(outputStr)
		}
	}()
}
