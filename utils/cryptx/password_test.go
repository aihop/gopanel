package cryptx

import (
	"strings"
	"testing"
)

func TestEncodePassword(t *testing.T) {
	hash, err := EncodePassword("strong-password")
	if err != nil {
		t.Fatal(err)
	}
	if !ValidatePassword(hash, "strong-password") {
		t.Fatal("generated password hash did not validate")
	}
}

func TestEncodePasswordReturnsBcryptError(t *testing.T) {
	if hash, err := EncodePassword(strings.Repeat("a", 73)); err == nil || hash != "" {
		t.Fatalf("expected bcrypt length error, got hash=%q err=%v", hash, err)
	}
}
