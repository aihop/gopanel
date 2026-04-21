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
	if len(ids) > 0 {
		var domains []model.WebsiteDomain
		if err := global.DB.Where("website_id IN ?", ids).Find(&domains).Error; err != nil {
			return err
		}
		for _, d := range domains {
			domainByWebsite[d.WebsiteID] = append(domainByWebsite[d.WebsiteID], d)
		}
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

		addrs := collectAddresses(w, domainByWebsite[w.ID])
		if len(addrs) == 0 {
			continue
		}

		b.WriteString(strings.Join(addrs, ", "))
		b.WriteString(" {\n")

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
			b.WriteString("    zone website_" + strconv.Itoa(int(w.ID)) + "\n")
			b.WriteString("    key {remote_ip}\n")
			b.WriteString("    window 1s\n")
			if w.RateLimitMode == "strict" {
				b.WriteString("    events 3\n")
			} else { // normal
				b.WriteString("    events 10\n")
			}
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
			b.WriteString("    path /.* /*/.*\n")
			b.WriteString("    not path /.well-known/*\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @sensitive_hidden \"Access Denied\" 403\n\n")

			b.WriteString("  @sensitive_ext {\n")
			b.WriteString("    path *.sql *.bak *.log *.conf *.ini\n")
			b.WriteString("  }\n")
			b.WriteString("  respond @sensitive_ext \"Access Denied\" 403\n\n")
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
		case constant.Proxy, constant.WebApp:
			up := strings.TrimSpace(w.Proxy)
			if up == "" {
				up = "127.0.0.1:80"
			}
			b.WriteString("  reverse_proxy ")
			b.WriteString(up)
			b.WriteString("\n")
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
	}

	return b.String()
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

