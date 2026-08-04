package api

import (
	"errors"
	"regexp"
	"strings"
)

var (
	errCodeGitAuthentication = errors.New("Git 远端认证失败")
	codeGitURLCredential     = regexp.MustCompile(`(?i)(https?://)[^\s/@]+@`)
)

func normalizeCodeGitCommandError(message string) error {
	message = redactCodeGitCommandOutput(message)
	if isCodeGitAuthenticationError(message) {
		return fmtCodeGitAuthenticationError()
	}
	return errors.New(message)
}

func redactCodeGitCommandOutput(message string) string {
	return codeGitURLCredential.ReplaceAllString(message, `${1}***@`)
}

func isCodeGitAuthenticationError(message string) bool {
	lowerMessage := strings.ToLower(message)
	for _, fragment := range []string{
		"authentication failed",
		"could not read username",
		"could not read password",
		"unable to get password",
		"terminal prompts disabled",
		"permission denied (publickey)",
		"repository not found",
		"http basic: access denied",
	} {
		if strings.Contains(lowerMessage, fragment) {
			return true
		}
	}
	return false
}

func fmtCodeGitAuthenticationError() error {
	return errors.Join(
		errCodeGitAuthentication,
		errors.New("请为 GoPanel 服务运行账户配置有效的 Git credential helper 或 SSH 密钥后重试"),
	)
}
