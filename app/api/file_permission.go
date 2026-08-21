package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func requireFileAccess(c fiber.Ctx, paths ...string) error {
	claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil {
		return buserr.New(constant.ErrFileUnauthorized)
	}
	if claims.Role != constant.UserRoleSubAdmin {
		return nil
	}
	for _, path := range paths {
		if err := service.ValidatePathWithinBase(claims.FileBaseDir, path); err != nil {
			return err
		}
	}
	return nil
}

func newFileTaskKey(c fiber.Ctx, prefix string) (string, error) {
	claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil {
		return "", buserr.New(constant.ErrFileUnauthorized)
	}
	return fmt.Sprintf("%s%d_%s", prefix, claims.UserId, common.RandStrAndNum(20)), nil
}

func requireFileTaskAccess(c fiber.Ctx, key, prefix string) error {
	claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil {
		return buserr.New(constant.ErrFileUnauthorized)
	}
	if !validFileTaskKey(key, prefix) {
		return buserr.New(constant.ErrFileInvalidTaskKey)
	}
	if claims.Role == constant.UserRoleSubAdmin {
		ownerPrefix := prefix + strconv.FormatUint(uint64(claims.UserId), 10) + "_"
		if !strings.HasPrefix(key, ownerPrefix) {
			return buserr.New(constant.ErrFilePermissionDenied)
		}
	}
	return nil
}

func validFileTaskKey(key, prefix string) bool {
	if !strings.HasPrefix(key, prefix) || len(key) <= len(prefix) {
		return false
	}
	for _, char := range key {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func fileBaseDir(c fiber.Ctx) (string, bool) {
	claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil || claims.Role != constant.UserRoleSubAdmin {
		return "", false
	}
	baseDir := strings.TrimSpace(claims.FileBaseDir)
	if baseDir == "" || !filepath.IsAbs(baseDir) {
		return "", true
	}
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", true
	}
	return absBase, true
}
