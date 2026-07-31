package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMobileResourceSummariesDoNotExposeSecrets(t *testing.T) {
	values := []any{
		mobileWebsiteSummary{ID: 1, PrimaryDomain: "example.com"},
		mobileDatabaseSummary{Name: "app", Server: "local"},
		mobileSSLSummary{ID: 2, PrimaryDomain: "example.com", ExpireDate: time.Now()},
		mobileAppSummary{ID: 3, Name: "demo"},
	}
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		serialized := strings.ToLower(string(data))
		for _, secretField := range []string{"password", "privatekey", "pem", "env", "dockercompose"} {
			if strings.Contains(serialized, secretField) {
				t.Fatalf("mobile resource summary exposes %s: %s", secretField, serialized)
			}
		}
	}
}
