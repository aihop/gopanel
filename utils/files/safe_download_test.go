package files

import (
	"context"
	"net/netip"
	"testing"
)

func TestValidateDownloadURL(t *testing.T) {
	for _, rawURL := range []string{"file:///etc/passwd", "ftp://example.com/file", "http://user:pass@example.com/file", "http:///file", "http://127.0.0.1/file", "http://169.254.169.254/latest/meta-data"} {
		if err := validateDownloadURL(rawURL); err == nil {
			t.Fatalf("expected %q to be rejected", rawURL)
		}
	}
	if err := validateDownloadURL("https://example.com/file"); err != nil {
		t.Fatalf("expected public HTTPS URL to pass syntax validation: %v", err)
	}
}

func TestSafeDownloadRejectsMetadataAddressBeforeCreatingFile(t *testing.T) {
	dst := t.TempDir() + "/download"
	err := DownloadFileWithCallbackSafe(context.Background(), "http://169.254.169.254/latest/meta-data", dst, false, nil)
	if err == nil {
		t.Fatal("expected metadata address to be rejected")
	}
}

func TestBlockedDownloadAddresses(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "::1", "fc00::1", "fe80::1", "2001:db8::1"}
	for _, value := range blocked {
		if !isBlockedDownloadAddr(netip.MustParseAddr(value)) {
			t.Fatalf("expected %s to be blocked", value)
		}
	}
	if isBlockedDownloadAddr(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("expected public address to be allowed")
	}
}
