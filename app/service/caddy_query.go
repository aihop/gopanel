package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/gpagent"
	"os"
	"regexp"
	"strings"
	"time"
)

func CaddyExistDomain(ctx context.Context, domain string) (bool, error) {
	if domain == "" {
		return false, fmt.Errorf("域名、IP不能为空")
	}
	content, err := os.ReadFile(CaddyFilePath())
	if err != nil {
		return false, err
	}
	if len(content) == 0 || string(content) == "" {
		return false, nil
	}
	config, err := CaddyFileToStruct(ctx, string(content))
	if err != nil {
		return false, err
	}
	info := common.ParseHostType(domain)
	for _, server := range config.Apps.HTTP.Servers {
		for _, route := range server.Routes {
			for _, match := range route.Match {
				for _, h := range match.Host {
					if h == info.Host {
						if info.Port != "" {
							for _, l := range server.Listen {
								if ":"+info.Port == l {
									return true, nil
								}
							}
						} else {
							return true, nil
						}
					}
				}
			}
		}
	}
	return false, err
}
func CaddyExistAddress(domain string) (bool, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false, fmt.Errorf("域名、IP不能为空")
	}
	hosts, err := CaddyGetAllHosts()
	if err != nil {
		return false, err
	}
	for _, host := range hosts {
		if strings.TrimSpace(host) == domain {
			return true, nil
		}
	}
	return false, nil
}
func CaddyGetAllHosts() ([]string, error) {
	content, err := os.ReadFile(CaddyFilePath())
	if err != nil {
		return nil, err
	}
	if len(content) == 0 || string(content) == "" {
		return []string{}, nil
	}
	var hosts []string
	hostMap := make(map[string]bool)
	re := regexp.MustCompile(`(?m)^([a-zA-Z0-9.-_:/]+(?:,\s*[a-zA-Z0-9.-_:/]+)*)\s*\{`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) > 1 {
			rawBlockHeader := strings.TrimSpace(match[1])
			parts := strings.Split(rawBlockHeader, ",")
			for _, p := range parts {
				h := strings.TrimSpace(p)
				if h != "" && !hostMap[h] {
					hostMap[h] = true
					hosts = append(hosts, h)
				}
			}
		}
	}
	return hosts, nil
}
func CaddyFileToStruct(ctx context.Context, content string) (*dto.CaddyConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	jsonStr, err := gpagent.Do(callCtx, "CADDY_FILE_JSON", map[string]interface{}{"content": content})
	if err != nil {
		return nil, err
	}
	var cfg *dto.CaddyConfig
	if err := json.Unmarshal([]byte(jsonStr.Output), &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
func GetDomainsFromReq(primaryDomain string, otherDomains string) []string {
	domains := []string{primaryDomain}
	if otherDomains != "" {
		normalized := strings.ReplaceAll(otherDomains, ",", "\n")
		for _, d := range strings.Split(normalized, "\n") {
			d = strings.TrimSpace(d)
			if d != "" {
				domains = append(domains, d)
			}
		}
	}
	domains = utils.RemoveDuplicateStrings(domains)
	return domains
}
func CaddyGetDomainsConfigAsString(content string, primaryDomain string, otherDomains string) (string, error) {
	domains := GetDomainsFromReq(primaryDomain, otherDomains)
	if len(domains) == 0 {
		return "", fmt.Errorf("没有提供任何域名")
	}
	var matches []string
	for _, domain := range domains {
		reStr := fmt.Sprintf(`(?m)^(?:https?://)?%s\s*\{[^{}]*\}`, regexp.QuoteMeta(domain))
		re, err := regexp.Compile(reStr)
		if err != nil {
			return "", err
		}
		found := re.FindAllString(content, -1)
		matches = append(matches, found...)
	}
	return strings.Join(matches, "\n"), nil
}
func CaddyUpdateReplace(content, newContent, primaryDomain, otherDomains string) (string, error) {
	domains := GetDomainsFromReq(primaryDomain, otherDomains)
	if len(domains) == 0 {
		return "", fmt.Errorf("没有提供任何域名")
	}
	updatedContent := content
	for _, domain := range domains {
		reStr := fmt.Sprintf(`(?m)^(?:https?://)?%s\s*\{[^{}]*\}`, regexp.QuoteMeta(domain))
		re, err := regexp.Compile(reStr)
		if err != nil {
			return "", err
		}
		updatedContent = re.ReplaceAllString(updatedContent, newContent)
	}
	return updatedContent, nil
}
func CaddyUpdateOtherDomains(ctx context.Context, content string, primaryDomain, otherDomains, newOtherDomains string) (string, error) {
	domains := GetDomainsFromReq("", otherDomains)
	var newBlocks []string // 构建新的 server block 内容

	newDomainList := GetDomainsFromReq("", newOtherDomains)
	for _, d := range newDomainList {
		if d == "" {
			continue
		}
		block := fmt.Sprintf("\n%s {\n    redir %s permanent \n}\n", d, primaryDomain)
		newBlocks = append(newBlocks, block)
	}
	newContent := strings.Join(newBlocks, "\n")
	content = CaddyDeleteByDomain(content, domains)
	if newContent != "" {
		content += "\n"
		content += newContent
	}
	return content, nil
}
func CaddyUpdateProxy(content string, primaryDomain, newProxy string) (string, error) {
	if primaryDomain == "" || newProxy == "" || len(content) == 0 {
		return "", fmt.Errorf("参数 primaryDomain、newProxy 和 content 都不能为空")
	}
	reStr := fmt.Sprintf(`(?m)(^%s\s*\{[^{}]*reverse_proxy\s+)([^\s]+)([^{}]*\})`, regexp.QuoteMeta(primaryDomain))
	re, err := regexp.Compile(reStr)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(content, fmt.Sprintf("${1}%s${3}", newProxy)), nil
}
func CaddyDeleteByDomain(content string, targetDomains []string) string {
	result := content
	for _, domain := range targetDomains {
		var pattern string
		if len(domain) > 2 && domain[:2] == "*." {
			suffix := regexp.QuoteMeta(domain[1:])
			pattern = fmt.Sprintf(`(?ms)^[ \t]*(?:https?://)?[a-zA-Z0-9_-]+%s(:\d+)?\s*\{(?:[^{}]*|\{(?:[^{}]*|\{[^{}]*\})*\})*\}\s*`, suffix)
		} else {
			domainPattern := regexp.QuoteMeta(domain)
			pattern = fmt.Sprintf(`(?ms)^[ \t]*(?:https?://)?%s(:\d+)?\s*\{(?:[^{}]*|\{(?:[^{}]*|\{[^{}]*\})*\})*\}\s*`, domainPattern)
		}
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}
	return result
}
