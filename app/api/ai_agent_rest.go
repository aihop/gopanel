package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var previewProbeResolver = net.DefaultResolver

var aiPreviewProbeClient = &http.Client{
	Timeout:   2 * time.Second,
	Transport: &http.Transport{DialContext: dialPublicPreviewAddress},
	CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		return validatePreviewProbeURL(req.Context(), req.URL)
	},
}

func dialPublicPreviewAddress(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("预览地址无效")
	}
	addresses, err := resolvePublicPreviewAddresses(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}

func validatePreviewProbeURL(ctx context.Context, parsed *url.URL) error {
	if parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || strings.TrimSpace(parsed.Hostname()) == "" {
		return errors.New("预览地址无效")
	}
	_, err := resolvePublicPreviewAddresses(ctx, parsed.Hostname())
	return err
}

func resolvePublicPreviewAddresses(ctx context.Context, host string) ([]net.IP, error) {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if strings.EqualFold(host, "localhost") {
		return nil, errors.New("不允许探测本机或内网预览地址")
	}
	if parsedIP := net.ParseIP(host); parsedIP != nil {
		if !isPublicPreviewIP(parsedIP) {
			return nil, errors.New("不允许探测本机或内网预览地址")
		}
		return []net.IP{parsedIP}, nil
	}
	resolved, err := previewProbeResolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return nil, errors.New("预览地址无法解析")
	}
	addresses := make([]net.IP, 0, len(resolved))
	for _, address := range resolved {
		if !isPublicPreviewIP(address.IP) {
			return nil, errors.New("不允许探测本机或内网预览地址")
		}
		addresses = append(addresses, address.IP)
	}
	return addresses, nil
}

func isPublicPreviewIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}
