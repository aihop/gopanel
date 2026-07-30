package api

import "testing"

func TestMobileContainerOperationAllowlist(t *testing.T) {
	for _, operation := range []string{"start", "stop", "restart"} {
		if !isMobileContainerOperationAllowed(operation) {
			t.Fatalf("expected %q to be allowed", operation)
		}
	}
	for _, operation := range []string{"remove", "kill", "pause", "up", ""} {
		if isMobileContainerOperationAllowed(operation) {
			t.Fatalf("expected %q to be rejected", operation)
		}
	}
}
