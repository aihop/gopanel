package middleware

import "testing"

func TestHostTerminalOriginAllowed(t *testing.T) {
	tests := []struct {
		origin  string
		host    string
		allowed bool
	}{
		{"", "panel.example.com", true},
		{"https://panel.example.com", "panel.example.com", true},
		{"http://localhost:8080", "localhost:8080", true},
		{"https://evil.example.com", "panel.example.com", false},
		{"http://localhost:8081", "localhost:8080", false},
		{"not-a-url", "panel.example.com", false},
	}
	for _, test := range tests {
		if actual := isHostTerminalOriginAllowed(test.origin, test.host); actual != test.allowed {
			t.Errorf("origin=%q host=%q allowed=%v, want %v", test.origin, test.host, actual, test.allowed)
		}
	}
}
