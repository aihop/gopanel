package service

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUpdateDownloadURLRequiresHTTPS(t *testing.T) {
	if err := validateUpdateDownloadURL("https://example.com/gopanel.tar.gz"); err != nil {
		t.Fatalf("valid HTTPS URL rejected: %v", err)
	}
	for _, value := range []string{"http://example.com/gopanel.tar.gz", "file:///tmp/gopanel.tar.gz", "not-a-url"} {
		if err := validateUpdateDownloadURL(value); err == nil {
			t.Fatalf("unsafe update URL accepted: %s", value)
		}
	}
}

func TestVerifyFileSHA256RejectsModifiedPackage(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "package.tar.gz")
	original := []byte("trusted package")
	if err := os.WriteFile(filePath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original)
	expected := hex.EncodeToString(sum[:])
	if err := verifyFileSHA256(filePath, expected); err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("modified package"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyFileSHA256(filePath, expected); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("modified package accepted: %v", err)
	}
}

func TestNormalizeSHA256RejectsMalformedValue(t *testing.T) {
	valid := strings.Repeat("a", 64)
	if actual, err := normalizeSHA256(strings.ToUpper(valid)); err != nil || actual != valid {
		t.Fatalf("valid checksum rejected: value=%s err=%v", actual, err)
	}
	for _, value := range []string{"short", strings.Repeat("z", 64)} {
		if _, err := normalizeSHA256(value); err == nil {
			t.Fatalf("malformed checksum accepted: %s", value)
		}
	}
}
