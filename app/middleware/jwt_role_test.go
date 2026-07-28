package middleware

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestIsRoleAllowed(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		required string
		want     bool
	}{
		{name: "super can access admin", actual: constant.UserRoleSuper, required: constant.UserRoleAdmin, want: true},
		{name: "admin can access admin", actual: constant.UserRoleAdmin, required: constant.UserRoleAdmin, want: true},
		{name: "sub admin cannot access admin", actual: constant.UserRoleSubAdmin, required: constant.UserRoleAdmin, want: false},
		{name: "admin can access sub admin", actual: constant.UserRoleAdmin, required: constant.UserRoleSubAdmin, want: true},
		{name: "sub admin can access sub admin", actual: constant.UserRoleSubAdmin, required: constant.UserRoleSubAdmin, want: true},
		{name: "demo cannot access sub admin", actual: constant.UserRoleDemo, required: constant.UserRoleSubAdmin, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRoleAllowed(tt.actual, tt.required); got != tt.want {
				t.Fatalf("isRoleAllowed(%q, %q) = %v, want %v", tt.actual, tt.required, got, tt.want)
			}
		})
	}
}

func TestJwtCheckUsesCurrentUserRoleAndFileBaseDir(t *testing.T) {
	oldDB := global.DB
	oldConf := global.CONF
	t.Cleanup(func() {
		global.DB = oldDB
		global.CONF = oldConf
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "jwt.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	global.DB = db
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	user := &model.User{
		Email:       "sub-admin@example.com",
		Salt:        "salt01",
		Role:        constant.UserRoleSubAdmin,
		Status:      constant.UserStatusNormal,
		FileBaseDir: "/srv/current",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	staleToken, err := token.Create(user.ID, constant.UserRoleAdmin, user.Salt, "/srv/old", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := JwtCheck(staleToken, constant.UserRoleSubAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != constant.UserRoleSubAdmin || claims.FileBaseDir != user.FileBaseDir {
		t.Fatalf("claims were not refreshed from current user: %#v", claims)
	}
	if _, err := JwtCheck(staleToken, constant.UserRoleAdmin); err == nil {
		t.Fatal("stale admin role must not pass an admin-only check")
	}
}
