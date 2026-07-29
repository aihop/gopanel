package repo

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMobilePairingCanOnlyBeConsumedOnce(t *testing.T) {
	repository, database := setupMobileAccessRepo(t)
	pairing := &model.MobilePairing{UserID: 7, CodeHash: "pair-hash", ExpiresAt: time.Now().Add(time.Minute)}
	if err := repository.CreatePairing(pairing); err != nil {
		t.Fatal(err)
	}
	device := &model.MobileDevice{Name: "phone", TokenHash: "token-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repository.ConsumePairing("pair-hash", device); err != nil {
		t.Fatal(err)
	}
	if device.UserID != 7 {
		t.Fatalf("device user ID = %d", device.UserID)
	}
	if err := repository.ConsumePairing("pair-hash", &model.MobileDevice{Name: "second", TokenHash: "second-token", ExpiresAt: time.Now().Add(time.Hour)}); err == nil {
		t.Fatal("expected reused pairing to be rejected")
	}
	var count int64
	if err := database.Model(&model.MobileDevice{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("device count = %d, err = %v", count, err)
	}
}

func TestMobileDeviceCanBeRevoked(t *testing.T) {
	repository, database := setupMobileAccessRepo(t)
	device := &model.MobileDevice{UserID: 9, Name: "phone", TokenHash: "stored-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := database.Create(device).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindDevice("stored-hash"); err != nil {
		t.Fatal(err)
	}
	if err := repository.RevokeDevice(device.ID, device.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindDevice("stored-hash"); err == nil {
		t.Fatal("expected revoked device to be rejected")
	}
}

func TestExpiredMobilePairingIsRejected(t *testing.T) {
	repository, _ := setupMobileAccessRepo(t)
	pairing := &model.MobilePairing{UserID: 7, CodeHash: "expired", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := repository.CreatePairing(pairing); err != nil {
		t.Fatal(err)
	}
	device := &model.MobileDevice{Name: "phone", TokenHash: "token-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repository.ConsumePairing("expired", device); err == nil {
		t.Fatal("expected expired pairing to be rejected")
	}
}

func setupMobileAccessRepo(t *testing.T) (*MobileAccessRepo, *gorm.DB) {
	t.Helper()
	oldDatabase := global.DB
	t.Cleanup(func() { global.DB = oldDatabase })
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mobile-access.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.MobilePairing{}, &model.MobileDevice{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	return NewMobileAccessRepo(), database
}
