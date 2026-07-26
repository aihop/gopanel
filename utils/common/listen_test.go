package common

import "testing"

func TestParseListenPort(t *testing.T) {
	cases := map[string]int{
		":5470":          5470,
		"5470":           5470,
		"0.0.0.0:5470":   5470,
		"127.0.0.1:8080": 8080,
		"[::1]:9000":     9000,
		" :5470 ":        5470,
		"":               0,
		":0":             0,
		":70000":         0,
		"abc":            0,
	}
	for raw, want := range cases {
		if got := ParseListenPort(raw); got != want {
			t.Errorf("ParseListenPort(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestNormalizeListenAddr(t *testing.T) {
	cases := map[string]string{
		"5470":         ":5470",
		":5470":        ":5470",
		"0.0.0.0:5470": "0.0.0.0:5470",
		" 5470 ":       ":5470",
		"":             "",
	}
	for raw, want := range cases {
		if got := NormalizeListenAddr(raw); got != want {
			t.Errorf("NormalizeListenAddr(%q) = %q, want %q", raw, got, want)
		}
	}
}
