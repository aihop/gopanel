package service

import (
	"errors"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	valid := []string{"server-01", "Web01", "node-1.example.com", "a"}
	for _, hostname := range valid {
		if err := validateHostname(hostname); err != nil {
			t.Fatalf("expected %q to be valid: %v", hostname, err)
		}
	}

	invalid := []string{"", "服务器", "server_name", "-server", "server-", "a..b", "server name", "example.com."}
	for _, hostname := range invalid {
		if err := validateHostname(hostname); !errors.Is(err, ErrHostnameInvalid) {
			t.Fatalf("expected %q to be invalid, got: %v", hostname, err)
		}
	}
}

func TestValidateHostnameLengthLimits(t *testing.T) {
	label64 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := validateHostname(label64); !errors.Is(err, ErrHostnameInvalid) {
		t.Fatalf("expected 64-character label to be invalid, got: %v", err)
	}

	hostname254 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa." +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb." +
		"ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc." +
		"ddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	if len(hostname254) != 255 {
		t.Fatalf("test fixture length is %d", len(hostname254))
	}
	if err := validateHostname(hostname254); !errors.Is(err, ErrHostnameInvalid) {
		t.Fatalf("expected overlong hostname to be invalid, got: %v", err)
	}
}
