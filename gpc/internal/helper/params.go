package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}

func getInt(m map[string]interface{}, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func getBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, err := strconv.ParseBool(x)
		return err == nil && b
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, it := range x {
			if s, ok := it.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	default:
		return nil
	}
}

func getIntSlice(m map[string]interface{}, key string) ([]int, error) {
	if m == nil {
		return nil, nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil, nil
	}
	switch x := v.(type) {
	case []int:
		return x, nil
	case []interface{}:
		var out []int
		for _, it := range x {
			switch y := it.(type) {
			case float64:
				out = append(out, int(y))
			case int:
				out = append(out, y)
			case string:
				n, err := strconv.Atoi(y)
				if err != nil {
					return nil, errors.New("invalid params: ports")
				}
				out = append(out, n)
			default:
				return nil, errors.New("invalid params: ports")
			}
		}
		return out, nil
	case float64:
		return []int{int(x)}, nil
	case int:
		return []int{x}, nil
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return nil, errors.New("invalid params: ports")
		}
		return []int{n}, nil
	default:
		return nil, errors.New("invalid params: ports")
	}
}
