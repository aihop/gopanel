package service

import (
	"fmt"
	"strings"

	"github.com/aihop/gopanel/constant"
)

func BuildWebsiteCaddyDomain(primaryDomain, protocol string) string {
	primaryDomain = sanitizeWebsitePrimaryDomain(primaryDomain)
	if primaryDomain == "" {
		return ""
	}
	if strings.HasPrefix(primaryDomain, "http://") || strings.HasPrefix(primaryDomain, "https://") {
		return primaryDomain
	}
	if strings.EqualFold(protocol, constant.ProtocolHTTPS) {
		return fmt.Sprintf("https://%s", primaryDomain)
	}
	return fmt.Sprintf("http://%s", primaryDomain)
}

func sanitizeWebsitePrimaryDomain(primaryDomain string) string {
	primaryDomain = strings.TrimSpace(primaryDomain)
	primaryDomain = strings.Trim(primaryDomain, "`\"'")
	return strings.TrimSpace(primaryDomain)
}

func normalizeWebsiteDomainForCompare(domain string) string {
	domain = sanitizeWebsitePrimaryDomain(domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	return strings.TrimSpace(domain)
}
