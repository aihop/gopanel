package api

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
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
		Model: "gpt-4o-mini", Enabled: enabled, UseForMemoryExtraction: forMemory, Priority: priority,
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
