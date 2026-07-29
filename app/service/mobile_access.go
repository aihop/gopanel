package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
)

const (
	MobilePairingTTL           = 2 * time.Minute
	DefaultMobileDeviceTTLDays = 30
)

var allowedMobileDeviceTTLDays = map[int]struct{}{
	1: {}, 7: {}, 30: {}, 90: {}, 365: {},
}

func IssueMobilePairing(userID uint, requestedTTLDays int) (string, time.Time, error) {
	deviceTTLDays, err := normalizeMobileDeviceTTLDays(requestedTTLDays)
	if err != nil {
		return "", time.Time{}, err
	}
	code, err := randomMobileSecret()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(MobilePairingTTL)
	err = repo.NewMobileAccessRepo().CreatePairing(&model.MobilePairing{
		UserID: userID, CodeHash: hashMobileSecret(code), DeviceTTLDays: deviceTTLDays, ExpiresAt: expiresAt,
	})
	return code, expiresAt, err
}

func ExchangeMobilePairing(code, deviceName, ip, agent string) (string, *model.MobileDevice, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil, errors.New("配对码不能为空")
	}
	token, err := randomMobileSecret()
	if err != nil {
		return "", nil, err
	}
	codeHash := hashMobileSecret(code)
	pairing, err := repo.NewMobileAccessRepo().FindPairing(codeHash)
	if err != nil {
		return "", nil, errors.New("配对码无效、已过期或已使用")
	}
	deviceTTLDays, err := normalizeMobileDeviceTTLDays(pairing.DeviceTTLDays)
	if err != nil {
		return "", nil, errors.New("配对码授权期限无效")
	}
	device := &model.MobileDevice{
		Name:       normalizeMobileDeviceName(deviceName),
		TokenHash:  hashMobileSecret(token),
		ExpiresAt:  time.Now().Add(time.Duration(deviceTTLDays) * 24 * time.Hour),
		LastIP:     ip,
		LastAgent:  truncateMobileValue(agent, 255),
		LastSeenAt: mobileTimePointer(time.Now()),
	}
	if err := repo.NewMobileAccessRepo().ConsumePairing(codeHash, device); err != nil {
		return "", nil, errors.New("配对码无效、已过期或已使用")
	}
	return token, device, nil
}

func normalizeMobileDeviceTTLDays(requested int) (int, error) {
	if requested == 0 {
		return DefaultMobileDeviceTTLDays, nil
	}
	if _, allowed := allowedMobileDeviceTTLDays[requested]; !allowed {
		return 0, errors.New("手机授权期限无效")
	}
	return requested, nil
}

func AuthenticateMobileDevice(token, ip, agent string) (*model.MobileDevice, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("手机授权已失效")
	}
	device, err := repo.NewMobileAccessRepo().FindDevice(hashMobileSecret(token))
	if err != nil {
		return nil, errors.New("手机授权已失效")
	}
	_ = repo.NewMobileAccessRepo().TouchDevice(device.ID, ip, truncateMobileValue(agent, 255))
	return device, nil
}

func randomMobileSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func hashMobileSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func normalizeMobileDeviceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "手机浏览器"
	}
	return truncateMobileValue(name, 128)
}

func truncateMobileValue(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes)
}

func mobileTimePointer(value time.Time) *time.Time {
	return &value
}
