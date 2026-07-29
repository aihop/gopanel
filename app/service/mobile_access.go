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
	MobilePairingTTL = 2 * time.Minute
	MobileDeviceTTL  = 7 * 24 * time.Hour
)

func IssueMobilePairing(userID uint) (string, time.Time, error) {
	code, err := randomMobileSecret()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(MobilePairingTTL)
	err = repo.NewMobileAccessRepo().CreatePairing(&model.MobilePairing{
		UserID: userID, CodeHash: hashMobileSecret(code), ExpiresAt: expiresAt,
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
	device := &model.MobileDevice{
		Name:       normalizeMobileDeviceName(deviceName),
		TokenHash:  hashMobileSecret(token),
		ExpiresAt:  time.Now().Add(MobileDeviceTTL),
		LastIP:     ip,
		LastAgent:  truncateMobileValue(agent, 255),
		LastSeenAt: mobileTimePointer(time.Now()),
	}
	if err := repo.NewMobileAccessRepo().ConsumePairing(hashMobileSecret(code), device); err != nil {
		return "", nil, errors.New("配对码无效、已过期或已使用")
	}
	return token, device, nil
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
