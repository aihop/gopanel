package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type codexAppServerMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type codexAppServerClient struct {
	ctx       context.Context
	command   *exec.Cmd
	stdin     io.WriteCloser
	reader    *bufio.Scanner
	output    *boundedCodeOutput
	sessionID uint
	threadID  string
	text      string
	itemID    string
	status    string
	requestID int
	usage     codexTokenUsage
}

func buildCodexAppServerCommand(ctx context.Context, workDir string, session *model.AIDevSession) (*exec.Cmd, error) {
	commandPath, commandEnv, err := resolveCodeExecutorCommand("codex")
	if err != nil {
		return nil, err
	}
	approvalPolicy := ""
	if session != nil {
		approvalPolicy = session.ApprovalPolicy
	}
	args := append(codexSandboxArgs(approvalPolicy), "app-server")
	if session != nil {
		writableDirs, writableErr := codexWritableDirsForSessionWithRepair(session)
		if writableErr != nil {
			return nil, writableErr
		}
		args = addCodexWritableDirArgs(args, writableDirs)
	}
	command := exec.CommandContext(ctx, commandPath, args...)
	command.Dir = workDir
	command.Env = commandEnv
	if err := configureCodeProviderCommand("codex", command, session); err != nil {
		return nil, err
	}
	return command, nil
}

func executeCodexAppServer(
	ctx context.Context,
	command *exec.Cmd,
	sessionID uint,
	nativeSessionID string,
	prompt string,
) (codeExecutorOutput, error) {
	stdout, err := command.StdoutPipe()
	if err != nil {
		return codeExecutorOutput{}, err
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		return codeExecutorOutput{}, err
	}
	output := &boundedCodeOutput{}
	command.Stderr = output
	if err := command.Start(); err != nil {
		return codeExecutorOutput{}, err
	}
	client := &codexAppServerClient{
		ctx:       ctx,
		command:   command,
		stdin:     stdin,
		reader:    bufio.NewScanner(stdout),
		output:    output,
		sessionID: sessionID,
		threadID:  strings.TrimSpace(nativeSessionID),
		requestID: 1,
	}
	client.reader.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result, runErr := client.run(prompt)
	_ = stdin.Close()
	waitDone := make(chan error, 1)
	go func() { waitDone <- command.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-ctx.Done():
		_ = command.Process.Kill()
		waitErr = <-waitDone
	case <-time.After(2 * time.Second):
		_ = command.Process.Kill()
		waitErr = <-waitDone
	}
	if runErr == nil && waitErr != nil && client.status == "" {
		runErr = waitErr
	}
	result.RawOutput = string(output.Bytes())
	result.NativeSessionID = client.threadID
	result.Message = client.text
	result.InputTokens = client.usage.InputTokens
	result.OutputTokens = client.usage.OutputTokens
	result.CachedInputTokens = client.usage.CachedInputTokens
	result.ReasoningTokens = client.usage.ReasoningOutputTokens
	result.TotalTokens = client.usage.TotalTokens
	result.TokenUsageReported = result.TotalTokens > 0 || result.InputTokens > 0 || result.OutputTokens > 0
	if result.TotalTokens == 0 {
		result.TotalTokens = result.InputTokens + result.OutputTokens
	}
	return result, runErr
}

func (client *codexAppServerClient) run(prompt string) (codeExecutorOutput, error) {
	if _, err := client.request("initialize", map[string]any{
		"clientInfo": map[string]string{"name": "gopanel", "title": "GoPanel Code", "version": "1"},
	}); err != nil {
		return codeExecutorOutput{}, err
	}
	if err := client.notify("initialized", map[string]any{}); err != nil {
		return codeExecutorOutput{}, err
	}
	if err := client.openThread(); err != nil {
		return codeExecutorOutput{}, err
	}
	if _, err := client.request("turn/start", map[string]any{
		"threadId": client.threadID,
		"cwd":      client.command.Dir,
		"input":    []map[string]string{{"type": "text", "text": prompt}},
	}); err != nil {
		return codeExecutorOutput{}, err
	}
	for client.status == "" && client.reader.Scan() {
		message, err := client.parseLine(client.reader.Bytes())
		if err != nil {
			return codeExecutorOutput{}, err
		}
		if isCodexServerRequest(message) {
			if err := client.respondToApproval(message); err != nil {
				return codeExecutorOutput{}, err
			}
		}
		if client.status != "" {
			break
		}
	}
	if err := client.reader.Err(); err != nil {
		return codeExecutorOutput{}, err
	}
	if client.status == "" {
		if client.ctx.Err() != nil {
			return codeExecutorOutput{}, client.ctx.Err()
		}
		return codeExecutorOutput{}, io.ErrUnexpectedEOF
	}
	if client.status == "failed" {
		return codeExecutorOutput{}, errors.New("Codex app-server turn failed")
	}
	if client.status == "interrupted" && client.ctx.Err() != nil {
		return codeExecutorOutput{}, client.ctx.Err()
	}
	return codeExecutorOutput{}, nil
}

func (client *codexAppServerClient) openThread() error {
	params := map[string]any{"cwd": client.command.Dir, "sandbox": "workspace-write"}
	if client.threadID != "" {
		if _, err := client.request("thread/resume", map[string]any{"threadId": client.threadID, "cwd": client.command.Dir}); err == nil {
			return nil
		} else if !isCodexThreadNotFound(err) {
			return err
		}
	}
	result, err := client.request("thread/start", params)
	if err != nil {
		return err
	}
	var payload struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &payload); err != nil || payload.Thread.ID == "" {
		return errors.New("Codex app-server 未返回会话 ID")
	}
	client.threadID = payload.Thread.ID
	return nil
}

func isCodexThreadNotFound(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") || strings.Contains(message, "unknown thread") || strings.Contains(message, "no such")
}

func (client *codexAppServerClient) request(method string, params any) (json.RawMessage, error) {
	id := client.requestID
	client.requestID++
	if err := client.writeMessage(map[string]any{"method": method, "id": id, "params": params}); err != nil {
		return nil, err
	}
	for client.reader.Scan() {
		message, err := client.parseLine(client.reader.Bytes())
		if err != nil {
			return nil, err
		}
		if isCodexServerRequest(message) {
			if err := client.respondToApproval(message); err != nil {
				return nil, err
			}
			continue
		}
		if message.ID != nil && string(message.ID) == fmt.Sprintf("%d", id) {
			if message.Error != nil {
				return nil, errors.New(message.Error.Message)
			}
			return message.Result, nil
		}
	}
	if err := client.reader.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (client *codexAppServerClient) notify(method string, params any) error {
	return client.writeMessage(map[string]any{"method": method, "params": params})
}

func (client *codexAppServerClient) writeMessage(message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = client.stdin.Write(data)
	return err
}

func (client *codexAppServerClient) parseLine(line []byte) (codexAppServerMessage, error) {
	if _, err := client.output.Write(append(line, '\n')); err != nil {
		return codexAppServerMessage{}, err
	}
	var message codexAppServerMessage
	if err := json.Unmarshal(line, &message); err != nil {
		return codexAppServerMessage{}, err
	}
	if message.Method != "" {
		if err := client.handleNotification(message.Method, message.Params); err != nil {
			return codexAppServerMessage{}, err
		}
	}
	return message, nil
}

func (client *codexAppServerClient) handleNotification(method string, params json.RawMessage) error {
	switch method {
	case "item/agentMessage/delta":
		var payload struct {
			Delta  string `json:"delta"`
			ItemID string `json:"itemId"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return err
		}
		if payload.Delta != "" {
			delta := payload.Delta
			if client.itemID != "" && payload.ItemID != "" && payload.ItemID != client.itemID {
				delta = "\n\n" + delta
			}
			client.itemID = payload.ItemID
			client.text += delta
			codeConversationStreams.SetText(client.sessionID, client.text, delta)
		}
	case "item/completed":
		var payload struct {
			Item struct {
				ID      string `json:"id"`
				Type    string `json:"type"`
				Text    string `json:"text"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"item"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return err
		}
		if payload.Item.Type == "agentMessage" && (client.itemID == "" || payload.Item.ID != client.itemID) {
			text := payload.Item.Text
			if text == "" && len(payload.Item.Content) > 0 {
				for _, part := range payload.Item.Content {
					text += part.Text
				}
			}
			if text != "" {
				if client.text != "" {
					client.text += "\n\n"
				}
				client.itemID = payload.Item.ID
				client.text += text
				codeConversationStreams.SetText(client.sessionID, client.text, "")
			}
		}
	case "thread/tokenUsage/updated":
		var payload struct {
			TokenUsage struct {
				Total struct {
					InputTokens           int64 `json:"inputTokens"`
					CachedInputTokens     int64 `json:"cachedInputTokens"`
					OutputTokens          int64 `json:"outputTokens"`
					ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
					TotalTokens           int64 `json:"totalTokens"`
				} `json:"total"`
			} `json:"tokenUsage"`
		}
		if err := json.Unmarshal(params, &payload); err == nil {
			client.usage = codexTokenUsage{
				InputTokens:           payload.TokenUsage.Total.InputTokens,
				CachedInputTokens:     payload.TokenUsage.Total.CachedInputTokens,
				OutputTokens:          payload.TokenUsage.Total.OutputTokens,
				ReasoningOutputTokens: payload.TokenUsage.Total.ReasoningOutputTokens,
				TotalTokens:           payload.TokenUsage.Total.TotalTokens,
			}
		}
	case "turn/completed":
		var payload struct {
			Turn struct {
				Status string `json:"status"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"turn"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return err
		}
		client.status = payload.Turn.Status
		if payload.Turn.Error != nil && payload.Turn.Error.Message != "" {
			return errors.New(payload.Turn.Error.Message)
		}
	default:
	}
	return nil
}

func isCodexServerRequest(message codexAppServerMessage) bool {
	return message.ID != nil && len(message.ID) > 0 && message.Method != ""
}

func (client *codexAppServerClient) respondToApproval(message codexAppServerMessage) error {
	var result map[string]any
	switch message.Method {
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		result = map[string]any{"decision": "decline"}
	case "item/permissions/requestApproval":
		result = map[string]any{"permissions": map[string]any{}}
	case "item/tool/requestUserInput":
		result = map[string]any{"answers": map[string]any{}}
	case "mcpServer/elicitation/request":
		result = map[string]any{"action": "decline"}
	default:
		result = map[string]any{"success": false, "contentItems": []any{}}
	}
	return client.writeMessage(map[string]any{"id": json.RawMessage(message.ID), "result": result})
}
