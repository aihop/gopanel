package api

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// 探测只跑一次 ls-remote，超时给得比常规 fetch 短：
// 它是保存凭据时的同步动作，用户在界面上等着结果。
const codeGitCredentialProbeTimeout = 20 * time.Second

// validateCodeGitProbeRemote 限制可探测的远端地址。
//
// 这个地址来自用户输入并直接交给 git，不设限就是一个远程命令执行口子：
// Git 的 `ext::<command>` 传输会把 URL 里的内容当命令执行，
// `file://` 和本地路径则能被用来试探服务器上有哪些目录。
// 凭据是「用户名 + 令牌」，本来就只对 http(s)/ssh 有意义，收窄不损失能力。
func validateCodeGitProbeRemote(remote string) (string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", errors.New("请填写用于校验的仓库地址")
	}
	if len(remote) > 2048 {
		return "", errors.New("仓库地址过长")
	}
	if strings.ContainsAny(remote, "\r\n\x00") {
		return "", errors.New("仓库地址包含无效字符")
	}
	// 以 - 开头会被 git 当成选项；虽然调用处也用了 -- 分隔，这里再挡一道。
	if strings.HasPrefix(remote, "-") {
		return "", errors.New("仓库地址无效")
	}
	if strings.Contains(remote, "::") {
		return "", errors.New("不支持该传输方式的仓库地址")
	}
	lowered := strings.ToLower(remote)
	for _, scheme := range []string{"https://", "http://", "ssh://"} {
		if strings.HasPrefix(lowered, scheme) {
			return remote, nil
		}
	}
	// scp 简写：git@host:org/repo.git，@ 必须出现在第一个 / 之前。
	if at := strings.Index(remote, "@"); at > 0 {
		rest := remote[at+1:]
		if colon := strings.Index(rest, ":"); colon > 0 {
			if slash := strings.Index(rest, "/"); slash < 0 || colon < slash {
				return remote, nil
			}
		}
	}
	return "", errors.New("仅支持 https、http、ssh 或 git@host:path 形式的仓库地址")
}

// probeCodeGitCredentialRemote 用给定凭据访问一次远端仓库。
//
// 用 ls-remote 而不是 clone：它只走一次认证握手、不落盘，
// 是「这套凭据能不能读到这个仓库」最轻的问法。
func probeCodeGitCredentialRemote(username, secret, remote string) error {
	remote, err := validateCodeGitProbeRemote(remote)
	if err != nil {
		return err
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(secret) == "" {
		return errors.New("凭据信息不完整，无法校验")
	}
	env, cleanup, err := codeGitCredentialEnvironmentFor(username, secret, codeGitEnvironment())
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), codeGitCredentialProbeTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git",
		"-c", "credential.helper=",
		"ls-remote", "--heads", "--", remote,
	)
	command.Env = env
	output, runErr := command.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New("校验仓库连接超时，请确认网络与仓库地址")
	}
	if runErr != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = runErr.Error()
		}
		return normalizeCodeGitCommandError(message)
	}
	return nil
}
