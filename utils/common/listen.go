package common

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseListenPort 解析监听地址中的端口号，支持 ":5470"、"5470"、"0.0.0.0:5470" 等写法。
// 解析失败或端口非法时返回 0。
func ParseListenPort(raw string) int {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return 0
	}
	portStr := addr
	if strings.Contains(addr, ":") {
		if _, p, err := net.SplitHostPort(addr); err == nil {
			portStr = p
		} else {
			portStr = addr[strings.LastIndex(addr, ":")+1:]
		}
	}
	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || port < 1 || port > 65535 {
		return 0
	}
	return port
}

// NormalizeListenAddr 把配置里的端口写法统一成 net.Listen 可用的监听地址（如 ":5470"）。
// 历史配置/接口可能只写了纯数字（"5470"），直接交给 net.Listen 会报
// "missing port in address"，导致面板启动即退出，这里统一补齐冒号。
func NormalizeListenAddr(raw string) string {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return ""
	}
	if strings.Contains(addr, ":") {
		return addr
	}
	if port := ParseListenPort(addr); port > 0 {
		return fmt.Sprintf(":%d", port)
	}
	return addr
}
