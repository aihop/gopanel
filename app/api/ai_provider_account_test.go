package api

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/aiprovider"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func withAIProviderDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "providers.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AIProviderAccount{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

func newProviderAccount(userID uint, name string, priority int, enabled, forMemory bool) *model.AIProviderAccount {
	return &model.AIProviderAccount{
		UserID: userID, Name: name, BaseURL: "https://example.com/v1", APIKey: "cipher",
		Protocol: aiprovider.ProtocolOpenAIChat, Model: "gpt-4o-mini",
		Enabled: enabled, UseForMemoryExtraction: forMemory, Priority: priority,
	}
}

// 自动挑选按优先级，与 codux 的 automatic 一致。
func TestSelectAIProviderAccountForMemoryPicksByPriority(t *testing.T) {
	withAIProviderDB(t)
	for _, account := range []*model.AIProviderAccount{
		newProviderAccount(3, "备用", 50, true, true),
		newProviderAccount(3, "首选", 10, true, true),
		newProviderAccount(3, "更低优先", 90, true, true),
	} {
		if err := global.DB.Create(account).Error; err != nil {
			t.Fatal(err)
		}
	}
	account, err := selectAIProviderAccountForMemory(3, 0)
	if err != nil || account.Name != "首选" {
		t.Fatalf("应挑优先级最高的：%#v, %v", account, err)
	}
}

// 抽取会把整段对话发出去，没打开授权开关的账号绝不能被自动选中。
func TestSelectAIProviderAccountForMemorySkipsUnauthorizedAndDisabled(t *testing.T) {
	withAIProviderDB(t)
	for _, account := range []*model.AIProviderAccount{
		newProviderAccount(3, "未授权抽取", 1, true, false),
		newProviderAccount(3, "已停用", 2, false, true),
		newProviderAccount(3, "可用", 80, true, true),
	} {
		if err := global.DB.Create(account).Error; err != nil {
			t.Fatal(err)
		}
	}
	account, err := selectAIProviderAccountForMemory(3, 0)
	if err != nil || account.Name != "可用" {
		t.Fatalf("只应选中已启用且已授权的账号：%#v, %v", account, err)
	}
}

// 指定的账号被停用或撤回授权后，不能静默回落到别的账号——
// 那等于把对话记录发去了用户没点头的地方。
func TestSelectAIProviderAccountForMemoryDoesNotFallBackFromRevokedAccount(t *testing.T) {
	withAIProviderDB(t)
	revoked := newProviderAccount(3, "已撤回授权", 1, true, false)
	if err := global.DB.Create(revoked).Error; err != nil {
		t.Fatal(err)
	}
	other := newProviderAccount(3, "另一个可用的", 2, true, true)
	if err := global.DB.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := selectAIProviderAccountForMemory(3, revoked.ID); err == nil {
		t.Fatal("指定账号已撤回授权时应报错，而不是换一个账号继续发")
	}
}

func TestSelectAIProviderAccountForMemoryIsolatesUsers(t *testing.T) {
	withAIProviderDB(t)
	foreign := newProviderAccount(99, "别人的账号", 1, true, true)
	if err := global.DB.Create(foreign).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := selectAIProviderAccountForMemory(3, 0); err == nil {
		t.Fatal("不该选中别的用户的账号")
	}
	if _, err := selectAIProviderAccountForMemory(3, foreign.ID); err == nil {
		t.Fatal("按 id 指定也不该跨用户命中")
	}
}

func TestCodeProviderRequestForAccountUsesOwnedEnabledAccount(t *testing.T) {
	withAIProviderDB(t)
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	ciphertext, err := encrypt.StringEncrypt("session-secret")
	if err != nil {
		t.Fatal(err)
	}
	account := newProviderAccount(3, "开发账号", 1, true, false)
	account.APIKey = ciphertext
	if err := global.DB.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	provider, err := codeProviderRequestForAccount(3, account.ID, "opencode")
	if err != nil || provider.APIKey != "session-secret" || provider.Model != account.Model {
		t.Fatalf("应解析当前用户启用的账号：%#v, %v", provider, err)
	}
	if _, err := codeProviderRequestForAccount(99, account.ID, "opencode"); err == nil {
		t.Fatal("不应允许其他用户使用该账号")
	}
	account.Enabled = false
	if err := global.DB.Save(account).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := codeProviderRequestForAccount(3, account.ID, "opencode"); err == nil {
		t.Fatal("不应允许会话使用已停用账号")
	}
}

func TestCodeProviderRequestForAccountRejectsProtocolMismatch(t *testing.T) {
	withAIProviderDB(t)
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	ciphertext, err := encrypt.StringEncrypt("session-secret")
	if err != nil {
		t.Fatal(err)
	}
	account := newProviderAccount(3, "Chat 账号", 1, true, false)
	account.APIKey = ciphertext
	if err := global.DB.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := codeProviderRequestForAccount(3, account.ID, "codex"); err == nil {
		t.Fatal("Codex 不应接受 Chat Completions 账号")
	}
}

func TestCodeExecutorSupportsProviderProtocol(t *testing.T) {
	tests := []struct {
		executor string
		protocol string
	}{
		{executor: "codex", protocol: aiprovider.ProtocolOpenAIResponses},
		{executor: "claude", protocol: aiprovider.ProtocolAnthropic},
		{executor: "opencode", protocol: aiprovider.ProtocolOpenAIChat},
		{executor: "aider", protocol: aiprovider.ProtocolOpenAIChat},
	}
	for _, test := range tests {
		if !codeExecutorSupportsProviderProtocol(test.executor, test.protocol) {
			t.Fatalf("%s 应支持 %s", test.executor, test.protocol)
		}
	}
	if codeExecutorSupportsProviderProtocol("codex", aiprovider.ProtocolOpenAIChat) {
		t.Fatal("Codex 不应接受 Chat Completions")
	}
}

func TestAIProviderAccountViewsDefaultsLegacyProtocol(t *testing.T) {
	views := aiProviderAccountViews([]model.AIProviderAccount{{Name: "legacy"}})
	if len(views) != 1 || views[0].Protocol != aiprovider.ProtocolOpenAIChat {
		t.Fatalf("旧账号应默认使用 Chat Completions：%#v", views)
	}
}

// 只发探测确认支持的参数：填了推理强度但模型不支持时，
// 正确的行为是不发，而不是发出去被 400。
func TestCodeMemoryOptionsForAccountRespectsProbedCapabilities(t *testing.T) {
	schema := map[string]any{"type": "object"}

	full := &model.AIProviderAccount{
		SupportsTemperature: true, SupportsJSONSchema: true,
		SupportsReasoningEffort: true, DefaultReasoningEffort: codeReasoningEffortHigh,
	}
	options := codeMemoryOptionsForAccount(full, schema)
	if options.Temperature == nil || *options.Temperature != 0 {
		t.Fatalf("支持时应发 temperature=0：%#v", options.Temperature)
	}
	if options.Schema == nil || options.ReasoningEffort != codeReasoningEffortHigh {
		t.Fatalf("支持时应发 schema 和推理强度：%#v", options)
	}

	// 推理模型的典型形态：拒绝 temperature，也不认 reasoning_effort。
	limited := &model.AIProviderAccount{
		SupportsTemperature: false, SupportsJSONSchema: false,
		SupportsReasoningEffort: false, DefaultReasoningEffort: codeReasoningEffortHigh,
	}
	options = codeMemoryOptionsForAccount(limited, schema)
	if options.Temperature != nil {
		t.Fatal("不支持时不该发 temperature")
	}
	if options.Schema != nil {
		t.Fatal("不支持时不该发 json_schema")
	}
	if options.ReasoningEffort != codeReasoningEffortNone {
		t.Fatalf("不支持时不该发推理强度：%q", options.ReasoningEffort)
	}
}

func TestNormalizeCodeReasoningEffortAcceptsKnownLevelsOnly(t *testing.T) {
	valid := map[string]string{
		"low": codeReasoningEffortLow, "LOW": codeReasoningEffortLow,
		" medium ": codeReasoningEffortMedium, "high": codeReasoningEffortHigh,
	}
	for input, expected := range valid {
		if actual := normalizeCodeReasoningEffort(input); actual != expected {
			t.Fatalf("%q 应归一为 %q，实际 %q", input, expected, actual)
		}
	}
	// 未知值一律当成「不设置」，交给服务端默认——
	// 猜一个值发出去，可能正好是这个服务不认的那个。
	for _, input := range []string{"", "maximum", "auto", "很高"} {
		if actual := normalizeCodeReasoningEffort(input); actual != codeReasoningEffortNone {
			t.Fatalf("%q 应归一为空，实际 %q", input, actual)
		}
	}
}

func TestNormalizeAIProviderPriorityClampsRange(t *testing.T) {
	cases := map[int]int{-5: 0, 0: 0, 10: 10, 999: 999, 5000: 999}
	for input, expected := range cases {
		if actual := normalizeAIProviderPriority(input); actual != expected {
			t.Fatalf("%d 应收敛为 %d，实际 %d", input, expected, actual)
		}
	}
}
