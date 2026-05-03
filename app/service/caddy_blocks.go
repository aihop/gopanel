package service

import (
	"context"
	"fmt"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/files"
	"os"
	"regexp"
	"strings"
)

func CaddyAddServerBlock(ctx context.Context, domain, proxy, otherDomains, protocol string) (bool, error) {
	exist, err := CaddyExistAddress(domain)
	if exist && err == nil {
		return true, nil
	}
	base := "\n%s {\n    reverse_proxy /* %s\n}\n"
	current, err := CaddyContent()
	if err != nil {
		return false, err
	}
	if protocol == "http" {
		domain = "http://" + domain
		otherDomains = "http://" + otherDomains
	}
	content := current + fmt.Sprintf(base, domain, proxy)
	redirects, err := buildRedirectBlocks(domain, otherDomains)
	if err != nil {
		return false, err
	}
	if redirects != "" {
		content += redirects
	}
	return true, CaddySaveContent(ctx, content)
}
func CaddyReplaceServerBlock(ctx context.Context, domain, proxy, otherDomains, protocol string) (bool, error) {
	if _, err := CaddyRemoveServerBlock(ctx, domain, otherDomains); err != nil {
		return false, err
	}
	return CaddyAddServerBlock(ctx, domain, proxy, otherDomains, protocol)
}
func CaddyAddServerPathBlock(ctx context.Context, domain, routePath, proxy, otherDomains, protocol string) (bool, error) {
	domain = strings.TrimSpace(domain)
	routePath = normalizeCaddyRoutePath(routePath)
	if domain == "" || routePath == "" || proxy == "" {
		return false, fmt.Errorf("参数不能为空")
	}
	filePath := CaddyFilePath()
	fileUtil := files.NewFileOp()
	content, err := fileUtil.GetContent(filePath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	directive := fmt.Sprintf("    reverse_proxy %s %s\n", routePath, proxy)
	updated, changed, err := upsertServerPathBlock(string(content), domain, directive)
	if err != nil {
		return false, err
	}
	if !changed {
		return true, nil
	}
	redirects, err := buildRedirectBlocks(domain, otherDomains)
	if err != nil {
		return false, err
	}
	return true, CaddySaveContent(ctx, updated+redirects)
}
func CaddyAddStaticServerBlock(ctx context.Context, domain, siteRoot, otherDomains, protocol string) (bool, error) {
	domain = strings.TrimSpace(domain)
	siteRoot = strings.TrimSpace(siteRoot)
	if domain == "" || siteRoot == "" {
		return false, fmt.Errorf("参数不能为空")
	}
	exist, err := CaddyExistAddress(domain)
	if exist && err == nil {
		return true, nil
	}
	base := "\n%s {\n    root * %s\n    file_server\n}\n"
	current, err := CaddyContent()
	if err != nil {
		return false, err
	}
	if protocol == constant.ProtocolHTTP {
		if !strings.HasPrefix(domain, "http://") {
			domain = "http://" + domain
		}
		if !strings.HasPrefix(otherDomains, "http://") {
			otherDomains = "http://" + otherDomains
		}
	}
	content := current + fmt.Sprintf(base, domain, siteRoot)
	redirects, err := buildRedirectBlocks(domain, otherDomains)
	if err != nil {
		return false, err
	}
	if redirects != "" {
		content += redirects
	}
	return true, CaddySaveContent(ctx, content)
}
func CaddyReplaceStaticServerBlock(ctx context.Context, domain, siteRoot, otherDomains, protocol string) (bool, error) {
	if _, err := CaddyRemoveServerBlock(ctx, domain, otherDomains); err != nil {
		return false, err
	}
	return CaddyAddStaticServerBlock(ctx, domain, siteRoot, otherDomains, protocol)
}
func CaddyAddReverseProxy(ctx context.Context, domain string, extraDomains string) (bool, error) {
	exist, err := CaddyExistAddress(domain)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, fmt.Errorf("域名还没有绑定，无法给扩展域名做跳转")
	}
	normalized := strings.ReplaceAll(extraDomains, ",", "\n")
	lines := strings.Split(normalized, "\n")
	var blocks []string
	for _, ln := range lines {
		d := strings.TrimSpace(ln)
		if d == "" {
			continue
		}
		ex, err := CaddyExistAddress(d)
		if err != nil {
			return false, err
		}
		if ex {
			continue
		}
		block := fmt.Sprintf("\n%s {\n    redir %s{uri} permanent \n}\n", d, buildCaddyRedirectTarget(domain))
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		return true, nil
	}
	current, err := CaddyContent()
	if err != nil {
		return false, err
	}
	content := current + strings.Join(blocks, "")
	return true, CaddySaveContent(ctx, content)
}
func CaddyRemoveServerBlock(ctx context.Context, primaryDomain, otherDomains string) (bool, error) {
	fileUtil := files.NewFileOp()
	caddyFile, err := fileUtil.GetContent(CaddyFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var domains []string // 统一处理主域名和扩展域名

	primaryDomain = strings.TrimSpace(primaryDomain)
	if primaryDomain != "" {
		domains = append(domains, expandCaddyDomainAliases(primaryDomain)...)
	}
	if otherDomains != "" {
		normalized := strings.ReplaceAll(otherDomains, ",", "\n")
		for _, d := range strings.Split(normalized, "\n") {
			d = strings.TrimSpace(d)
			if d != "" {
				domains = append(domains, expandCaddyDomainAliases(d)...)
			}
		}
	}
	domains = uniqueStrings(domains)
	trimmed := string(caddyFile)
	deleted := false
	for _, d := range domains {
		lines := strings.Split(trimmed, "\n")
		var result []string
		inTargetBlock := false
		bracketCount := 0
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if !inTargetBlock {
				if strings.HasSuffix(trimmedLine, "{") && strings.Contains(trimmedLine, d) {
					headerPart := strings.TrimSuffix(trimmedLine, "{")
					headerParts := strings.Split(headerPart, ",")
					exactMatch := false
					for _, hp := range headerParts {
						if strings.TrimSpace(hp) == d {
							exactMatch = true
							break
						}
					}
					if exactMatch {
						inTargetBlock = true
						bracketCount = strings.Count(line, "{") - strings.Count(line, "}")
						if bracketCount == 0 {
							inTargetBlock = false
						}
						deleted = true
						continue
					}
				}
				result = append(result, line)
			} else {
				bracketCount += strings.Count(line, "{")
				bracketCount -= strings.Count(line, "}")
				if bracketCount <= 0 {
					inTargetBlock = false
					bracketCount = 0
				}
			}
		}
		trimmed = strings.Join(result, "\n")
	}
	trimmedBytes := strings.TrimSpace(trimmed)
	if deleted {
		if err = CaddySaveContent(ctx, trimmedBytes); err != nil {
			return false, err
		}
	}
	return deleted, nil
}
func normalizeCaddyRoutePath(routePath string) string {
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		return ""
	}
	if !strings.HasPrefix(routePath, "/") {
		routePath = "/" + routePath
	}
	if routePath == "/" {
		return "/*"
	}
	return strings.TrimRight(routePath, "/") + "/*"
}
func upsertServerPathBlock(content, domain, directive string) (string, bool, error) {
	reStr := fmt.Sprintf(`(?ms)^(%s\s*\{\n)(.*?)(\n\})`, regexp.QuoteMeta(domain))
	re, err := regexp.Compile(reStr)
	if err != nil {
		return "", false, err
	}
	if !re.MatchString(content) {
		block := fmt.Sprintf("\n%s {\n%s}\n", domain, directive)
		return strings.TrimRight(content, "\n") + block, true, nil
	}
	matches := re.FindStringSubmatch(content)
	if len(matches) != 4 {
		return "", false, fmt.Errorf("解析 Caddy 配置失败")
	}
	body := matches[2]
	if strings.Contains(body, strings.TrimSpace(directive)) {
		return content, false, nil
	}
	replaced := re.ReplaceAllString(content, fmt.Sprintf("${1}%s%s${3}", body, directive))
	return replaced, true, nil
}
func buildRedirectBlocks(domain string, extraDomains string) (string, error) {
	if strings.TrimSpace(extraDomains) == "" {
		return "", nil
	}
	target := buildCaddyRedirectTarget(domain)
	normalized := strings.ReplaceAll(extraDomains, ",", "\n")
	lines := strings.Split(normalized, "\n")
	var blocks []string
	for _, ln := range lines {
		d := strings.TrimSpace(ln)
		if d == "" {
			continue
		}
		ex, err := CaddyExistAddress(d)
		if err != nil {
			return "", err
		}
		if ex {
			continue
		}
		blocks = append(blocks, fmt.Sprintf("\n%s {\n    redir %s{uri} permanent \n}\n", d, target))
	}
	return strings.Join(blocks, ""), nil
}
func buildCaddyRedirectTarget(domain string) string {
	target := strings.TrimSpace(domain)
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	return "http://" + target
}
func expandCaddyDomainAliases(domain string) []string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil
	}
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "http://"), "https://")
	var aliases []string
	if host != "" {
		aliases = append(aliases, host, "http://"+host, "https://"+host)
	}
	if strings.Contains(domain, "://") {
		aliases = append([]string{domain}, aliases...)
	}
	return uniqueStrings(aliases)
}
func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
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
