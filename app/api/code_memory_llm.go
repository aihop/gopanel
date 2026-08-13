package api

import (
	"context"
	"errors"
	"strings"

	"github.com/aihop/gopanel/utils/aiprovider"
)

// 单次发给模型的记录上限。整段会话可能上百 KB，全发过去既贵又会稀释重点，
// 而且多数 provider 有上下文上限，超了直接报错。
const codeMemoryTranscriptMaxRunes = 24000

type codeMemoryLLMConfig struct {
	Protocol string
	BaseURL  string
	APIKey   string
	Model    string
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

// callCodeMemoryLLM 使用账号声明的接口协议。
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
	providerMessages := make([]aiprovider.Message, 0, len(messages))
	for _, message := range messages {
		providerMessages = append(providerMessages, aiprovider.Message{Role: message.Role, Content: message.Content})
	}
	response, err := aiprovider.Call(ctx, aiprovider.Config{
		Protocol: config.Protocol,
		BaseURL:  config.BaseURL,
		APIKey:   config.APIKey,
		Model:    config.Model,
	}, aiprovider.Request{
		Messages:        providerMessages,
		Temperature:     options.Temperature,
		Schema:          options.Schema,
		SchemaName:      "gopanel_memory_extraction",
		ReasoningEffort: normalizeCodeReasoningEffort(options.ReasoningEffort),
		MaxTokens:       options.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(response.Message.Content)
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
