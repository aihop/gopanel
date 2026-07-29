package middleware

import "testing"

func TestMobilePathsBypassSecurityEntrance(t *testing.T) {
	for _, path := range []string{
		"/mobile",
		"/mobile/auth",
		"/mobile/",
		"/api/mobile/pair/exchange",
		"/api/mobile/app/overview",
		"/assets/mobile.js",
	} {
		if !shouldBypassEntrance(path) {
			t.Fatalf("mobile path should bypass entrance: %s", path)
		}
	}
}

func TestMobileManagementKeepsSecurityEntrance(t *testing.T) {
	for _, path := range []string{
		"/api/mobile/management/pair/issue",
		"/api/mobile/management/devices",
		"/api/file/list",
	} {
		if shouldBypassEntrance(path) {
			t.Fatalf("management path must keep entrance protection: %s", path)
		}
	}
}
