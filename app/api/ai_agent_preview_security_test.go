package api

import (
	"net"
	"testing"
)

func TestPreviewProbeRejectsNonPublicAddresses(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", "0.0.0.0", "::1", "fc00::1", "fe80::1",
	}
	for _, raw := range blocked {
		if isPublicPreviewIP(net.ParseIP(raw)) {
			t.Fatalf("private preview address was allowed: %s", raw)
		}
	}
	for _, raw := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !isPublicPreviewIP(net.ParseIP(raw)) {
			t.Fatalf("public preview address was blocked: %s", raw)
		}
	}
}
