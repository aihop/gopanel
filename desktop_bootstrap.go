//go:build desktop

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const desktopAdminEmail = "admin@gopanel.local"

type desktopCredentials struct {
	Email    string
	Password string
}

func prepareDesktopCredentials(baseDir string) (*desktopCredentials, error) {
	databasePath := filepath.Join(baseDir, "db", "gopanel.db")
	if _, err := os.Stat(databasePath); err == nil {
		database, openErr := gorm.Open(sqlite.Open(databasePath), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if openErr != nil {
			return nil, fmt.Errorf("inspect desktop database: %w", openErr)
		}
		sqlDatabase, dbErr := database.DB()
		if dbErr != nil {
			return nil, fmt.Errorf("inspect desktop database connection: %w", dbErr)
		}
		hasUserTable := database.Migrator().HasTable(&model.User{})
		if closeErr := sqlDatabase.Close(); closeErr != nil {
			return nil, fmt.Errorf("close desktop database inspection: %w", closeErr)
		}
		if hasUserTable {
			return nil, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect desktop database: %w", err)
	}

	passwordBytes := make([]byte, 18)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("create initial desktop password: %w", err)
	}
	return &desktopCredentials{
		Email:    desktopAdminEmail,
		Password: base64.RawURLEncoding.EncodeToString(passwordBytes),
	}, nil
}

func (a *desktopApp) showInitialCredentials(ctx context.Context) {
	credentials := a.initialCredentials
	if credentials == nil {
		return
	}
	message := fmt.Sprintf("GoPanel 已完成首次初始化。\n\n账号：%s\n临时密码：%s\n\n登录后请立即修改密码。", credentials.Email, credentials.Password)
	selected, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.InfoDialog,
		Title:         "GoPanel 首次启动",
		Message:       message,
		Buttons:       []string{"复制密码", "关闭"},
		DefaultButton: "复制密码",
		CancelButton:  "关闭",
	})
	if err == nil && selected == "复制密码" {
		_ = runtime.ClipboardSetText(ctx, credentials.Password)
	}
}
