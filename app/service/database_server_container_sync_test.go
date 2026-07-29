package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	containertypes "github.com/docker/docker/api/types"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseTypeFromContainerImage(t *testing.T) {
	tests := map[string]model.DatabaseType{
		"mysql:8.4":                  model.DatabaseTypeMysql,
		"mariadb:11":                 model.DatabaseTypeMysql,
		"percona/percona-server:8.0": model.DatabaseTypeMysql,
		"postgres:17":                model.DatabaseTypePostgresql,
		"postgis/postgis:17-3.5":     model.DatabaseTypePostgresql,
	}
	for image, want := range tests {
		got, ok := databaseTypeFromContainerImage(image)
		if !ok || got != want {
			t.Fatalf("databaseTypeFromContainerImage(%q) = %q, %v; want %q", image, got, ok, want)
		}
	}
	if _, ok := databaseTypeFromContainerImage("nginx:latest"); ok {
		t.Fatal("non-database image was detected")
	}
}

func TestBuildContainerDatabaseCandidate(t *testing.T) {
	var inspect containerDatabaseInspect
	data := `[{"Name":"/shop-db","Config":{"Env":["MYSQL_ROOT_PASSWORD=secret"]},"HostConfig":{"NetworkMode":"bridge"},"NetworkSettings":{"Ports":{"3306/tcp":[{"HostIp":"0.0.0.0","HostPort":"33060"}]}}}]`
	var inspections []containerDatabaseInspect
	if err := json.Unmarshal([]byte(data), &inspections); err != nil {
		t.Fatal(err)
	}
	inspect = inspections[0]
	candidate, err := buildContainerDatabaseCandidate(containertypes.Container{}, inspect, model.DatabaseTypeMysql)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Name != "shop-db" || candidate.Host != "127.0.0.1" || candidate.Port != 33060 || candidate.Username != "root" || candidate.Password != "secret" {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
}

func TestBuildContainerDatabaseCandidateRequiresCredentialsAndPublishedPort(t *testing.T) {
	inspect := containerDatabaseInspect{Name: "/db"}
	if _, err := buildContainerDatabaseCandidate(containertypes.Container{}, inspect, model.DatabaseTypeMysql); err == nil {
		t.Fatal("missing credentials should be rejected")
	}
	inspect.Config.Env = []string{"POSTGRES_PASSWORD=secret"}
	if _, err := buildContainerDatabaseCandidate(containertypes.Container{}, inspect, model.DatabaseTypePostgresql); err == nil {
		t.Fatal("unpublished port should be rejected")
	}
}

func TestContainerDatabaseCredentialsFallbacks(t *testing.T) {
	username, password, port := containerDatabaseCredentials(model.DatabaseTypeMysql, map[string]string{"MYSQL_USER": "app", "MYSQL_PASSWORD": "app-secret"})
	if username != "app" || password != "app-secret" || port != 3306 {
		t.Fatalf("unexpected MySQL fallback credentials: %q %q %d", username, password, port)
	}
	username, password, port = containerDatabaseCredentials(model.DatabaseTypePostgresql, map[string]string{"POSTGRESQL_USERNAME": "admin", "POSTGRESQL_PASSWORD": "pg-secret"})
	if username != "admin" || password != "pg-secret" || port != 5432 {
		t.Fatalf("unexpected PostgreSQL fallback credentials: %q %q %d", username, password, port)
	}
}

func TestHasContainerDatabaseEnv(t *testing.T) {
	if !hasContainerDatabaseEnv(model.DatabaseTypeMysql, map[string]string{"MYSQL_ROOT_PASSWORD": "secret"}) {
		t.Fatal("MySQL environment was not detected")
	}
	if !hasContainerDatabaseEnv(model.DatabaseTypePostgresql, map[string]string{"POSTGRESQL_PASSWORD": "secret"}) {
		t.Fatal("PostgreSQL environment was not detected")
	}
	if hasContainerDatabaseEnv(model.DatabaseTypeMysql, map[string]string{"APP_MYSQL_URL": "mysql://example"}) {
		t.Fatal("unrelated application environment was detected as a database container")
	}
}

func TestContainerDatabasePublishedEndpointFallsBackToContainerList(t *testing.T) {
	inspect := containerDatabaseInspect{}
	host, port, ok := containerDatabasePublishedEndpoint(inspect, []containertypes.Port{{PrivatePort: 3306, PublicPort: 33060, Type: "tcp", IP: "::1"}}, 3306)
	if !ok || host != "127.0.0.1" || port != 33060 {
		t.Fatalf("unexpected endpoint: %q %d %v", host, port, ok)
	}
}

func TestContainerDatabaseServerUnchanged(t *testing.T) {
	candidate := containerDatabaseCandidate{Name: "db", Type: model.DatabaseTypeMysql, Host: "127.0.0.1", Port: 3306, Username: "root", Password: "secret"}
	server := model.DatabaseServer{Name: candidate.Name, Type: candidate.Type, Host: candidate.Host, Port: candidate.Port, Username: candidate.Username, Password: candidate.Password, Mode: model.DatabaseModeRemote, Remark: containerDatabaseRemarkPrefix + candidate.Name}
	if !containerDatabaseServerUnchanged(server, candidate) {
		t.Fatal("identical managed server should be skipped")
	}
	server.Port++
	if containerDatabaseServerUnchanged(server, candidate) {
		t.Fatal("changed endpoint should be updated")
	}
}

func TestSyncContainerDatabaseCandidateSkipsDuplicateEndpoint(t *testing.T) {
	oldDB := global.DB
	oldKey := global.CONF.System.EncryptKey
	t.Cleanup(func() {
		global.DB = oldDB
		global.CONF.System.EncryptKey = oldKey
	})
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "database.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	global.DB = database
	if err := database.AutoMigrate(&model.DatabaseServer{}); err != nil {
		t.Fatal(err)
	}
	existing := &model.DatabaseServer{Name: "manual-mariadb", Type: model.DatabaseTypeMariaDB, Host: "localhost", Port: 33060, Username: "root", Password: "manual", Mode: model.DatabaseModeRemote}
	if err := repo.NewDatabaseServer().Create(existing); err != nil {
		t.Fatal(err)
	}

	service := NewDatabaseServer()
	status, err := service.syncContainerDatabaseCandidate(containerDatabaseCandidate{Name: "container-mysql", Type: model.DatabaseTypeMysql, Host: "127.0.0.1", Port: 33060, Username: "root", Password: "container"})
	if err != nil {
		t.Fatal(err)
	}
	if status != "skipped" {
		t.Fatalf("duplicate endpoint status = %q, want skipped", status)
	}
	var count int64
	if err := database.Model(&model.DatabaseServer{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate endpoint created a record; count = %d", count)
	}
}
