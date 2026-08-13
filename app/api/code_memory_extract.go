package api

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 抽取要读整段会话记录、发一次模型请求，代价不小，因此按会话去重：
// 同一会话已在抽取时再次触发直接忽略，等这次结束后的下一轮再说。
var codeMemoryExtracting sync.Map

// 参与抽取的会话消息条数上限。太少抓不到结论，太多会把早期的试错也带进去。
const codeMemoryTranscriptMaxMessages = 60

// enqueueCodeMemoryExtraction 在后台抽取一次会话记忆。
//
// 不阻塞调用方：抽取是"顺手做的沉淀"，失败了不该影响任务本身的结果。
// 也正因如此，所有错误都只记日志，不向上冒泡。
func enqueueCodeMemoryExtraction(sessionID uint, trigger string, force bool) bool {
	if sessionID == 0 || global.DB == nil {
		return false
	}
	if _, loaded := codeMemoryExtracting.LoadOrStore(sessionID, struct{}{}); loaded {
		return false
	}
	if err := queueCodeMemoryExtractionStatus(sessionID, trigger); err != nil {
		codeMemoryExtracting.Delete(sessionID)
		warnCodeDelivery("Queue Code memory extraction for session %d failed: %v", sessionID, err)
		return false
	}
	go func() {
		defer codeMemoryExtracting.Delete(sessionID)
		if err := startCodeMemoryExtractionStatus(sessionID, trigger); err != nil {
			warnCodeDelivery("Start Code memory extraction for session %d failed: %v", sessionID, err)
			return
		}
		result, lastMessageID, err := runCodeMemoryExtraction(context.Background(), sessionID, force)
		status := codeMemoryExtractionSuccess
		reason := ""
		if err != nil {
			status = codeMemoryExtractionFailed
			reason = codeMemoryExtractionReasonFailed
			var skip codeMemorySkipError
			if errors.As(err, &skip) {
				status = codeMemoryExtractionSkipped
				reason = err.Error()
			}
			warnCodeDelivery("Extract Code memory for session %d did not complete: %v", sessionID, err)
		}
		if statusErr := finishCodeMemoryExtractionStatus(sessionID, status, reason, result, lastMessageID); statusErr != nil {
			warnCodeDelivery("Finish Code memory extraction status for session %d failed: %v", sessionID, statusErr)
		}
	}()
	return true
}

func runCodeMemoryExtraction(ctx context.Context, sessionID uint, force bool) (codeMemoryApplyResult, uint, error) {
	result := codeMemoryApplyResult{}
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		return result, 0, err
	}
	account, config, threshold, err := resolveCodeMemoryExtraction(session.UserID)
	if err != nil {
		if errors.Is(err, errCodeMemoryExtractionDisabled) {
			return result, 0, codeMemorySkipError{reason: "disabled"}
		}
		return result, 0, err
	}
	transcript, lastMessageID, newMessages, err := loadCodeMemoryTranscript(sessionID)
	if err != nil {
		return result, 0, err
	}
	if strings.TrimSpace(transcript) == "" {
		return result, 0, codeMemorySkipError{reason: "empty_transcript"}
	}
	// 闸门在模型调用之前：挡掉的这些本来就抽不出东西，
	// 花一次调用去确认它没东西可抽是纯粹的浪费。
	if !force {
		if gate := evaluateCodeMemoryGate(sessionID, transcript, threshold, newMessages); gate != codeMemoryGateAllow {
			return result, 0, codeMemorySkipError{reason: gate}
		}
	}
	userEntries, projectEntries, summary := loadCodeMemoryForPrompt(session.UserID, session.ProjectID)
	prompt := buildCodeMemoryExtractionPrompt(codeMemoryPromptInput{
		Transcript:      transcript,
		ProjectName:     codeMemoryProjectName(session.ProjectID),
		UserSummary:     summary,
		UserMemories:    userEntries,
		ProjectMemories: projectEntries,
		OutputLanguage:  "Chinese",
	})
	// 参数按账号保存时探测出的能力发：用户给账号定了推理强度但模型不支持时，
	// 正确的行为是不发，而不是发出去被 400——后台抽取失败只记日志。
	raw, err := callCodeMemoryLLMWithOptions(ctx, config, []codeMemoryChatMessage{
		{Role: "system", Content: codeMemoryExtractionSystemPrompt},
		{Role: "user", Content: prompt},
	}, codeMemoryOptionsForAccount(account, codeMemoryExtractionSchema()))
	if err != nil {
		return result, 0, err
	}
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		return result, 0, err
	}
	result, err = applyCodeMemoryExtraction(response, codeMemoryApplyContext{
		UserID: session.UserID, ProjectID: session.ProjectID, SessionID: sessionID,
	})
	if err != nil {
		return result, 0, err
	}
	return result, lastMessageID, nil
}

func codeMemoryProjectName(projectID uint) string {
	if projectID == 0 || global.DB == nil {
		return ""
	}
	var name string
	_ = global.DB.Model(&model.AIProject{}).Where("id = ?", projectID).Pluck("name", &name).Error
	return name
}

// loadCodeMemoryTranscript 取会话记录，并算出距上次抽取新增了多少条。
//
// 返回的 newMessages 为 -1 表示这个会话还没抽过，调用方据此跳过增量判断。
// 仍然取最近 60 条而不是只取新增：只喂新增的话，模型会丢掉「这段对话在讨论什么」
// 的上下文，抽出来的记忆会变成没有主语的碎片。
func loadCodeMemoryTranscript(sessionID uint) (string, uint, int, error) {
	var messages []model.AIMessage
	if err := global.DB.Where("session_id = ?", sessionID).
		Order("id DESC").Limit(codeMemoryTranscriptMaxMessages).
		Find(&messages).Error; err != nil {
		return "", 0, 0, err
	}
	if len(messages) == 0 {
		return "", 0, 0, nil
	}
	// 按 id 倒序取的，第一条就是最新的那条。
	lastMessageID := messages[0].ID
	newMessages := -1
	if lastExtracted := loadCodeMemoryExtractionState(sessionID); lastExtracted >= 0 {
		newMessages = 0
		for _, message := range messages {
			if int64(message.ID) > lastExtracted {
				newMessages++
			}
		}
	}
	transcript := make([]codeMemoryTranscriptMessage, 0, len(messages))
	// 查出来是倒序的，翻回时间顺序再拼。
	for index := len(messages) - 1; index >= 0; index-- {
		transcript = append(transcript, codeMemoryTranscriptMessage{
			Role: messages[index].Role, Content: messages[index].Content,
		})
	}
	return buildCodeMemoryTranscript(transcript), lastMessageID, newMessages, nil
}

// loadCodeMemoryForPrompt 取已有记忆喂给抽取，让模型能判重。
// 不给的话每轮都会把同一条偏好重新写一遍，几轮之后记忆库全是同义重复。
func loadCodeMemoryForPrompt(userID, projectID uint) ([]codeMemoryPromptEntry, []codeMemoryPromptEntry, string) {
	entries, summary := loadCodeMemoryForInjection(userID, projectID)
	userEntries := make([]codeMemoryPromptEntry, 0, len(entries))
	projectEntries := make([]codeMemoryPromptEntry, 0, len(entries))
	for _, entry := range entries {
		prompt := codeMemoryPromptEntry{
			ID: entry.ID, Scope: entry.Scope, Kind: entry.Kind,
			Tier: entry.Tier, ModuleKey: entry.ModuleKey, Content: entry.Content,
		}
		if entry.Scope == codeMemoryScopeUser {
			userEntries = append(userEntries, prompt)
			continue
		}
		projectEntries = append(projectEntries, prompt)
	}
	return userEntries, projectEntries, summary
}
