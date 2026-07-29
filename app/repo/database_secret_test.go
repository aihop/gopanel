package repo

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseSecretRoundTrip(t *testing.T) {
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })

	ciphertext, err := encryptDatabaseSecret("database-password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ciphertext, databaseSecretPrefix) || strings.Contains(ciphertext, "database-password") {
		t.Fatalf("credential was not encrypted: %q", ciphertext)
	}
	plaintext, err := decryptDatabaseSecret(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "database-password" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestDatabaseServerRepositoryEncryptsAndRedactsPassword(t *testing.T) {
	oldDB := global.DB
	oldKey := global.CONF.System.EncryptKey
	t.Cleanup(func() {
		global.DB = oldDB
		global.CONF.System.EncryptKey = oldKey
	})
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "database.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = db
	if err := db.AutoMigrate(&model.DatabaseServer{}); err != nil {
		t.Fatal(err)
	}
	server := &model.DatabaseServer{Name: "test", Password: "database-password"}
	if err := NewDatabaseServer().Create(server); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.Model(&model.DatabaseServer{}).Where("id = ?", server.ID).Pluck("password", &stored).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, databaseSecretPrefix) || strings.Contains(stored, server.Password) {
		t.Fatalf("password was stored in plaintext: %q", stored)
	}
	loaded, err := NewDatabaseServer().Get(server.ID)
	if err != nil || loaded.Password != server.Password {
		t.Fatalf("repository did not decrypt password: %#v, %v", loaded, err)
	}
	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "password") || strings.Contains(string(encoded), server.Password) {
		t.Fatalf("password leaked through JSON: %s", encoded)
	}
}

func TestLegacyDatabaseSecretRemainsReadable(t *testing.T) {
	plaintext, err := decryptDatabaseSecret("legacy-password")
	if err != nil || plaintext != "legacy-password" {
		t.Fatalf("legacy credential should remain readable: %q, %v", plaintext, err)
	}
}

func TestDecryptDatabaseServerPasswords(t *testing.T) {
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })

	ciphertext, err := encryptDatabaseSecret("database-password")
	if err != nil {
		t.Fatal(err)
	}
	servers := []*model.DatabaseServer{
		{Password: ciphertext},
		{Password: "legacy-password"},
	}
	if err := decryptDatabaseServerPasswords(servers); err != nil {
		t.Fatal(err)
	}
	if servers[0].Password != "database-password" || servers[1].Password != "legacy-password" {
		t.Fatalf("unexpected decrypted passwords: %#v", servers)
	}
}
