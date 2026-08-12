package api

import (
	"context"
	"strings"

	"github.com/aihop/gopanel/app/model"
)

// 推理强度的取值。空串表示不设置，交给服务端默认。
const (
	codeReasoningEffortNone   = ""
	codeReasoningEffortLow    = "low"
	codeReasoningEffortMedium = "medium"
	codeReasoningEffortHigh   = "high"
)

func normalizeCodeReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case codeReasoningEffortLow:
		return codeReasoningEffortLow
	case codeReasoningEffortMedium:
		return codeReasoningEffortMedium
	case codeReasoningEffortHigh:
		return codeReasoningEffortHigh
	default:
		return codeReasoningEffortNone
	}
}

// aiProviderProbeResult 是一次能力探测的结论。
type aiProviderProbeResult struct {
	SupportsTemperature     bool
	SupportsJSONSchema      bool
	SupportsReasoningEffort bool
}

// probeAIProviderAccount 探测账号的连通性与参数支持情况。
//
// 逐项降级而不是一次全发：一次全发只能得到「失败」这一个结论，
// 分不清是密钥不对、还是模型不吃 temperature。而这两者的处理方式完全不同——
// 前者要用户改配置，后者只需要我们少发一个参数。
//
// 探测在保存时做一次，运行时就永远不会再因为参数不兼容而失败。
// 换成运行时降级重试的话，每次抽取都要多付一个来回。
func probeAIProviderAccount(ctx context.Context, config codeMemoryLLMConfig, reasoningEffort string) (aiProviderProbeResult, error) {
	result := aiProviderProbeResult{}
	messages := []codeMemoryChatMessage{{Role: "user", Content: "ping"}}

	// 第一步只验最基本的连通与鉴权：任何附加参数都可能让请求因为
	// 「参数不支持」而失败，那样就分不清到底是不是密钥的问题。
	if _, err := callCodeMemoryLLMWithOptions(ctx, config, messages, codeMemoryLLMOptions{
		MaxTokens: 16,
	}); err != nil {
		return result, err
	}

	// 逐项加参数，能加上就说明支持。失败不是错误，只是这项不支持。
	if _, err := callCodeMemoryLLMWithOptions(ctx, config, messages, codeMemoryLLMOptions{
		MaxTokens: 16, Temperature: floatPointer(0),
	}); err == nil {
		result.SupportsTemperature = true
	}
	if _, err := callCodeMemoryLLMWithOptions(ctx, config, messages, codeMemoryLLMOptions{
		MaxTokens: 16, Schema: codeMemoryProbeSchema(),
	}); err == nil {
		result.SupportsJSONSchema = true
	}
	if effort := normalizeCodeReasoningEffort(reasoningEffort); effort != codeReasoningEffortNone {
		if _, err := callCodeMemoryLLMWithOptions(ctx, config, messages, codeMemoryLLMOptions{
			MaxTokens: 16, ReasoningEffort: effort,
		}); err == nil {
			result.SupportsReasoningEffort = true
		}
	}
	return result, nil
}

// codeMemoryProbeSchema 是探测用的最小结构化输出 schema。
// 用一个平凡的形状而不是抽取那个大 schema：这里只想知道服务端认不认
// json_schema 这个字段，不想因为 schema 太复杂而收到别的错误。
func codeMemoryProbeSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{"ok": map[string]any{"type": "string"}},
		"required":   []string{"ok"},
	}
}

func floatPointer(value float64) *float64 {
	return &value
}

// codeMemoryOptionsForAccount 按探测结果拼出安全的请求参数。
//
// 只发探测确认支持的：用户填了 high 但模型不支持推理强度时，
// 正确的行为是不发，而不是发出去被 400——后台抽取失败只记日志，
// 用户只会看到记忆一直不出现。
func codeMemoryOptionsForAccount(account *model.AIProviderAccount, schema map[string]any) codeMemoryLLMOptions {
	options := codeMemoryLLMOptions{}
	if account == nil {
		return options
	}
	if account.SupportsTemperature {
		options.Temperature = floatPointer(0)
	}
	if account.SupportsJSONSchema {
		options.Schema = schema
	}
	if account.SupportsReasoningEffort {
		options.ReasoningEffort = normalizeCodeReasoningEffort(account.DefaultReasoningEffort)
	}
	return options
}
