package repo

import (
	"errors"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type MobileAccessRepo struct{}

func NewMobileAccessRepo() *MobileAccessRepo {
	return &MobileAccessRepo{}
}

func (r *MobileAccessRepo) CreatePairing(pairing *model.MobilePairing) error {
	return global.DB.Create(pairing).Error
}

func (r *MobileAccessRepo) FindPairing(codeHash string) (*model.MobilePairing, error) {
	var pairing model.MobilePairing
	err := global.DB.Where("code_hash = ? AND used_at IS NULL AND expires_at > ?", codeHash, time.Now()).First(&pairing).Error
	return &pairing, err
}

func (r *MobileAccessRepo) ConsumePairing(codeHash string, device *model.MobileDevice) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var pairing model.MobilePairing
		if err := tx.Where("code_hash = ? AND used_at IS NULL AND expires_at > ?", codeHash, time.Now()).First(&pairing).Error; err != nil {
			return err
		}
		now := time.Now()
		result := tx.Model(&model.MobilePairing{}).
			Where("id = ? AND used_at IS NULL", pairing.ID).
			Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("配对码已被使用")
		}
		device.UserID = pairing.UserID
		return tx.Create(device).Error
	})
}

func (r *MobileAccessRepo) FindDevice(tokenHash string) (*model.MobileDevice, error) {
	var device model.MobileDevice
	err := global.DB.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).First(&device).Error
	return &device, err
}

func (r *MobileAccessRepo) TouchDevice(id uint, ip, agent string) error {
	now := time.Now()
	return global.DB.Model(&model.MobileDevice{}).Where("id = ?", id).Updates(map[string]any{
		"last_seen_at": now,
		"last_ip":      ip,
		"last_agent":   agent,
	}).Error
}

func (r *MobileAccessRepo) ListDevices(userID uint) ([]model.MobileDevice, error) {
	var devices []model.MobileDevice
	err := global.DB.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).Order("updated_at desc").Find(&devices).Error
	return devices, err
}

func (r *MobileAccessRepo) RevokeDevice(id, userID uint) error {
	now := time.Now()
	result := global.DB.Model(&model.MobileDevice{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
