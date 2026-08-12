package api

import (
	"strings"
)

const codeMemoryRedactedMarker = "[REDACTED_SECRET]"
const codeMemoryRedactedKeyMarker = "[REDACTED_PRIVATE_KEY]"

// scrubCodeMemoryText 在会话记录离开本机之前抹掉其中的凭据。
//
// 这一层是必须的而不是加分项：抽取要把整段对话发给 LLM，抽出来的记忆还会
// 长期存库并在之后每次执行时回灌。一次泄漏会被反复放大——终端里 echo 出来的
// 一个 token，可能变成一条永久记忆，然后在之后每个会话的上下文里出现。
//
// 四道各管一类，顺序有讲究：先切掉多行私钥块（它内部的 base64 会被后面的
// 令牌规则逐行误伤成一堆碎片），再处理行级的 header 与环境变量，最后才是
// 逐 token 的兜底。
func scrubCodeMemoryText(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	scrubbed := redactCodeMemoryPrivateKeyBlocks(text)
	scrubbed = redactCodeMemoryAuthorizationHeaders(scrubbed)
	scrubbed = redactCodeMemoryEnvSecrets(scrubbed)
	return redactCodeMemoryTokens(scrubbed)
}

// redactCodeMemoryPrivateKeyBlocks 整块抹掉 PEM 私钥。
// 按「BEGIN ... PRIVATE KEY」进入、「END ... PRIVATE KEY」退出，
// 中间不论多少行一律丢弃。
func redactCodeMemoryPrivateKeyBlocks(text string) string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		upper := strings.ToUpper(line)
		if !inBlock {
			if strings.Contains(upper, "-----BEGIN") && strings.Contains(upper, "PRIVATE KEY") {
				inBlock = true
				result = append(result, codeMemoryRedactedKeyMarker)
				// 单行就写完整块的情况（BEGIN 和 END 同一行）。
				if strings.Contains(upper, "-----END") {
					inBlock = false
				}
				continue
			}
			result = append(result, line)
			continue
		}
		if strings.Contains(upper, "-----END") && strings.Contains(upper, "PRIVATE KEY") {
			inBlock = false
		}
	}
	return strings.Join(result, "\n")
}

func redactCodeMemoryAuthorizationHeaders(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if !strings.HasPrefix(lower, "authorization:") && !strings.HasPrefix(lower, "proxy-authorization:") {
			continue
		}
		if colon := strings.Index(line, ":"); colon >= 0 {
			lines[index] = line[:colon+1] + " " + codeMemoryRedactedMarker
		}
	}
	return strings.Join(lines, "\n")
}

// redactCodeMemoryEnvSecrets 处理 KEY=value 形式，键名看着像密钥就抹掉值。
func redactCodeMemoryEnvSecrets(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		equals := strings.Index(line, "=")
		if equals < 0 {
			continue
		}
		key := strings.TrimSpace(line[:equals])
		if key == "" || strings.TrimSpace(line[equals+1:]) == "" {
			continue
		}
		if !looksLikeCodeMemorySecretKey(key) {
			continue
		}
		lines[index] = key + "=" + codeMemoryRedactedMarker
	}
	return strings.Join(lines, "\n")
}

func looksLikeCodeMemorySecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, marker := range []string{"SECRET", "TOKEN", "PASSWORD", "PASSWD", "API_KEY", "APIKEY", "PRIVATE_KEY", "CREDENTIAL"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return strings.HasSuffix(upper, "_KEY")
}

// redactCodeMemoryTokens 是逐 token 的兜底：认已知前缀，外加「够长的纯字母数字串」。
func redactCodeMemoryTokens(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	var token strings.Builder
	flush := func() {
		if token.Len() > 0 {
			builder.WriteString(redactCodeMemoryToken(token.String()))
			token.Reset()
		}
	}
	for _, character := range text {
		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			flush()
			builder.WriteRune(character)
			continue
		}
		token.WriteRune(character)
	}
	flush()
	return builder.String()
}

// 每个前缀配一个「主体最少多长」的门槛。
//
// 门槛不能一刀切：github_pat_ / glpat- / xoxb- 这些前缀本身就是唯一的，
// 见到即可判定；而 sk- 会和 sk-SK 这类语言地区码、sk-learn 这类缩写撞车，
// 必须要求足够长的主体才认。一刀切要么漏掉真令牌，要么误伤正常文本。
var codeMemorySecretPrefixes = []struct {
	prefix     string
	minBodyLen int
}{
	{"github_pat_", 8},
	{"glpat-", 8},
	{"xoxb-", 8},
	{"xoxp-", 8},
	{"ghp_", 8},
	{"ghs_", 8},
	{"gho_", 8},
	{"sk-", 16},
}

func redactCodeMemoryToken(token string) string {
	// 先按已知前缀在 token 内部查找。日志和 JSON 里令牌常常被引号、冒号
	// 紧紧裹住（{"token":"sk-…"}），按空白切出来的 token 并不以前缀开头，
	// 只看开头会整片漏掉——而那恰恰是最常见的形态。
	for _, candidate := range codeMemorySecretPrefixes {
		index := strings.Index(token, candidate.prefix)
		if index < 0 {
			continue
		}
		end := index + len(candidate.prefix)
		for end < len(token) && isCodeMemorySecretBodyByte(token[end]) {
			end++
		}
		if end-index-len(candidate.prefix) < candidate.minBodyLen {
			continue
		}
		return token[:index] + codeMemoryRedactedMarker + redactCodeMemoryToken(token[end:])
	}
	if index := strings.Index(token, "AKIA"); index >= 0 {
		end := index + len("AKIA")
		for end < len(token) && isCodeMemorySecretBodyByte(token[end]) {
			end++
		}
		if end-index >= 16 {
			return token[:index] + codeMemoryRedactedMarker + redactCodeMemoryToken(token[end:])
		}
	}
	trimmed := strings.Trim(token, ",;:\"'()[]{}")
	if trimmed == "" || !looksLikeCodeMemoryOpaqueSecret(trimmed) {
		return token
	}
	return strings.Replace(token, trimmed, codeMemoryRedactedMarker, 1)
}

func isCodeMemorySecretBodyByte(character byte) bool {
	switch {
	case character >= 'a' && character <= 'z':
	case character >= 'A' && character <= 'Z':
	case character >= '0' && character <= '9':
	case character == '-' || character == '_':
	default:
		return false
	}
	return true
}

// looksLikeCodeMemoryOpaqueSecret 处理没有已知前缀的长随机串。
// commit sha、内容哈希也长这样——抹掉它们会让记忆里的
// 「哪个提交引入了这个 bug」变成一串占位符，所以纯十六进制的放行。
func looksLikeCodeMemoryOpaqueSecret(token string) bool {
	return len(token) >= 32 && isCodeMemoryAlphanumeric(token) && !looksLikeCodeMemoryHash(token)
}

func isCodeMemoryAlphanumeric(token string) bool {
	for _, character := range token {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		default:
			return false
		}
	}
	return true
}

func looksLikeCodeMemoryHash(token string) bool {
	if len(token) < 32 {
		return false
	}
	for _, character := range token {
		switch {
		case character >= '0' && character <= '9':
		case character >= 'a' && character <= 'f':
		case character >= 'A' && character <= 'F':
		default:
			return false
		}
	}
	return true
}
