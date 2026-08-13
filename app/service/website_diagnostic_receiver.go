package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cryptx"
)

const WebsiteDiagnosticRemoteBodyLimit = 32 * 1024

var diagnosticReceiverRate = struct {
	sync.Mutex
	entries map[string][]time.Time
}{entries: make(map[string][]time.Time)}

func RotateWebsiteDiagnosticHookSecret(websiteID uint) (string, error) {
	setting, err := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(websiteID)
	if err != nil || setting == nil {
		return "", buserr.New("ErrWebsiteDiagnosticSettingRequired")
	}
	secretBytes := make([]byte, 32)
	if _, err = rand.Read(secretBytes); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(secretBytes)
	encrypted, err := cryptx.AesEncrypt(secret, "")
	if err != nil {
		return "", err
	}
	if err = repo.NewWebsiteDiagnostic(global.DB).SaveHookSecret(websiteID, encrypted); err != nil {
		return "", err
	}
	return secret, nil
}

func ReceiveWebsiteDiagnosticEvent(websiteID uint, requestPath, timestamp, nonce, signature, origin, rateKey string, body []byte) (*model.WebsiteIssue, error) {
	if len(body) == 0 || len(body) > WebsiteDiagnosticRemoteBodyLimit {
		return nil, buserr.New("ErrWebsiteDiagnosticRemoteBody")
	}
	setting, err := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(websiteID)
	if err != nil || setting == nil || !setting.Enabled || (!setting.BackendHook && !setting.BrowserHook) {
		return nil, buserr.New("ErrWebsiteDiagnosticReceiverDisabled")
	}
	website, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(websiteID))
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteNotFound")
	}
	if !websiteDiagnosticOriginAllowed(&website, origin) {
		return nil, buserr.New("ErrWebsiteDiagnosticOriginForbidden")
	}
	if !allowWebsiteDiagnosticRemoteRequest(strconv.FormatUint(uint64(websiteID), 10)+":"+rateKey, time.Now()) {
		return nil, buserr.New("ErrWebsiteDiagnosticRateLimited")
	}
	secret, err := cryptx.AesDecrypt(setting.HookSecretEncrypted, "")
	if err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticSecretRequired")
	}
	timestampValue, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Since(time.Unix(timestampValue, 0)).Abs() > 5*time.Minute {
		return nil, buserr.New("ErrWebsiteDiagnosticSignatureExpired")
	}
	nonce = limitedDiagnosticText(nonce, 128)
	if nonce == "" || len(nonce) < 16 {
		return nil, buserr.New("ErrWebsiteDiagnosticNonceInvalid")
	}
	bodyHash := sha256.Sum256(body)
	canonical := strings.Join([]string{timestamp, nonce, "POST", requestPath, hex.EncodeToString(bodyHash[:])}, "\n")
	expected := hmac.New(sha256.New, []byte(secret))
	_, _ = expected.Write([]byte(canonical))
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || !hmac.Equal(expected.Sum(nil), provided) {
		return nil, buserr.New("ErrWebsiteDiagnosticSignatureInvalid")
	}
	if err = repo.NewWebsiteDiagnostic(global.DB).ClaimNonce(websiteID, nonce, time.Now().Add(10*time.Minute)); err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticReplay")
	}
	var envelope WebsiteDiagnosticEnvelope
	if err = json.Unmarshal(body, &envelope); err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticInvalidEvent")
	}
	issue, _, err := ingestWebsiteDiagnosticEnvelope(websiteID, &envelope)
	return issue, err
}

func websiteDiagnosticOriginAllowed(website *model.Website, origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == strings.ToLower(strings.Split(website.PrimaryDomain, ":")[0]) {
		return true
	}
	for _, domain := range website.Domains {
		if host == strings.ToLower(strings.Split(domain.Domain, ":")[0]) {
			return true
		}
	}
	return false
}

func allowWebsiteDiagnosticRemoteRequest(key string, now time.Time) bool {
	diagnosticReceiverRate.Lock()
	defer diagnosticReceiverRate.Unlock()
	cutoff := now.Add(-time.Minute)
	recent := diagnosticReceiverRate.entries[key][:0]
	for _, item := range diagnosticReceiverRate.entries[key] {
		if item.After(cutoff) {
			recent = append(recent, item)
		}
	}
	if len(recent) >= 60 {
		diagnosticReceiverRate.entries[key] = recent
		return false
	}
	diagnosticReceiverRate.entries[key] = append(recent, now)
	return true
}
