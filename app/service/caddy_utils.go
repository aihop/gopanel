package service

import (
	"fmt"
	"regexp"
)

func caddyDeleteByDomain(content []byte, targetDomains []string) []byte {
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
		result = re.ReplaceAll(result, []byte(""))
	}
	return result
}
