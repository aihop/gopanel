package api

import "testing"

func TestNormalizeCodeSessionTitle(t *testing.T) {
	title, err := normalizeCodeSessionTitle("  移动端修复  ")
	if err != nil || title != "移动端修复" {
		t.Fatalf("normalized title = %q, err=%v", title, err)
	}
	if _, err := normalizeCodeSessionTitle("   "); err == nil {
		t.Fatal("empty title should be rejected")
	}
	longTitle := make([]rune, 256)
	for index := range longTitle {
		longTitle[index] = '会'
	}
	if _, err := normalizeCodeSessionTitle(string(longTitle)); err == nil {
		t.Fatal("title longer than 255 characters should be rejected")
	}
}
