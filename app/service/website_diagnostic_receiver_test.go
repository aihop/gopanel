package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cryptx"
)

func TestReceiveWebsiteDiagnosticEventVerifiesSignatureAndReplay(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	website := model.Website{PrimaryDomain: "example.com", Alias: "example.com", Protocol: "HTTPS"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	secret := "0123456789abcdef0123456789abcdef"
	encrypted, err := cryptx.AesEncrypt(secret, "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	setting := defaultWebsiteDiagnosticSetting(website.ID)
	setting.Enabled, setting.BackendHook, setting.HookSecretEncrypted = true, true, encrypted
	if err = database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	event := WebsiteDiagnosticEnvelope{Schema: websiteDiagnosticSchema, EventID: "remote-1", WebsiteID: website.ID, Source: "backend", Kind: "runtime_error", Severity: "error", OccurredAt: time.Now()}
	body, _ := json.Marshal(event)
	timestamp, nonce, path := strconv.FormatInt(time.Now().Unix(), 10), "0123456789abcdef", "/api/website-diagnostics/1/events"
	bodyHash := sha256.Sum256(body)
	canonical := timestamp + "\n" + nonce + "\nPOST\n" + path + "\n" + hex.EncodeToString(bodyHash[:])
	signer := hmac.New(sha256.New, []byte(secret))
	_, _ = signer.Write([]byte(canonical))
	signature := hex.EncodeToString(signer.Sum(nil))
	issue, err := ReceiveWebsiteDiagnosticEvent(website.ID, path, timestamp, nonce, signature, "https://example.com", "client", body)
	if err != nil || issue == nil {
		t.Fatalf("receive: issue=%#v err=%v", issue, err)
	}
	if _, err = ReceiveWebsiteDiagnosticEvent(website.ID, path, timestamp, nonce, signature, "https://example.com", "client", body); err == nil {
		t.Fatal("replayed event was accepted")
	}
}

func TestWebsiteIssueVerificationResolvesAndReopens(t *testing.T) {
	database := setupWebsiteDiagnosticTestDB(t)
	setting := defaultWebsiteDiagnosticSetting(1)
	setting.TriggerWindowMinutes = 1
	if err := database.Create(&setting).Error; err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-2 * time.Minute)
	issue := model.WebsiteIssue{WebsiteID: 1, Fingerprint: "verify", Status: "verifying", Severity: "error", Title: "failure", Kind: "runtime_error", FirstSeenAt: started, LastSeenAt: started, VerifyRelease: "v2", VerifyStartedAt: &started}
	if err := database.Create(&issue).Error; err != nil {
		t.Fatal(err)
	}
	resolvedAt := time.Now()
	if err := ReconcileWebsiteIssueVerification(resolvedAt); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&issue, issue.ID).Error; err != nil || issue.Status != "resolved" {
		t.Fatalf("issue=%#v err=%v", issue, err)
	}
	event := model.WebsiteDiagnosticEvent{WebsiteID: 1, EventID: "after-resolve", Source: "backend", Kind: issue.Kind, Severity: "error", Fingerprint: issue.Fingerprint, Title: issue.Title, Release: "v2", OccurredAt: resolvedAt.Add(time.Second)}
	updated, _, err := repo.NewWebsiteDiagnostic(database).IngestEvent(&event)
	if err != nil || updated.Status != "reopened" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
}
