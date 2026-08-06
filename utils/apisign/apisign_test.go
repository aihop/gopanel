package apisign

import "testing"

func TestSignNormalizesAndExcludesAuthenticationQuery(t *testing.T) {
	first := Sign("secret", "1", "nonce", "post", "/api/test", "b=2&a=1&timestamp=1&apiKey=ignored&nonce=nonce&signatureVersion=v2", []byte("body"))
	second := Sign("secret", "1", "nonce", "POST", "/api/test", "a=1&b=2", []byte("body"))
	if first != second {
		t.Fatalf("normalized signatures differ: %s != %s", first, second)
	}
}
