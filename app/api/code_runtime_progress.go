package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type codeRuntimeProgress struct {
	CurrentStep    int       `json:"currentStep"`
	TotalSteps     int       `json:"totalSteps"`
	CompletedSteps int       `json:"completedSteps"`
	StepTitle      string    `json:"stepTitle"`
	ChangedFiles   int       `json:"changedFiles"`
	Additions      int       `json:"additions"`
	Deletions      int       `json:"deletions"`
	Files          []string  `json:"files,omitempty"`
	Source         string    `json:"source"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type codexPlanItem struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

func parseCodexPlanProgress(name, input string, updatedAt time.Time) *codeRuntimeProgress {
	var argument string
	if name == "update_plan" {
		argument = strings.TrimSpace(input)
	} else {
		argument = extractJSCallArgument(input, "tools.update_plan")
	}
	items := parseCodexPlanItems(argument)
	if len(items) == 0 {
		return nil
	}
	progress := &codeRuntimeProgress{TotalSteps: len(items), Source: "codex_plan", UpdatedAt: updatedAt}
	current := -1
	for index, item := range items {
		status := normalizeCodexPlanStatus(item.Status)
		if status == "completed" {
			progress.CompletedSteps++
		}
		if current < 0 && status == "in_progress" {
			current = index
		}
	}
	if current < 0 {
		for index, item := range items {
			if normalizeCodexPlanStatus(item.Status) == "pending" {
				current = index
				break
			}
		}
	}
	if current < 0 {
		current = len(items) - 1
	}
	progress.CurrentStep = current + 1
	progress.StepTitle = strings.TrimSpace(items[current].Step)
	return progress
}

func normalizeCodexPlanStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.ReplaceAll(status, "-", "_")
	if status == "inprogress" {
		return "in_progress"
	}
	return status
}

func extractJSCallArgument(input, callName string) string {
	searchFrom := 0
	var latest string
	for {
		offset := strings.Index(input[searchFrom:], callName)
		if offset < 0 {
			return latest
		}
		position := searchFrom + offset + len(callName)
		for position < len(input) && isJSSpace(input[position]) {
			position++
		}
		if position < len(input) && input[position] == '(' {
			if argument, end, ok := scanJSBalanced(input, position, '(', ')'); ok {
				latest = strings.TrimSpace(argument[1 : len(argument)-1])
				searchFrom = end
				continue
			}
		}
		searchFrom = position
		if searchFrom >= len(input) {
			return latest
		}
	}
}

func scanJSBalanced(input string, start int, open, close byte) (string, int, bool) {
	if start >= len(input) || input[start] != open {
		return "", start, false
	}
	depth := 0
	quote := byte(0)
	escaped := false
	for index := start; index < len(input); index++ {
		character := input[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
			} else if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		if character == open {
			depth++
		} else if character == close {
			depth--
			if depth == 0 {
				return input[start : index+1], index + 1, true
			}
		}
	}
	return "", start, false
}

func parseCodexPlanItems(argument string) []codexPlanItem {
	var payload struct {
		Plan []codexPlanItem `json:"plan"`
	}
	if json.Unmarshal([]byte(argument), &payload) == nil && len(payload.Plan) > 0 {
		return payload.Plan
	}
	planArray := findJSPropertyValue(argument, "plan", '[', ']')
	if planArray == "" {
		return nil
	}
	items := make([]codexPlanItem, 0)
	for position := 1; position < len(planArray)-1; {
		if planArray[position] != '{' {
			position++
			continue
		}
		object, end, ok := scanJSBalanced(planArray, position, '{', '}')
		if !ok {
			break
		}
		step, stepOK := findJSStringProperty(object, "step")
		status, statusOK := findJSStringProperty(object, "status")
		if stepOK && statusOK {
			items = append(items, codexPlanItem{Step: step, Status: status})
		}
		position = end
	}
	return items
}

func findJSPropertyValue(input, property string, open, close byte) string {
	for position := 0; position < len(input); {
		key, next, ok := scanJSPropertyKey(input, position)
		if !ok {
			position++
			continue
		}
		position = next
		for position < len(input) && isJSSpace(input[position]) {
			position++
		}
		if position >= len(input) || input[position] != ':' {
			continue
		}
		position++
		for position < len(input) && isJSSpace(input[position]) {
			position++
		}
		if key == property && position < len(input) && input[position] == open {
			value, _, ok := scanJSBalanced(input, position, open, close)
			if ok {
				return value
			}
		}
	}
	return ""
}

func findJSStringProperty(input, property string) (string, bool) {
	for position := 0; position < len(input); {
		key, next, ok := scanJSPropertyKey(input, position)
		if !ok {
			position++
			continue
		}
		position = next
		for position < len(input) && isJSSpace(input[position]) {
			position++
		}
		if position >= len(input) || input[position] != ':' {
			continue
		}
		position++
		for position < len(input) && isJSSpace(input[position]) {
			position++
		}
		if key == property {
			value, _, ok := scanJSString(input, position)
			return value, ok
		}
	}
	return "", false
}

func scanJSPropertyKey(input string, position int) (string, int, bool) {
	for position < len(input) && isJSSpace(input[position]) {
		position++
	}
	if position >= len(input) {
		return "", position, false
	}
	if input[position] == '\'' || input[position] == '"' {
		return scanJSString(input, position)
	}
	start := position
	for position < len(input) && ((input[position] >= 'a' && input[position] <= 'z') ||
		(input[position] >= 'A' && input[position] <= 'Z') || input[position] == '_' || input[position] == '$') {
		position++
	}
	return input[start:position], position, position > start
}

func scanJSString(input string, start int) (string, int, bool) {
	if start >= len(input) || (input[start] != '\'' && input[start] != '"' && input[start] != '`') {
		return "", start, false
	}
	quote := input[start]
	var value strings.Builder
	for position := start + 1; position < len(input); position++ {
		character := input[position]
		if character == quote {
			return value.String(), position + 1, true
		}
		if character != '\\' || position+1 >= len(input) {
			value.WriteByte(character)
			continue
		}
		position++
		escaped := input[position]
		switch escaped {
		case 'n':
			value.WriteByte('\n')
		case 'r':
			value.WriteByte('\r')
		case 't':
			value.WriteByte('\t')
		case 'b':
			value.WriteByte('\b')
		case 'f':
			value.WriteByte('\f')
		case 'u':
			if position+4 < len(input) {
				if codepoint, err := strconv.ParseInt(input[position+1:position+5], 16, 32); err == nil {
					value.WriteRune(rune(codepoint))
					position += 4
					continue
				}
			}
			value.WriteByte(escaped)
		default:
			value.WriteByte(escaped)
		}
	}
	return "", start, false
}

func isJSSpace(character byte) bool {
	return character == ' ' || character == '\n' || character == '\r' || character == '\t'
}
