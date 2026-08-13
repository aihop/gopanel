package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	containertypes "github.com/docker/docker/api/types"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPublishedContainerAddress(t *testing.T) {
	ports := []containertypes.Port{
		{IP: "0.0.0.0", PrivatePort: 3000, PublicPort: 13000, Type: "tcp"},
		{IP: "0.0.0.0", PrivatePort: 5353, PublicPort: 15353, Type: "udp"},
	}
	address, err := publishedContainerAddress(ports, 13000)
	if err != nil {
		t.Fatal(err)
	}
	if address != "127.0.0.1:13000" {
		t.Fatalf("unexpected address: %q", address)
	}
	if _, err := publishedContainerAddress(ports, 15353); err == nil {
		t.Fatal("UDP port should not be accepted")
	}
}

func TestBindContainerTargetToWebsite(t *testing.T) {
	database := prepareContainerWebsiteTestDB(t)
	website := model.Website{PrimaryDomain: "app.example.com", Type: constant.Proxy, Alias: "app", Status: constant.WebRunning, Protocol: constant.ProtocolHTTP, Proxy: "127.0.0.1:9000"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.WebsiteUpstream{WebsiteID: website.ID, Address: "127.0.0.1:9000", Scheme: "http", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.AppDeploy{WebsiteID: website.ID, Version: "old", SourceType: "container_bind", Status: constant.WebRunning, ContainerID: "old", Port: 9000, IsActive: true}).Error; err != nil {
		t.Fatal(err)
	}

	if err := bindContainerTargetToWebsite(context.Background(), containerWebsiteTarget{
		ContainerID: "new-container",
		RuntimeHost: "unix:///run/user/1000/podman/podman.sock",
		WebsiteID:   website.ID,
		HostPort:    13000,
		Scheme:      "http",
		Address:     "127.0.0.1:13000",
	}, ""); err != nil {
		t.Fatal(err)
	}

	var updated model.Website
	if err := database.First(&updated, website.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.ContainerID != "new-container" || updated.Proxy != "127.0.0.1:13000" || updated.CodeSource != "container" {
		t.Fatalf("unexpected website binding: %#v", updated)
	}
	var upstreams []model.WebsiteUpstream
	if err := database.Where("website_id = ?", website.ID).Find(&upstreams).Error; err != nil {
		t.Fatal(err)
	}
	if len(upstreams) != 1 || upstreams[0].Address != "127.0.0.1:13000" {
		t.Fatalf("unexpected upstreams: %#v", upstreams)
	}
	var active []model.AppDeploy
	if err := database.Where("website_id = ? AND is_active = ?", website.ID, true).Find(&active).Error; err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ContainerID != "new-container" || active[0].RuntimeHost == "" {
		t.Fatalf("unexpected active deploy: %#v", active)
	}
}

func TestBindContainerTargetRollsBackWhenCaddyFails(t *testing.T) {
	database := prepareContainerWebsiteTestDB(t)
	website := model.Website{PrimaryDomain: "app.example.com", Type: constant.Proxy, Alias: "app", Status: constant.WebRunning, Protocol: constant.ProtocolHTTP, Proxy: "127.0.0.1:9000"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	applyContainerWebsiteCaddy = func(context.Context) error { return errors.New("apply failed") }

	err := bindContainerTargetToWebsite(context.Background(), containerWebsiteTarget{
		ContainerID: "new-container",
		WebsiteID:   website.ID,
		HostPort:    13000,
		Scheme:      "http",
		Address:     "127.0.0.1:13000",
	}, "")
	if err == nil {
		t.Fatal("expected Caddy failure")
	}
	var updated model.Website
	if err := database.First(&updated, website.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.ContainerID != "" || updated.Proxy != "127.0.0.1:9000" {
		t.Fatalf("website was not rolled back: %#v", updated)
	}
	var count int64
	if err := database.Model(&model.AppDeploy{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deploy audit was not rolled back: %d", count)
	}
}

func TestBindContainerTargetRejectsNonProxyWebsite(t *testing.T) {
	database := prepareContainerWebsiteTestDB(t)
	website := model.Website{PrimaryDomain: "static.example.com", Type: constant.Static, Alias: "static", Status: constant.WebRunning, Protocol: constant.ProtocolHTTP}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	err := bindContainerTargetToWebsite(context.Background(), containerWebsiteTarget{WebsiteID: website.ID}, "")
	if err == nil {
		t.Fatal("expected non-proxy website rejection")
	}
}

func prepareContainerWebsiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := global.DB
	oldApply := applyContainerWebsiteCaddy
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "container-website.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Website{}, &model.WebsiteDomain{}, &model.WebsiteUpstream{}, &model.AppDeploy{}); err != nil {
		t.Fatal(err)
	}
	global.DB = database
	applyContainerWebsiteCaddy = func(context.Context) error { return nil }
	t.Cleanup(func() {
		global.DB = oldDB
		applyContainerWebsiteCaddy = oldApply
	})
	return database
}
