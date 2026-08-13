package aiprovider

import (
	"encoding/json"
	"strings"
)

func responsesInput(messages []Message) []any {
	items := make([]any, 0, len(messages))
	for _, message := range messages {
		if message.Role == "tool" {
			items = append(items, map[string]any{
				"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content,
			})
			continue
		}
		if strings.TrimSpace(message.Content) != "" {
			items = append(items, map[string]any{"role": message.Role, "content": message.Content})
		}
		for _, call := range message.ToolCalls {
			items = append(items, map[string]any{
				"type": "function_call", "call_id": call.ID,
				"name": call.Function.Name, "arguments": call.Function.Arguments,
			})
		}
	}
	return items
}

func responsesTools(tools []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		function, _ := tool["function"].(map[string]any)
		if function == nil {
			continue
		}
		result = append(result, map[string]any{
			"type": "function", "name": function["name"],
			"description": function["description"], "parameters": function["parameters"],
		})
	}
	return result
}

func anthropicMessages(messages []Message) (string, []map[string]any) {
	systemParts := make([]string, 0)
	result := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		if message.Role == "system" {
			if strings.TrimSpace(message.Content) != "" {
				systemParts = append(systemParts, strings.TrimSpace(message.Content))
			}
			continue
		}
		if message.Role == "tool" {
			block := map[string]any{
				"type": "tool_result", "tool_use_id": message.ToolCallID, "content": message.Content,
			}
			if len(result) > 0 && result[len(result)-1]["role"] == "user" {
				if content, ok := result[len(result)-1]["content"].([]map[string]any); ok {
					result[len(result)-1]["content"] = append(content, block)
					continue
				}
			}
			result = append(result, map[string]any{"role": "user", "content": []map[string]any{block}})
			continue
		}
		content := make([]map[string]any, 0, 1+len(message.ToolCalls))
		if strings.TrimSpace(message.Content) != "" {
			content = append(content, map[string]any{"type": "text", "text": message.Content})
		}
		for _, call := range message.ToolCalls {
			input := map[string]any{}
			_ = json.Unmarshal([]byte(call.Function.Arguments), &input)
			content = append(content, map[string]any{
				"type": "tool_use", "id": call.ID, "name": call.Function.Name, "input": input,
			})
		}
		if len(content) > 0 {
			result = append(result, map[string]any{"role": message.Role, "content": content})
		}
	}
	return strings.Join(systemParts, "\n\n"), result
}

func anthropicTools(tools []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		function, _ := tool["function"].(map[string]any)
		if function == nil {
			continue
		}
		result = append(result, map[string]any{
			"name": function["name"], "description": function["description"], "input_schema": function["parameters"],
		})
	}
	return result
}
