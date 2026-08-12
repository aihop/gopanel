package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 抽取是后台作业，不占用户时间，超时可以给得宽松些；
// 但不能没有上限——卡住的请求会一直占着抽取队列的位置。
const codeMemoryLLMTimeout = 120 * time.Second

// 单次发给模型的记录上限。整段会话可能上百 KB，全发过去既贵又会稀释重点，
// 而且多数 provider 有上下文上限，超了直接报错。
const codeMemoryTranscriptMaxRunes = 24000

type codeMemoryLLMConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type codeMemoryChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// callCodeMemoryLLM 走 OpenAI 兼容的 chat/completions。
//
// GoPanel 本身不直接调模型，只给各家 CLI 配置 provider；抽取是第一个需要
// 自己发请求的场景。复用会话已经配好的 provider，不引入新的配置项——
// 多一处要填的密钥就多一处会填错的地方。
func callCodeMemoryLLM(ctx context.Context, config codeMemoryLLMConfig, messages []codeMemoryChatMessage, schema map[string]any) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.Model) == "" {
		return "", errors.New("会话未配置可用的模型服务，无法抽取记忆")
	}
	payload := map[string]any{
		"model":    config.Model,
		"messages": messages,
		// 温度压到 0：同一段记录反复抽取应当得到一致结果，
		// 记忆库不该因为采样抖动而长出两条同义条目。
		"temperature": 0,
	}
	if schema != nil {
		payload["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "gopanel_memory_extraction",
				"strict": false,
				"schema": schema,
			},
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	requestCtx, cancel := context.WithTimeout(ctx, codeMemoryLLMTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(config.APIKey); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("调用模型服务失败：%w", err)
	}
	defer response.Body.Close()
	// 限制读取量：模型服务异常时可能返回极大的响应体。
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("模型服务返回 %d：%s", response.StatusCode, truncateCodeMemoryText(string(raw), 200))
	}
	return firstCodeMemoryChoiceContent(raw)
}

func firstCodeMemoryChoiceContent(raw []byte) (string, error) {
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", errors.New("模型服务响应无法解析")
	}
	if len(decoded.Choices) == 0 {
		return "", errors.New("模型服务没有返回结果")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("模型服务返回了空结果")
	}
	return content, nil
}

// buildCodeMemoryTranscript 把会话消息拼成给模型看的记录。
//
// 从最近的往前取而不是从头取：一次会话里越靠后的内容越接近最终结论，
// 开头往往是被推翻的试错。脱敏在拼装时就做，不等到发送前。
func buildCodeMemoryTranscript(messages []codeMemoryTranscriptMessage) string {
	lines := make([]string, 0, len(messages))
	total := 0
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		content := strings.TrimSpace(scrubCodeMemoryText(message.Content))
		if content == "" {
			continue
		}
		line := message.Role + ": " + content
		length := len([]rune(line))
		if total+length > codeMemoryTranscriptMaxRunes {
			break
		}
		total += length
		lines = append(lines, line)
	}
	// 上面是倒着收集的，翻回时间顺序再交给模型。
	for left, right := 0, len(lines)-1; left < right; left, right = left+1, right-1 {
		lines[left], lines[right] = lines[right], lines[left]
	}
	return strings.Join(lines, "\n\n")
}

type codeMemoryTranscriptMessage struct {
	Role    string
	Content string
}
