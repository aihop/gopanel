package api

import (
	"strings"
	"testing"
)

// 泄漏在这里会被反复放大：终端里 echo 出来的一个 token，可能变成一条永久
// 记忆，然后出现在之后每个会话的上下文里。所以逐类都要验到。
func TestScrubCodeMemoryTextRedactsCredentials(t *testing.T) {
	cases := map[string]struct{ input, leaked string }{
		"OpenAI 令牌":       {"用 sk-abcdefghijklmnopqrstuvwxyz012345 调用", "sk-abcdefghijklmnopqrstuvwxyz012345"},
		"GitHub 经典令牌":     {"token ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123"},
		"GitHub 细粒度令牌":    {"github_pat_11ABCDE_xyz", "github_pat_11ABCDE_xyz"},
		"Slack 令牌":        {"xoxb-1234-5678-abcdefg", "xoxb-1234-5678-abcdefg"},
		"GitLab 令牌":       {"glpat-ABCdefGHIjklMNO", "glpat-ABCdefGHIjklMNO"},
		"AWS Access Key":  {"AKIAIOSFODNN7EXAMPLE 用于上传", "AKIAIOSFODNN7EXAMPLE"},
		"Authorization 头": {"Authorization: Bearer verysecretvalue", "verysecretvalue"},
		"环境变量密钥":          {"GITHUB_TOKEN=ghs_realsecretvalue", "ghs_realsecretvalue"},
		"环境变量口令":          {"DB_PASSWORD=hunter2", "hunter2"},
		"以 _KEY 结尾":       {"ENCRYPT_KEY=abc123def456", "abc123def456"},
		"长随机串":            {"key is Ab3Xy9Zq7Wm2Nk5Rt8Uv1Cd4Gh6Jl0Pw5", "Ab3Xy9Zq7Wm2Nk5Rt8Uv1Cd4Gh6Jl0Pw5"},
	}
	for name, testCase := range cases {
		scrubbed := scrubCodeMemoryText(testCase.input)
		if strings.Contains(scrubbed, testCase.leaked) {
			t.Fatalf("%s 未被脱敏：%q", name, scrubbed)
		}
		if !strings.Contains(scrubbed, "[REDACTED") {
			t.Fatalf("%s 应留下脱敏标记：%q", name, scrubbed)
		}
	}
}

// 私钥是多行的：只抹掉首行会把主体的 base64 原样留下。
func TestScrubCodeMemoryTextRemovesWholePrivateKeyBlock(t *testing.T) {
	input := strings.Join([]string{
		"部署时用到这个私钥：",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAAB",
		"AAAAMwAAAAtzc2gtZWQyNTUxOQAAACBsecretmaterialhere000",
		"-----END OPENSSH PRIVATE KEY-----",
		"之后重启服务。",
	}, "\n")
	scrubbed := scrubCodeMemoryText(input)
	for _, leaked := range []string{"b3BlbnNzaC1rZXktdjEA", "secretmaterialhere"} {
		if strings.Contains(scrubbed, leaked) {
			t.Fatalf("私钥主体泄漏：%q", scrubbed)
		}
	}
	if !strings.Contains(scrubbed, codeMemoryRedactedKeyMarker) {
		t.Fatalf("应留下私钥脱敏标记：%q", scrubbed)
	}
	// 前后的正常内容不能被一起吃掉。
	if !strings.Contains(scrubbed, "部署时用到这个私钥") || !strings.Contains(scrubbed, "之后重启服务") {
		t.Fatalf("私钥块前后的正文被误删：%q", scrubbed)
	}
}

// 抹掉 commit sha 会让「哪个提交引入了这个 bug」变成一串占位符，
// 而那正是 bug_lesson 类记忆最有价值的部分。
func TestScrubCodeMemoryTextKeepsHashesAndOrdinaryText(t *testing.T) {
	kept := []string{
		"回归出现在 8e9656596d803d8d6200e57ae9c2be49af3b2f83 这次提交",
		"文件校验和 d41d8cd98f00b204e9800998ecf8427e",
		"运行 go test ./... -count=1 验证",
		"分支 gopanel/code-108 已合入",
		"PATH=/usr/local/bin:/usr/bin",
		// sk- 会和语言地区码撞车，短主体不能被当成令牌抹掉。
		"locale 设为 sk-SK 时排序不对",
		"依赖 sk-learn 的版本要锁住",
	}
	for _, text := range kept {
		if scrubbed := scrubCodeMemoryText(text); scrubbed != text {
			t.Fatalf("正常内容被误伤：\n原文 %q\n结果 %q", text, scrubbed)
		}
	}
}

func TestScrubCodeMemoryTextHandlesEmptyInput(t *testing.T) {
	if scrubCodeMemoryText("") != "" {
		t.Fatal("空输入应原样返回")
	}
	if scrubCodeMemoryText("   ") != "   " {
		t.Fatal("空白输入应原样返回")
	}
}

// 标点紧跟令牌是最常见的形态（JSON、日志、句末），不能因此漏掉。
func TestScrubCodeMemoryTextRedactsTokensWithSurroundingPunctuation(t *testing.T) {
	for _, input := range []string{
		`{"token":"sk-abcdefghijklmnopqrstuvwxyz012345"}`,
		"密钥是 sk-abcdefghijklmnopqrstuvwxyz012345。",
		"(sk-abcdefghijklmnopqrstuvwxyz012345)",
	} {
		if strings.Contains(scrubCodeMemoryText(input), "sk-abcdefghijklmnopqrstuvwxyz012345") {
			t.Fatalf("带标点的令牌未被脱敏：%q", scrubCodeMemoryText(input))
		}
	}
}
