package docker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func getAnyString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case string:
			return x
		default:
			return strings.TrimSpace(fmt.Sprint(x))
		}
	}
	return ""
}

func getFirstString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case []string:
			if len(x) > 0 {
				return x[0]
			}
		case []interface{}:
			if len(x) > 0 {
				if s, ok := x[0].(string); ok {
					return s
				}
				return strings.TrimSpace(fmt.Sprint(x[0]))
			}
		case string:
			return x
		}
	}
	return ""
}

func getStringSlice(m map[string]interface{}, keys ...string) []string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case []string:
			return x
		case []interface{}:
			var out []string
			for _, it := range x {
				out = append(out, strings.TrimSpace(fmt.Sprint(it)))
			}
			out = compactStrings(out)
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func getAnyStringMap(m map[string]interface{}, keys ...string) map[string]string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case map[string]string:
			return x
		case map[string]interface{}:
			out := make(map[string]string, len(x))
			for kk, vv := range x {
				out[kk] = strings.TrimSpace(fmt.Sprint(vv))
			}
			return out
		}
	}
	return nil
}

func getAnyTime(m map[string]interface{}, keys ...string) time.Time {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case float64:
			if x > 0 {
				return time.Unix(int64(x), 0)
			}
		case int64:
			if x > 0 {
				return time.Unix(x, 0)
			}
		case json.Number:
			n, _ := x.Int64()
			if n > 0 {
				return time.Unix(n, 0)
			}
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				continue
			}
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				return time.Unix(n, 0)
			}
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func getPodmanPorts(m map[string]interface{}) []string {
	v, ok := m["Ports"]
	if !ok || v == nil {
		v, ok = m["ports"]
		if !ok || v == nil {
			return nil
		}
	}
	switch x := v.(type) {
	case []string:
		return compactStrings(x)
	case []interface{}:
		var out []string
		for _, it := range x {
			switch p := it.(type) {
			case string:
				out = append(out, p)
			case map[string]interface{}:
				out = append(out, formatPodmanPortMap(p))
			}
		}
		return compactStrings(out)
	}
	return nil
}

func formatPodmanPortMap(m map[string]interface{}) string {
	hostIP := getAnyString(m, "host_ip", "HostIP", "hostIP")
	hostPort := getAnyString(m, "host_port", "HostPort", "hostPort")
	containerPort := getAnyString(m, "container_port", "ContainerPort", "containerPort")
	proto := strings.ToLower(getAnyString(m, "protocol", "Protocol", "proto"))
	if hostPort == "" || containerPort == "" {
		return ""
	}
	if hostIP == "" {
		hostIP = "0.0.0.0"
	}
	if proto == "" {
		proto = "tcp"
	}
	return hostIP + ":" + hostPort + "->" + containerPort + "/" + proto
}

func compactStrings(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
