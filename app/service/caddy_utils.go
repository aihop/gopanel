package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpagent"
)

type caddyGetConfigResp struct {
	Caddyfile string `json:"caddyfile"`
}

// 获取配置文件
func CaddyFilePath() string {
	return utils.GetConfigPath(global.CONF.System.BaseDir, "Caddyfile")
}

func CaddyContent() (string, error) {
	content, err := os.ReadFile(CaddyFilePath())
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(content), nil
}

func GetCaddyFile(ctx context.Context) (string, error) {
	resp, err := gpagent.Do(ctx, "CADDY_CONFIG", nil)
	if err != nil {
		return "", err
	}
	var out caddyGetConfigResp
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return "", err
	}
	return out.Caddyfile, nil
}

// 应用，相当于就是重启等
func CaddyApplyCaddyFile(ctx context.Context, content string) error {
	_, err := gpagent.Do(ctx, "CADDY_APPLY", map[string]interface{}{"caddyfile": content})
	return err
}

// 添加服务器块
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

// 替换服务器块
func CaddyReplaceServerBlock(ctx context.Context, domain, proxy, otherDomains, protocol string) (bool, error) {
	if _, err := CaddyRemoveServerBlock(ctx, domain, otherDomains); err != nil {
		return false, err
	}
	return CaddyAddServerBlock(ctx, domain, proxy, otherDomains, protocol)
}

// 添加服务器路径块
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
	// 如果协议是 HTTP，添加 HTTP 重定向
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
	// 域名必须已存在
	exist, err := CaddyExistAddress(domain)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, fmt.Errorf("域名还没有绑定，无法给扩展域名做跳转")
	}
	// 解析 extraDomains：支持换行或逗号分隔
	normalized := strings.ReplaceAll(extraDomains, ",", "\n")
	lines := strings.Split(normalized, "\n")

	var blocks []string
	for _, ln := range lines {
		d := strings.TrimSpace(ln)
		if d == "" {
			continue
		}
		// 如果扩展域名已经存在于配置中，跳过
		ex, err := CaddyExistAddress(d)
		if err != nil {
			return false, err
		}
		if ex {
			continue
		}
		// 生成 301 重定向到主域名，保留 URI 和查询参数
		block := fmt.Sprintf("\n%s {\n    redir %s{uri} permanent \n}\n", d, buildCaddyRedirectTarget(domain))
		blocks = append(blocks, block)
	}

	if len(blocks) == 0 {
		// 没有可添加的域名（都已存在或输入为空）
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

	// 统一处理主域名和扩展域名
	var domains []string
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
		// Replace regex with an exact bracket-counting block deleter.
		lines := strings.Split(trimmed, "\n")
		var result []string
		inTargetBlock := false
		bracketCount := 0

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			if !inTargetBlock {
				// Check if this line starts the block we want to delete
				// A block header can be multiple domains separated by comma
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
							inTargetBlock = false // one-liner block
						}
						deleted = true
						continue
					}
				}
				result = append(result, line)
			} else {
				// We are inside the block we want to delete
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
	trimmedBytes := strings.TrimSpace(trimmed) // 去掉首尾空行
	// 写回文件
	if deleted {
		if err = CaddySaveContent(ctx, trimmedBytes); err != nil {
			return false, err
		}
	}
	return deleted, nil
}

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

			// We might have comma separated multiple domains in one block header
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

	jsonStr, err := gpagent.Do(callCtx, "CADDY_FILE_JSON", map[string]interface{}{
		"content": content,
	})
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
	// 这里我要提取caddyfile中，所有以domain开头的server块
	for _, domain := range domains {
		// 匹配所有以 domain 开头的 server block（支持多行内容），要支持http 和 https
		// (?m) 多行模式，^ 和 $ 匹配每行的开头和结尾
		// \s* 匹配空白字符（包括换行）
		// [^{}]* 匹配非大括号的任意字符（非贪婪）
		// 支持 http://、https://、无协议三种写法
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
	// 将 content 中，所有以 domains 开头的 server block 替换为 newContent
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
	// 构建新的 server block 内容
	var newBlocks []string
	newDomainList := GetDomainsFromReq("", newOtherDomains)
	for _, d := range newDomainList {
		if d == "" {
			continue // 已存在的域名跳过
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

func CaddySaveContent(ctx context.Context, content string) error {
	fileUtil := files.NewFileOp()
	filePath := CaddyFilePath()
	backup, err := CaddyContent()
	if err != nil {
		return err
	}
	if err := fileUtil.SaveFileWithByte(filePath, []byte(content), 0644); err != nil {
		return err
	}
	if err := CaddyApplyCaddyFile(ctx, content); err != nil {
		if err != nil {
			_ = fileUtil.SaveFileWithByte(filePath, []byte(backup), 0644)
			return err
		}
	}
	return nil
}

func CaddyUpdateProxy(content string, primaryDomain, newProxy string) (string, error) {
	if primaryDomain == "" || newProxy == "" || len(content) == 0 {
		return "", fmt.Errorf("参数 primaryDomain、newProxy 和 content 都不能为空")
	}
	// 查找 primaryDomain 对应的 server block，并替换其中的 reverse_proxy 指令
	reStr := fmt.Sprintf(`(?m)(^%s\s*\{[^{}]*reverse_proxy\s+)([^\s]+)([^{}]*\})`, regexp.QuoteMeta(primaryDomain))
	re, err := regexp.Compile(reStr)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(content, fmt.Sprintf("${1}%s${3}", newProxy)), nil
}

func CaddyDeleteByDomain(content string, targetDomains []string) string {
	// 不要把所有内容变成一行，直接用多行正则
	result := content
	for _, domain := range targetDomains {
		var pattern string
		if len(domain) > 2 && domain[:2] == "*." {
			// 泛域名，匹配任意子域名
			suffix := regexp.QuoteMeta(domain[1:])
			// 匹配 http/https/无协议/端口
			// 最多三层嵌套
			pattern = fmt.Sprintf(`(?ms)^[ \t]*(?:https?://)?[a-zA-Z0-9_-]+%s(:\d+)?\s*\{(?:[^{}]*|\{(?:[^{}]*|\{[^{}]*\})*\})*\}\s*`, suffix)
		} else {
			// 精确域名
			domainPattern := regexp.QuoteMeta(domain)
			pattern = fmt.Sprintf(`(?ms)^[ \t]*(?:https?://)?%s(:\d+)?\s*\{(?:[^{}]*|\{(?:[^{}]*|\{[^{}]*\})*\})*\}\s*`, domainPattern)
		}
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}
	return result
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
