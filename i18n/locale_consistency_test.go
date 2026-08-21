package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestLocaleConsistency 验证 resource/locale 下 zh.yaml 与 en.yaml 顶层 key 完全一致，
// 且都能成功解析。任何单方面新增/删除 key 都会让该测试失败。
func TestLocaleConsistency(t *testing.T) {
	root, err := findLocaleRoot(t)
	if err != nil {
		t.Skip(err)
	}
	zhPath := filepath.Join(root, "zh.yaml")
	enPath := filepath.Join(root, "en.yaml")

	zhKeys := loadTopLevelKeys(t, zhPath)
	enKeys := loadTopLevelKeys(t, enPath)

	if len(zhKeys) != len(enKeys) {
		t.Errorf("key count mismatch: zh=%d en=%d", len(zhKeys), len(enKeys))
	}

	missing := diffKeys(zhKeys, enKeys)
	if len(missing) > 0 {
		t.Errorf("keys missing from en.yaml: %v", missing)
	}
	extra := diffKeys(enKeys, zhKeys)
	if len(extra) > 0 {
		t.Errorf("keys extra in en.yaml: %v", extra)
	}
}

// TestLocaleTemplateVars 验证同一 key 在两种语言下使用的模板变量保持一致，
// 避免 zh 用 {{ .name }} 而 en 用 {{ .user }} 导致 fallback 失效。
func TestLocaleTemplateVars(t *testing.T) {
	root, err := findLocaleRoot(t)
	if err != nil {
		t.Skip(err)
	}
	zh := loadRawTemplate(t, filepath.Join(root, "zh.yaml"))
	en := loadRawTemplate(t, filepath.Join(root, "en.yaml"))

	if len(zh) != len(en) {
		t.Errorf("key count mismatch: zh=%d en=%d", len(zh), len(en))
	}

	for key, zhVal := range zh {
		enVal, ok := en[key]
		if !ok {
			t.Errorf("key %q missing in en.yaml", key)
			continue
		}
		zhVars := extractTemplateVars(zhVal)
		enVars := extractTemplateVars(enVal)
		if !equalStringSets(zhVars, enVars) {
			t.Errorf("template vars mismatch for %q: zh=%v en=%v", key, zhVars, enVars)
		}
	}
}

// findLocaleRoot 查找 resource/locale 目录以支持从 repo 根或子目录运行 go test。
func findLocaleRoot(t *testing.T) (string, error) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "resource", "locale")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// loadTopLevelKeys 读取 yaml 顶层 key 列表，忽略 value。
func loadTopLevelKeys(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// loadRawTemplate 读取 yaml 原始 map（key -> string value），仅保留 string 类型的值。
func loadRawTemplate(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	out := make(map[string]string, len(data))
	for k, v := range data {
		if s, ok := v.(string); ok {
			out[k] = s
		} else if v == nil {
			out[k] = ""
		}
	}
	return out
}

// extractTemplateVars 从形如 "hello {{ .name }} age {{ .age }}" 的字符串中提取所有变量名。
func extractTemplateVars(s string) []string {
	re := templateVarRe.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, m := range re {
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

var templateVarRe = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*(\|[^}]*)?\}\}`)

func diffKeys(a, b []string) []string {
	set := map[string]bool{}
	for _, k := range b {
		set[k] = true
	}
	out := []string{}
	for _, k := range a {
		if !set[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
