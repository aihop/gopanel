package api

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
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
func enqueueCodeMemoryExtraction(sessionID uint) {
	if sessionID == 0 || global.DB == nil {
		return
	}
	if _, loaded := codeMemoryExtracting.LoadOrStore(sessionID, struct{}{}); loaded {
		return
	}
	go func() {
		defer codeMemoryExtracting.Delete(sessionID)
		if err := runCodeMemoryExtraction(context.Background(), sessionID); err != nil {
			warnCodeDelivery("Extract Code memory for session %d skipped: %v", sessionID, err)
		}
	}()
}

func runCodeMemoryExtraction(ctx context.Context, sessionID uint) error {
	var session model.AIDevSession
	if err := global.DB.First(&session, sessionID).Error; err != nil {
		return err
	}
	config, err := codeMemoryLLMConfigForSession(&session)
	if err != nil {
		return err
	}
	transcript, err := loadCodeMemoryTranscript(sessionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(transcript) == "" {
		return errors.New("会话没有可抽取的内容")
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
	raw, err := callCodeMemoryLLM(ctx, config, []codeMemoryChatMessage{
		{Role: "system", Content: codeMemoryExtractionSystemPrompt},
		{Role: "user", Content: prompt},
	}, codeMemoryExtractionSchema())
	if err != nil {
		return err
	}
	response, err := parseCodeMemoryExtractionResponse(raw)
	if err != nil {
		return err
	}
	_, err = applyCodeMemoryExtraction(response, codeMemoryApplyContext{
		UserID: session.UserID, ProjectID: session.ProjectID, SessionID: sessionID,
	})
	return err
}

// codeMemoryLLMConfigForSession 复用会话自己配的 provider。
// 会话没配就不抽取——与其偷偷用别处的密钥，不如什么都不做。
func codeMemoryLLMConfigForSession(session *model.AIDevSession) (codeMemoryLLMConfig, error) {
	if session == nil || strings.TrimSpace(session.ProviderBaseURL) == "" {
		return codeMemoryLLMConfig{}, errors.New("会话未配置模型服务")
	}
	apiKey := ""
	if strings.TrimSpace(session.ProviderAPIKey) != "" {
		decrypted, err := encrypt.StringDecrypt(session.ProviderAPIKey)
		if err != nil {
			return codeMemoryLLMConfig{}, errors.New("会话模型密钥无法解密")
		}
		apiKey = decrypted
	}
	return codeMemoryLLMConfig{
		BaseURL: session.ProviderBaseURL, APIKey: apiKey, Model: session.ProviderModel,
	}, nil
}

func codeMemoryProjectName(projectID uint) string {
	if projectID == 0 || global.DB == nil {
		return ""
	}
	var name string
	_ = global.DB.Model(&model.AIProject{}).Where("id = ?", projectID).Pluck("name", &name).Error
	return name
}

func loadCodeMemoryTranscript(sessionID uint) (string, error) {
	var messages []model.AIMessage
	if err := global.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").Limit(codeMemoryTranscriptMaxMessages).
		Find(&messages).Error; err != nil {
		return "", err
	}
	transcript := make([]codeMemoryTranscriptMessage, 0, len(messages))
	// 查出来是倒序的，翻回时间顺序再拼。
	for index := len(messages) - 1; index >= 0; index-- {
		transcript = append(transcript, codeMemoryTranscriptMessage{
			Role: messages[index].Role, Content: messages[index].Content,
		})
	}
	return buildCodeMemoryTranscript(transcript), nil
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
