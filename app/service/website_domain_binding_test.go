package service

import "testing"

func TestNormalizeWebsiteBindingInput(t *testing.T) {
	primary, others := normalizeWebsiteBindingInput(
		" https://Example.com ",
		"www.example.com, https://api.example.com\nexample.com\nWWW.example.com",
	)
	if primary != "Example.com" {
		t.Fatalf("unexpected primary domain: %q", primary)
	}
	if others != "www.example.com\napi.example.com" {
		t.Fatalf("unexpected additional domains: %q", others)
	}
}
