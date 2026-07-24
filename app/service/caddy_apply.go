package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/gpagent"
)

// ApplyCaddyFromDB 从数据库应用Caddy配置
// @param ctx 上下文
// @return error 错误
func ApplyCaddyFromDB(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var websites []model.Website
	if err := global.DB.Order("id ASC").Find(&websites).Error; err != nil {
		return err
	}

	ids := make([]uint, 0, len(websites))
	for i := range websites {
		ids = append(ids, websites[i].ID)
	}

	domainByWebsite := map[uint][]model.WebsiteDomain{}
	upstreamByWebsite := map[uint][]*model.WebsiteUpstream{}
	if len(ids) > 0 {
		var domains []model.WebsiteDomain
		if err := global.DB.Where("website_id IN ?", ids).Find(&domains).Error; err != nil {
			return err
		}
		for _, d := range domains {
			domainByWebsite[d.WebsiteID] = append(domainByWebsite[d.WebsiteID], d)
		}
		var upstreams []model.WebsiteUpstream
		if err := global.DB.Where("website_id IN ?", ids).Order("sort asc,id asc").Find(&upstreams).Error; err != nil {
			return err
		}
		for i := range upstreams {
			item := &upstreams[i]
			upstreamByWebsite[item.WebsiteID] = append(upstreamByWebsite[item.WebsiteID], item)
		}
	}

	for i := range websites {
		websites[i].Upstreams = upstreamByWebsite[websites[i].ID]
	}
	caddyfile := renderCaddyfile(websites, domainByWebsite)
	if strings.TrimSpace(caddyfile) == "" {
		return errors.New("generated caddyfile is empty")
	}

	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := gpagent.Do(callCtx, "CADDY_APPLY", map[string]interface{}{
		"caddyfile": caddyfile,
	})
	return err
}

func renderCaddyfile(websites []model.Website, domainByWebsite map[uint][]model.WebsiteDomain) string {
	var b strings.Builder

	for _, w := range websites {
		if w.Status != "" && w.Status != constant.WebRunning && w.Status != "Running" {
			continue
		}

		otherDomains := domainByWebsite[w.ID]

		// When redirectDomainsToPrimary is enabled:
		// - Redirect type: merge all domains into the main block (same redirect target)
		// - Other types: exclude secondary domains to avoid duplicate blocks
		mainDomains := otherDomains
		if w.RedirectDomainsToPrimary && w.Type != constant.Redirect {
			var primaryOnly []model.WebsiteDomain
			for _, d := range otherDomains {
				if normalizeWebsiteDomainForCompare(d.Domain) == normalizeWebsiteDomainForCompare(w.PrimaryDomain) {
					primaryOnly = append(primaryOnly, d)
				}
			}
			mainDomains = primaryOnly
		}

		addrs := collectAddresses(w, mainDomains)
		if len(addrs) == 0 {
			continue
		}

		b.WriteString(strings.Join(addrs, ", "))
		b.WriteString(" {\n")

		if w.AccessLog {
			accessLogPath := websiteAccessLogPath(w.Alias)
			if err := ensureWebsiteLogDir(w.Alias); err == nil {
				b.WriteString("  log {\n")
				b.WriteString("    output file ")
				b.WriteString(accessLogPath)
				b.WriteString("\n")
				b.WriteString("  }\n\n")
			}
		}

		if allowlist := normalizeSecurityIPList(w.IPAllowlist); len(allowlist) > 0 {
			b.WriteString("  @ip_allow_only {\n")
			b.WriteString("    not remote_ip ")
			b.WriteString(strings.Join(allowlist, " "))
			b.WriteString("\n  }\n")
			b.WriteString("  respond @ip_allow_only \"Forbidden by IP Allowlist\" 403\n\n")
		}

		if blocklist := normalizeSecurityIPList(w.IPBlocklist); len(blocklist) > 0 {
			b.WriteString("  @ip_blocked remote_ip ")
			b.WriteString(strings.Join(blocklist, " "))
			b.WriteString("\n")
			b.WriteString("  respond @ip_blocked \"Forbidden by IP Blocklist\" 403\n\n")
		}

		if w.AntiCrawler {
			b.WriteString("  @blocked_ua {\n")
			b.WriteString("    header_regexp User-Agent \"(?i)(curl|python|sqlmap|nmap|wget|headless|dirbuster|nikto|java|perl|ruby)\"\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @blocked_ua \"Forbidden by Anti-Crawler\" 403\n\n")
		}

		if w.AntiLeech {
			b.WriteString("  @leech {\n")
			b.WriteString("    path *.jpg *.jpeg *.png *.gif *.webp *.svg *.mp4 *.mp3 *.flv *.css *.js *.woff *.woff2 *.ttf *.eot\n")
			b.WriteString("    not header Referer \"\"\n")
			// Replace dots in primary domain for regex matching
			primaryRegex := strings.ReplaceAll(w.PrimaryDomain, ".", "\\.")
			b.WriteString("    not header_regexp Referer \"(?i)https?://(www\\.)?" + primaryRegex + "\"\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @leech \"Forbidden by Anti-Leech\" 403\n\n")
		}

		if w.RateLimitMode != "" && w.RateLimitMode != "none" {
			b.WriteString("  rate_limit {\n")
			b.WriteString("    zone website_" + strconv.Itoa(int(w.ID)) + " {\n")
			b.WriteString("      key {remote_ip}\n")
			b.WriteString("      window 1s\n")
			if w.RateLimitMode == "strict" {
				b.WriteString("      events 3\n")
			} else { // normal
				b.WriteString("      events 10\n")
			}
			b.WriteString("    }\n")
			b.WriteString("  }\n\n")
		}

		if w.WafEnable {
			b.WriteString("  @waf_sqli {\n")
			b.WriteString("    query ~*(?i)(union.*select|waitfor.*delay|select.*from.*information_schema|1=1)\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @waf_sqli \"Forbidden by WAF (SQLi)\" 403\n\n")

			b.WriteString("  @waf_xss {\n")
			b.WriteString("    query ~*(?i)(<script>|javascript:|onerror=)\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @waf_xss \"Forbidden by WAF (XSS)\" 403\n\n")

			b.WriteString("  @waf_path {\n")
			b.WriteString("    path_regexp (?i)(/\\.\\./|%c0%ae|/etc/passwd)\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @waf_path \"Forbidden by WAF (Path Traversal)\" 403\n\n")
		}

		if w.BlockSensitive {
			b.WriteString("  @sensitive_hidden {\n")
			b.WriteString("    path_regexp \"/\\\\.\"\n")
			b.WriteString("    not path /.well-known/*\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @sensitive_hidden \"Access Denied\" 403\n\n")

			b.WriteString("  @sensitive_ext {\n")
			b.WriteString("    path *.sql *.bak *.log *.conf *.ini\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @sensitive_ext \"Access Denied\" 403\n\n")
		}

		if w.SecurityHeader || (w.HstsEnabled && strings.EqualFold(w.Protocol, constant.ProtocolHTTPS)) {
			b.WriteString("  header {\n")
			if w.SecurityHeader {
				b.WriteString("    X-Frame-Options \"SAMEORIGIN\"\n")
				b.WriteString("    X-Content-Type-Options \"nosniff\"\n")
				b.WriteString("    Referrer-Policy \"strict-origin-when-cross-origin\"\n")
			}
			if w.HstsEnabled && strings.EqualFold(w.Protocol, constant.ProtocolHTTPS) {
				b.WriteString("    Strict-Transport-Security \"max-age=31536000; includeSubDomains; preload\"\n")
			}
			b.WriteString("  }\n\n")
		}

		if custom := strings.TrimSpace(w.HttpConfig); custom != "" {
			b.WriteString(custom)
			b.WriteString("\n\n")
		}

		switch w.Type {
		case constant.Static:
			root := strings.TrimSpace(w.SiteDir)
			if root == "" {
				root = "/"
			}
			b.WriteString("  root * ")
			b.WriteString(root)
			b.WriteString("\n  file_server\n")
		case constant.Redirect:
			target := strings.TrimSpace(w.Proxy)
			if target == "" {
				target = "/"
			}
			code := w.RedirectCode
			if code == 0 {
				code = 301
			}
			b.WriteString("  redir " + target + "{uri} " + strconv.Itoa(code) + "\n")
		case constant.Proxy, constant.WebApp:
			dials := reverseProxyDialList(w)
			if len(dials) == 0 {
				dials = []string{"127.0.0.1:80"}
			}
			healthURI, healthInterval, healthTimeout := reverseProxyHealthSettings(w)
			if len(dials) == 1 && healthURI == "" && healthInterval == "" && healthTimeout == "" {
				b.WriteString("  reverse_proxy ")
				b.WriteString(dials[0])
				b.WriteString("\n")
				break
			}
			b.WriteString("  reverse_proxy {\n")
			b.WriteString("    lb_policy round_robin\n")
			b.WriteString("    to ")
			b.WriteString(strings.Join(dials, " "))
			b.WriteString("\n")
			if healthURI != "" {
				b.WriteString("    health_uri ")
				b.WriteString(healthURI)
				b.WriteString("\n")
			}
			if healthInterval != "" {
				b.WriteString("    health_interval ")
				b.WriteString(healthInterval)
				b.WriteString("\n")
			}
			if healthTimeout != "" {
				b.WriteString("    health_timeout ")
				b.WriteString(healthTimeout)
				b.WriteString("\n")
			}
			b.WriteString("  }\n")
		default:
			up := strings.TrimSpace(w.Proxy)
			if up != "" {
				b.WriteString("  reverse_proxy ")
				b.WriteString(up)
				b.WriteString("\n")
			} else {
				b.WriteString("  respond \"not configured\" 502\n")
			}
		}

		b.WriteString("}\n\n")

		// Append separate redirect blocks for secondary domains
		if w.RedirectDomainsToPrimary && w.Type != constant.Redirect {
			for _, d := range domainByWebsite[w.ID] {
				host := strings.TrimSpace(d.Domain)
				if host == "" || normalizeWebsiteDomainForCompare(host) == normalizeWebsiteDomainForCompare(w.PrimaryDomain) {
					continue
				}
				proto := "https"
				if strings.EqualFold(w.Protocol, constant.ProtocolHTTP) {
					proto = "http"
				}
				// For redirect type websites, other domains should redirect
				// directly to the target URL (w.Proxy), not to the primary
				// domain which would then redirect again (2 hops → 1 hop).
				redirectURL := w.PrimaryDomain
				needProto := true
				if w.Type == constant.Redirect && strings.TrimSpace(w.Proxy) != "" {
					redirectURL = strings.TrimSpace(w.Proxy)
					needProto = false // redirect target is already a full URL
				}
				b.WriteString(host)
				b.WriteString(" {\n")
				if needProto {
					b.WriteString("  redir " + proto + "://" + redirectURL + "{uri} 301\n")
				} else {
					b.WriteString("  redir " + redirectURL + "{uri} 301\n")
				}
				b.WriteString("}\n\n")
			}
		}
	}

	return b.String()
}

func normalizeSecurityIPList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	replacer := strings.NewReplacer(",", "\n", ";", "\n", "\r", "\n", "\t", "\n")
	value = replacer.Replace(value)
	seen := map[string]struct{}{}
	var result []string
	for _, line := range strings.Split(value, "\n") {
		item := strings.TrimSpace(line)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func collectAddresses(w model.Website, domains []model.WebsiteDomain) []string {
	addrSet := map[string]struct{}{}
	for _, d := range domains {
		host := strings.TrimSpace(d.Domain)
		if host == "" {
			continue
		}
		p := d.Port
		if p > 0 && p != 80 && p != 443 {
			host = host + ":" + strconv.Itoa(p)
		}
		addrSet[host] = struct{}{}
	}
	if w.PrimaryDomain != "" {
		addrSet[strings.TrimSpace(w.PrimaryDomain)] = struct{}{}
	}

	addrs := make([]string, 0, len(addrSet))
	for a := range addrSet {
		if a == "" {
			continue
		}
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)
	return addrs
}
