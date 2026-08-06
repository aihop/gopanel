package apisign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxNonces = 20000

var authQueryKeys = map[string]struct{}{
	"apikey": {}, "timestamp": {}, "nonce": {}, "signatureversion": {},
}

type nonceStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

var usedNonces = &nonceStore{seen: make(map[string]time.Time)}

func Sign(secret, timestamp, nonce, method, path, rawQuery string, body []byte) string {
	payload := timestamp + "\n" + nonce + "\n" + strings.ToUpper(method) + "\n" + path + "\n" + NormalizeQuery(rawQuery) + "\n" + BodyHash(body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func Equal(expected, actual string) bool {
	return hmac.Equal([]byte(expected), []byte(actual))
}

func BodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func NormalizeQuery(rawQuery string) string {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}
	for key := range values {
		if _, isAuth := authQueryKeys[strings.ToLower(key)]; isAuth {
			values.Del(key)
		}
	}
	return values.Encode()
}

func ConsumeNonce(nonce string, validity time.Duration) bool {
	if strings.TrimSpace(nonce) == "" || validity <= 0 {
		return false
	}
	now := time.Now()
	usedNonces.mu.Lock()
	defer usedNonces.mu.Unlock()
	for key, at := range usedNonces.seen {
		if now.Sub(at) > validity {
			delete(usedNonces.seen, key)
		}
	}
	if len(usedNonces.seen) >= maxNonces {
		usedNonces.seen = make(map[string]time.Time)
	}
	if _, exists := usedNonces.seen[nonce]; exists {
		return false
	}
	usedNonces.seen[nonce] = now
	return true
}
