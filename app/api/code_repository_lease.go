package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errCodeRepositoryBusy = errors.New("repository operation busy")

func newCodeRepositoryLeaseOwner(purpose string) string {
	random := make([]byte, 8)
	_, _ = rand.Read(random)
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%s-%d-%s", purpose, hostname, os.Getpid(), hex.EncodeToString(random))
}

func codeDeliveryRepositoryKey(sourceDir, remoteName, targetBranch string) string {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(sourceDir))
	if err != nil {
		resolved, _ = filepath.Abs(filepath.Clean(sourceDir))
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{resolved, remoteName, targetBranch}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func normalizeCodeRepositoryLeaseKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	return normalized
}

func acquireCodeRepositoryLeases(owner string, jobID uint, keys []string) (bool, error) {
	if global.DB == nil {
		return false, errors.New("仓库协调器不可用")
	}
	keys = normalizeCodeRepositoryLeaseKeys(keys)
	if strings.TrimSpace(owner) == "" || len(keys) == 0 {
		return false, errors.New("仓库租约参数无效")
	}
	now, expiresAt := time.Now(), time.Now().Add(codeDeliveryLeaseDuration)
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, key := range keys {
			result := tx.Model(&model.AICodeDeliveryLease{}).Where(
				"repository_key = ? AND ((lease_owner = ? AND job_id = ?) OR lease_expires_at IS NULL OR lease_expires_at < ?)", key, owner, jobID, now,
			).Updates(map[string]any{"job_id": jobID, "lease_owner": owner, "lease_expires_at": expiresAt})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				continue
			}
			lease := model.AICodeDeliveryLease{RepositoryKey: key, JobID: jobID, LeaseOwner: owner, LeaseExpiresAt: &expiresAt}
			created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&lease)
			if created.Error != nil {
				return created.Error
			}
			if created.RowsAffected == 0 {
				return errCodeRepositoryBusy
			}
		}
		return nil
	})
	if errors.Is(err, errCodeRepositoryBusy) {
		return false, nil
	}
	return err == nil, err
}

func releaseCodeRepositoryLeases(owner string, keys []string) error {
	if global.DB == nil {
		return nil
	}
	keys = normalizeCodeRepositoryLeaseKeys(keys)
	if len(keys) == 0 {
		return nil
	}
	return global.DB.Where("lease_owner = ? AND repository_key IN ?", owner, keys).
		Delete(&model.AICodeDeliveryLease{}).Error
}

func heartbeatCodeRepositoryLeases(ctx context.Context, owner string, keys []string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			expiresAt := time.Now().Add(codeDeliveryLeaseDuration)
			_ = global.DB.Model(&model.AICodeDeliveryLease{}).
				Where("lease_owner = ? AND repository_key IN ?", owner, keys).
				Update("lease_expires_at", expiresAt).Error
		case <-ctx.Done():
			return
		}
	}
}
