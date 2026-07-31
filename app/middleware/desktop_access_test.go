package middleware

import "testing"

func TestMatchesDesktopAccess(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		provided string
		expected string
		want     bool
	}{
		{name: "loopback token", ip: "127.0.0.1", provided: "desktop-secret", expected: "desktop-secret", want: true},
		{name: "ipv6 loopback token", ip: "::1", provided: "desktop-secret", expected: "desktop-secret", want: true},
		{name: "invalid token", ip: "127.0.0.1", provided: "invalid", expected: "desktop-secret"},
		{name: "remote address", ip: "192.168.1.10", provided: "desktop-secret", expected: "desktop-secret"},
		{name: "disabled", ip: "127.0.0.1", provided: "desktop-secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesDesktopAccess(test.ip, test.provided, test.expected); got != test.want {
				t.Fatalf("matchesDesktopAccess() = %v, want %v", got, test.want)
			}
		})
	}
}
