package middleware

import "testing"

func TestMobileNodeProxyTargetAllowlist(t *testing.T) {
	tests := []struct {
		method  string
		path    string
		allowed bool
	}{
		{"GET", "/api/mobile/app/containers", true},
		{"POST", "mobile/app/containers/operate", true},
		{"GET", "mobile/app/containers/abc/publish-options", true},
		{"GET", "mobile/app/containers/abc/extra/publish-options", false},
		{"POST", "mobile/app/resources/websites/domains", true},
		{"GET", "mobile/app/sessions", false},
		{"POST", "mobile/app/containers/remove", false},
		{"GET", "user/list", false},
	}
	for _, test := range tests {
		if got := IsMobileNodeProxyTargetAllowed(test.method, test.path); got != test.allowed {
			t.Errorf("%s %s allowed = %v, want %v", test.method, test.path, got, test.allowed)
		}
	}
}
