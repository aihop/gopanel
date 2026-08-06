package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAuthenticateMobileDeviceUsesCurrentUserAuthorization(t *testing.T) {
	database := setupMobileAuthDatabase(t)
	user := &model.User{Email: "admin@example.com", Role: constant.UserRoleAdmin, Status: constant.UserStatusNormal}
	if err := database.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	token := "mobile-secret"
	device := &model.MobileDevice{UserID: user.ID, Name: "phone", TokenHash: hashMobileSecret(token), ExpiresAt: time.Now().Add(time.Hour)}
	if err := database.Create(device).Error; err != nil {
		t.Fatal(err)
	}

	_, currentUser, err := AuthenticateMobileDevice(token, "127.0.0.1", "test")
	if err != nil || currentUser.Role != constant.UserRoleAdmin {
		t.Fatalf("active admin authentication failed: user=%#v err=%v", currentUser, err)
	}
	if err := database.Model(user).Updates(map[string]any{"status": constant.UserStatusBlacklist, "role": constant.UserRoleSubAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := AuthenticateMobileDevice(token, "127.0.0.1", "test"); err == nil {
		t.Fatal("disabled or downgraded user must invalidate mobile authorization")
	}
}

func TestNormalizeMobileDeviceTTLDays(t *testing.T) {
	for _, days := range []int{1, 7, 30, 90, 365} {
		actual, err := normalizeMobileDeviceTTLDays(days)
		if err != nil || actual != days {
			t.Fatalf("days %d normalized to %d, err = %v", days, actual, err)
		}
	}
	actual, err := normalizeMobileDeviceTTLDays(0)
	if err != nil || actual != DefaultMobileDeviceTTLDays {
		t.Fatalf("default normalized to %d, err = %v", actual, err)
	}
	if _, err := normalizeMobileDeviceTTLDays(366); err == nil {
		t.Fatal("expected unsupported duration to be rejected")
	}
}

func setupMobileAuthDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	previous := global.DB
	t.Cleanup(func() { global.DB = previous })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mobile-auth.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.User{}, &model.MobileDevice{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	return database
}
