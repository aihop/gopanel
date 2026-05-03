package service

import (
	"fmt"
	"strings"
)

func parseInt(s string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}
func parseBool(s string, defaultValue bool) bool {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "1":
		return true
	case "false", "no", "off", "0":
		return false
	default:
		return defaultValue
	}
}
func parseIntSlice(s string) []int {
	parts := strings.Split(s, ",")
	var result []int
	for _, part := range parts {
		if num := parseInt(strings.TrimSpace(part), -999); num != -999 {
			result = append(result, num)
		}
	}
	return result
}
func parseEnvironment(s string) map[string]string {
	env := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			env[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return env
}
