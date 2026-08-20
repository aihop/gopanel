package api

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

const nativeHistoryDuplicateWindow = 5 * time.Minute

// 每个进程内每个会话只补拉一次原生历史。
// 补拉要读并解析磁盘上的 rollout 文件，而终端重连很频繁（断网、切任务、刷新页面都会重连），
// 不去重就变成每次重连都全量解析一遍。进程重启后标记自然清空，正好对应「重启后要补一次」。
var recoveredNativeHistorySessions sync.Map

// recoverNativeCodeHistoryOnce 供终端连接时调用：只在本进程内第一次连接该会话时补拉。
//
// 面板重启会丢掉内存里的 PTY 缓冲，但执行器写在磁盘上的原生对话还在。
// 以前只有「查看完整对话」接口会把它补回数据库，用户不点那里就以为记录全没了。
func recoverNativeCodeHistoryOnce(session *model.AIDevSession) {
	if session == nil || session.ID == 0 || !supportsNativeCodeHistory(session.AgentName) {
		return
	}
	if _, loaded := recoveredNativeHistorySessions.LoadOrStore(session.ID, struct{}{}); loaded {
		return
	}
	if _, err := recoverNativeCodeHistory(session); err != nil {
		// 补拉失败不能影响终端连接：历史是附加能力，终端本身要照常可用。
		recoveredNativeHistorySessions.Delete(session.ID)
		global.LOG.Warnf("Recover native %s history for session %d failed: %v", session.AgentName, session.ID, err)
	}
}

func supportsNativeCodeHistory(executorID string) bool {
	return executorID == "codex" || executorID == "grok" || executorID == "claude" || executorID == "opencode"
}

// recoverNativeCodeHistory 把执行器留在磁盘上的原生对话补回数据库，并返回这批消息。
//
// 三步缺一不可：
//  1. 修绑定 —— 交付完成后 session.WorkDir 会被改写成源仓路径，
//     而 rollout 记录的是当初的隔离 Worktree，不修就永久失联；
//  2. 读盘 —— 原生历史一直在 ~/.codex 等目录下，进程重启不会丢；
//  3. 落库 —— 写进 ai_messages，之后不依赖磁盘文件也能查。
//
// 这套流程原先只长在「查看完整对话」接口里，所以面板重启后不点开那个入口
// 就永远补不回来。抽出来给终端打开时也能调用。
func recoverNativeCodeHistory(session *model.AIDevSession) ([]*model.AIMessage, error) {
	if session == nil || !supportsNativeCodeHistory(session.AgentName) {
		return nil, nil
	}
	switch session.AgentName {
	case "codex":
		if err := repairNativeCodexSessionBinding(session); err != nil {
			return nil, err
		}
	case "opencode":
		if err := repairNativeOpenCodeSessionBinding(session); err != nil {
			return nil, err
		}
	}
	messages, err := getNativeCodeMessages(session)
	if err != nil {
		return nil, err
	}
	if err := persistNativeCodexMessages(session.ID, messages); err != nil {
		// 落库失败不该让调用方拿不到历史：磁盘上的内容仍然是可用的。
		global.LOG.Warnf("Persist native %s history for session %d failed: %v", session.AgentName, session.ID, err)
	}
	return messages, nil
}

func getNativeCodeMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	if session == nil {
		return nil, nil
	}
	switch session.AgentName {
	case "codex":
		return getNativeCodexMessages(session)
	case "grok":
		return getNativeGrokMessages(session)
	case "claude":
		return getNativeClaudeMessages(session)
	case "opencode":
		return getNativeOpenCodeMessages(session)
	default:
		return nil, nil
	}
}

func getNativeCodexMessages(session *model.AIDevSession) ([]*model.AIMessage, error) {
	if session == nil || strings.TrimSpace(session.NativeSessionID) == "" {
		return nil, nil
	}
	path := findCodexRuntimePath(session)
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseNativeCodexMessages(file, session.ID, session.LastTaskID)
}

// parseNativeCodexMessages 兼容 codex 两代 rollout 事件格式：
//   - 旧版：event_msg 下 user_message / agent_message 各自独立，正文是字符串。
//   - 新版：response_item 下统一为 message，用 role 区分，正文是 [{type,text}] 数组。
//
// 只认旧格式会让升级过 codex 之后的会话历史整段读不出来，而且是静默的。
func parseNativeCodexMessages(reader io.Reader, sessionID, taskID uint) ([]*model.AIMessage, error) {
	messages := make([]*model.AIMessage, 0)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event codexRuntimeEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		role, content := nativeCodexMessageFromEvent(&event)
		if role == "" || content == "" {
			continue
		}
		createdAt := parseCodexEventTime(event.Timestamp)
		messages = append(messages, &model.AIMessage{
			CreatedAt: createdAt,
			SessionID: sessionID,
			TaskID:    taskID,
			Role:      role,
			Content:   content,
			NativeID:  nativeCodexMessageID(event.Payload.ID, role, event.Timestamp, content),
		})
	}
	return messages, scanner.Err()
}

// nativeCodexMessageFromEvent 把一条 rollout 事件归一化成 (role, content)。
// 无法识别或不该展示的事件返回空 role。
func nativeCodexMessageFromEvent(event *codexRuntimeEvent) (string, string) {
	switch event.Type {
	case "event_msg":
		switch event.Payload.Type {
		case "user_message":
			return "user", nativeCodexMessageText(event.Payload.Message)
		case "agent_message":
			return "agent", nativeCodexMessageText(event.Payload.Message)
		}
	case "response_item":
		if event.Payload.Type != "message" {
			return "", ""
		}
		// developer 是注入的系统提示，不属于用户可见对话。
		switch event.Payload.Role {
		case "user":
			return "user", nativeCodexContentText(event.Payload.Content)
		case "assistant":
			return "agent", nativeCodexContentText(event.Payload.Content)
		}
	}
	return "", ""
}

// nativeCodexContentText 提取新版 message 的正文：content 是 [{type,text}] 数组。
func nativeCodexContentText(value json.RawMessage) string {
	var items []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(value, &items) != nil {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(item.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// nativeCodexMessageID 给一条原生消息生成稳定标识。
// 新版 rollout 自带 payload.id；旧版没有，用 role+时间戳+正文摘要兜底，
// 保证同一条消息反复读取时得到同一个 ID，不会重复入库。
func nativeCodexMessageID(payloadID, role, timestamp, content string) string {
	if id := strings.TrimSpace(payloadID); id != "" {
		return id
	}
	sum := sha256.Sum256([]byte(role + "\x00" + timestamp + "\x00" + content))
	return "sha:" + hex.EncodeToString(sum[:16])
}

// persistNativeCodexMessages 把原生历史增量固化进库。
// 一旦落库，rollout 文件被清理或 codex 再次变更事件格式都不会让历史消失。
func persistNativeCodexMessages(sessionID uint, messages []*model.AIMessage) error {
	if sessionID == 0 || len(messages) == 0 || global.DB == nil {
		return nil
	}
	var existingMessages []*model.AIMessage
	if err := global.DB.Where("session_id = ?", sessionID).Find(&existingMessages).Error; err != nil {
		return err
	}
	existingIDs := make([]string, 0, len(existingMessages))
	for _, existing := range existingMessages {
		if existing != nil && strings.TrimSpace(existing.NativeID) != "" {
			existingIDs = append(existingIDs, existing.NativeID)
		}
	}
	known := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		known[id] = struct{}{}
	}
	pending := make([]*model.AIMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil || strings.TrimSpace(message.NativeID) == "" {
			continue
		}
		if _, exists := known[message.NativeID]; exists {
			continue
		}
		if isDuplicateHistoryMessage(existingMessages, message) {
			continue
		}
		known[message.NativeID] = struct{}{}
		stored := *message
		stored.ID = 0
		pending = append(pending, &stored)
		existingMessages = append(existingMessages, &stored)
	}
	if len(pending) == 0 {
		return nil
	}
	return global.DB.CreateInBatches(pending, 200).Error
}

func nativeCodexMessageText(value json.RawMessage) string {
	var text string
	if json.Unmarshal(value, &text) != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

func mergeCodeHistoryMessages(databaseMessages, nativeMessages []*model.AIMessage) []*model.AIMessage {
	merged := append([]*model.AIMessage{}, databaseMessages...)
	for _, nativeMessage := range nativeMessages {
		if nativeMessage == nil {
			continue
		}
		if nativeMessage.Role == "user" && stripInjectedConversationPrompt(nativeMessage.Content) == "" {
			continue
		}
		if isDuplicateHistoryMessage(databaseMessages, nativeMessage) {
			continue
		}
		merged = append(merged, nativeMessage)
	}
	sort.SliceStable(merged, func(left, right int) bool {
		return merged[left].CreatedAt.Before(merged[right].CreatedAt)
	})
	var nextID uint
	for _, message := range merged {
		if message.ID > nextID {
			nextID = message.ID
		}
	}
	for _, message := range merged {
		if message.ID == 0 {
			nextID++
			message.ID = nextID
		}
	}
	return merged
}

func isDuplicateHistoryMessage(databaseMessages []*model.AIMessage, candidate *model.AIMessage) bool {
	for _, message := range databaseMessages {
		if message == nil || message.Role != candidate.Role {
			continue
		}
		if stripInjectedConversationPrompt(message.Content) != stripInjectedConversationPrompt(candidate.Content) {
			continue
		}
		if message.Role != "user" {
			return true
		}
		if message.CreatedAt.IsZero() || candidate.CreatedAt.IsZero() {
			return true
		}
		difference := message.CreatedAt.Sub(candidate.CreatedAt)
		if difference < 0 {
			difference = -difference
		}
		if difference <= nativeHistoryDuplicateWindow {
			return true
		}
	}
	return false
}
