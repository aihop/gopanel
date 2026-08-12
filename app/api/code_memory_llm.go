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

// codeMemoryLLMOptions 是一次请求的可选参数。
//
// 全部做成可选是因为各家支持面差别很大：OpenAI 的 o 系列直接拒绝
// temperature，部分服务不认 json_schema，reasoning_effort 更是只有少数几家有。
// 发一个不支持的参数换来的是 400，而不是被忽略。
type codeMemoryLLMOptions struct {
	Temperature     *float64
	Schema          map[string]any
	ReasoningEffort string
	MaxTokens       int
}

// callCodeMemoryLLM 走 OpenAI 兼容的 chat/completions。
//
// 温度压到 0（当模型支持时）：同一段记录反复抽取应当得到一致结果，
// 记忆库不该因为采样抖动而长出两条同义条目。
func callCodeMemoryLLM(ctx context.Context, config codeMemoryLLMConfig, messages []codeMemoryChatMessage, schema map[string]any) (string, error) {
	return callCodeMemoryLLMWithOptions(ctx, config, messages, codeMemoryLLMOptions{
		Temperature: floatPointer(0), Schema: schema,
	})
}

func callCodeMemoryLLMWithOptions(
	ctx context.Context,
	config codeMemoryLLMConfig,
	messages []codeMemoryChatMessage,
	options codeMemoryLLMOptions,
) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.Model) == "" {
		return "", errors.New("未配置可用的模型服务")
	}
	payload := map[string]any{
		"model":    config.Model,
		"messages": messages,
	}
	if options.Temperature != nil {
		payload["temperature"] = *options.Temperature
	}
	if effort := normalizeCodeReasoningEffort(options.ReasoningEffort); effort != codeReasoningEffortNone {
		payload["reasoning_effort"] = effort
	}
	if options.MaxTokens > 0 {
		payload["max_tokens"] = options.MaxTokens
	}
	if options.Schema != nil {
		payload["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "gopanel_memory_extraction",
				"strict": false,
				"schema": options.Schema,
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
