package api

import (
	"errors"
	"regexp"
	"strings"
)

var (
	errCodeGitAuthentication = errors.New("Git 远端认证失败")
	errCodeGitRepository     = errors.New("Git 远端仓库不存在或无权访问")
	errCodeGitNetwork        = errors.New("Git 远端网络不可用")
	codeGitURLCredential     = regexp.MustCompile(`(?i)(https?://)[^\s/@]+@`)
)

func normalizeCodeGitCommandError(message string) error {
	message = redactCodeGitCommandOutput(message)
	if isCodeGitRepositoryError(message) {
		return errors.Join(errCodeGitRepository, errors.New(message), errors.New("请检查远端仓库地址与当前账号的仓库权限"))
	}
	if isCodeGitAuthenticationError(message) {
		return fmtCodeGitAuthenticationError()
	}
	if isCodeGitNetworkError(message) {
		return errors.Join(errCodeGitNetwork, errors.New(message), errors.New("请检查 GoPanel 服务的网络与代理配置后重试"))
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
		"http basic: access denied",
	} {
		if strings.Contains(lowerMessage, fragment) {
			return true
		}
	}
	return false
}

func isCodeGitRepositoryError(message string) bool {
	lowerMessage := strings.ToLower(message)
	return strings.Contains(lowerMessage, "repository not found") ||
		strings.Contains(lowerMessage, "unknown repository path") ||
		strings.Contains(lowerMessage, "未知的仓库路径")
}

func isCodeGitNetworkError(message string) bool {
	lowerMessage := strings.ToLower(message)
	for _, fragment := range []string{
		"failed to connect", "connection refused", "could not resolve host", "connection timed out",
		"network is unreachable", "proxy connect aborted", "tls handshake timeout",
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
		errors.New("请在 Code 项目设置中选择有效的 Git 凭据，或为 GoPanel 服务配置 credential helper / SSH 密钥后重试"),
	)
}
