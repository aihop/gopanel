package common

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"
)

type HostType int

const (
	TypeUnknown HostType = iota
	TypeIPv4
	TypeIPv6
	TypeDomain
)

type HostInfo struct {
	HostType HostType
	Host     string // 纯 host（不含端口）
	Port     string // 可能为空
}

// ParseHostType 解析一段 URL（或 host:port）
func ParseHostType(raw string) HostInfo {
	// 1. 去掉 scheme（如果有）
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw // 统一补成 URL，便于解析
	}
	u, err := url.Parse(raw)
	if err != nil {
		return HostInfo{HostType: TypeUnknown}
	}

	// 2. 分离 host:port
	host, port, _ := net.SplitHostPort(u.Host)
	if host == "" {
		// 没端口的情况
		host = u.Host
	}

	// 3. 判断类型
	ip := net.ParseIP(host)
	switch {
	case ip == nil:
		return HostInfo{TypeDomain, host, port}
	case ip.To4() != nil:
		return HostInfo{TypeIPv4, host, port}
	default:
		return HostInfo{TypeIPv6, host, port}
	}
}

// 解析域名
func DomainFindIps(domain string, timeout time.Duration) ([]net.IP, error) {
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 1. 解析域名
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", domain)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, nil
	}
	return ips, nil
}

// 收集本机非 loopback IP
func LocalNonLoopbackIPs() (map[string]struct{}, error) {
	m := make(map[string]struct{})
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				m[ipn.IP.String()] = struct{}{}
			}
		}
	}
	return m, nil
}

// 把输入统一解析为 IPv4 切片,暂时不考虑ipv6
func ResolveIPv4(raw string) ([]net.IP, error) {
	info := ParseHostType(raw)
	switch info.HostType {
	case TypeIPv4:
		return []net.IP{net.ParseIP(info.Host)}, nil
	case TypeDomain:
		ips, err := DomainFindIps(info.Host, 5*time.Second)
		if err != nil {
			return nil, err
		}
		// 只保留 IPv4
		var v4 []net.IP
		for _, ip := range ips {
			if ip.To4() != nil {
				v4 = append(v4, ip)
			}
		}
		return v4, nil
	default:
		return nil, nil
	}
}
