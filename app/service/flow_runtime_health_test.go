package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestWaitForFlowWebsiteReadyRequiresConsecutiveSuccesses(t *testing.T) {
	restoreFlowHealthTestHooks(t)
	flowHealthPollInterval = time.Millisecond
	results := []error{nil, errors.New("temporary"), nil, nil}
	calls := 0
	flowHealthCheck = func(context.Context, string, string) error {
		result := results[calls]
		calls++
		return result
	}
	environment := model.FlowEnvironment{HealthCheckSuccessCount: 2, ExternalVerifyTimeoutSeconds: 1}
	if err := waitForFlowWebsiteReady(context.Background(), environment, containerWebsiteTarget{Scheme: "http", Address: "127.0.0.1:3000"}); err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatalf("health checks = %d, want 4", calls)
	}
}

func TestMonitorFlowWebsiteStabilizationUsesFailureAndRecoveryThresholds(t *testing.T) {
	restoreFlowHealthTestHooks(t)
	flowStabilizationPollInterval = time.Millisecond
	flowStabilizationDuration = func(int) time.Duration { return 100 * time.Millisecond }
	results := []error{
		errors.New("first"), nil, errors.New("second"), errors.New("third"),
	}
	calls := 0
	flowHealthCheck = func(context.Context, string, string) error {
		if calls >= len(results) {
			return nil
		}
		result := results[calls]
		calls++
		return result
	}
	environment := model.FlowEnvironment{
		RuntimeMonitorEnabled: true, StabilizationMinutes: 1,
		RuntimeFailureThreshold: 2, RuntimeRecoveryThreshold: 2,
	}
	err := monitorFlowWebsiteStabilization(context.Background(), nil, environment, containerWebsiteTarget{Scheme: "http", Address: "127.0.0.1:3000"})
	if err == nil {
		t.Fatal("expected stabilization failure")
	}
	if calls != 3 {
		t.Fatalf("health checks = %d, want 3", calls)
	}
}

func TestRestoreFlowWebsiteDeploymentRestoresProxyUpstreamAndActiveDeploy(t *testing.T) {
	database := prepareFlowRuntimeHealthTestDB(t)
	website := model.Website{PrimaryDomain: "app.example.com", Type: constant.Proxy, Alias: "app", Status: constant.WebRunning, Protocol: constant.ProtocolHTTP, Proxy: "127.0.0.1:9000", ContainerID: "old"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	oldUpstream := model.WebsiteUpstream{WebsiteID: website.ID, Address: "127.0.0.1:9000", Scheme: "http", Enabled: true}
	if err := database.Create(&oldUpstream).Error; err != nil {
		t.Fatal(err)
	}
	oldDeploy := model.AppDeploy{WebsiteID: website.ID, Version: "old", SourceType: flowDeploymentSourceType, Status: constant.WebRunning, ContainerID: "old", Port: 9000, IsActive: true}
	if err := database.Create(&oldDeploy).Error; err != nil {
		t.Fatal(err)
	}
	snapshot, err := captureFlowWebsiteDeploymentSnapshot(website.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := bindContainerTargetToWebsite(context.Background(), containerWebsiteTarget{
		ContainerID: "new", WebsiteID: website.ID, HostPort: 13000, Scheme: "http",
		Address: "127.0.0.1:13000", DeploymentSourceType: flowDeploymentSourceType,
	}, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := restoreFlowWebsiteDeployment(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	var restored model.Website
	if err := database.First(&restored, website.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restored.ContainerID != "old" || restored.Proxy != "127.0.0.1:9000" {
		t.Fatalf("website not restored: %#v", restored)
	}
	var upstreams []model.WebsiteUpstream
	if err := database.Where("website_id = ?", website.ID).Find(&upstreams).Error; err != nil {
		t.Fatal(err)
	}
	if len(upstreams) != 1 || upstreams[0].Address != oldUpstream.Address {
		t.Fatalf("upstream not restored: %#v", upstreams)
	}
	var active model.AppDeploy
	if err := database.Where("website_id = ? AND is_active = ?", website.ID, true).First(&active).Error; err != nil {
		t.Fatal(err)
	}
	if active.ID != oldDeploy.ID {
		t.Fatalf("active deploy = %d, want %d", active.ID, oldDeploy.ID)
	}
}

func TestDeployFlowRunnerEnvironmentRollsBackAfterStabilizationFailure(t *testing.T) {
	database := prepareFlowRuntimeHealthTestDB(t)
	restoreFlowHealthTestHooks(t)
	website := model.Website{PrimaryDomain: "app.example.com", Type: constant.Proxy, Alias: "app", Status: constant.WebRunning, Protocol: constant.ProtocolHTTP, Proxy: "127.0.0.1:9000", ContainerID: "old"}
	if err := database.Create(&website).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&model.WebsiteUpstream{WebsiteID: website.ID, Address: "127.0.0.1:9000", Scheme: "http", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	oldDeploy := model.AppDeploy{WebsiteID: website.ID, Version: "old", SourceType: flowDeploymentSourceType, Status: constant.WebRunning, ContainerID: "old", Port: 9000, IsActive: true}
	if err := database.Create(&oldDeploy).Error; err != nil {
		t.Fatal(err)
	}
	checks := 0
	flowHealthCheck = func(context.Context, string, string) error {
		checks++
		if checks == 1 {
			return nil
		}
		return errors.New("new website unhealthy")
	}
	flowStabilizationPollInterval = time.Millisecond
	flowStabilizationDuration = func(int) time.Duration { return time.Second }
	flowPreparePreviousContainer = func(context.Context, string, string) (bool, error) { return false, nil }
	flowPrepareTargetContainer = func(context.Context, string, string) error { return nil }
	environment := model.FlowEnvironment{
		WebsiteID: website.ID, HealthCheckSuccessCount: 1, ExternalVerifyTimeoutSeconds: 1,
		RuntimeMonitorEnabled: true, StabilizationMinutes: 1, RuntimeFailureThreshold: 2,
		RuntimeRecoveryThreshold: 1, AutoRollbackDuringStabilization: true,
	}
	record := &model.PipelineRecord{RunnerContainerID: "new", RunnerHostPort: 13000}
	err := deployFlowRunnerEnvironment(context.Background(), environment, record, "1.2.3")
	if err == nil {
		t.Fatal("expected stabilization rollback error")
	}
	var restored model.Website
	if err := database.First(&restored, website.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restored.ContainerID != "old" || restored.Proxy != "127.0.0.1:9000" {
		t.Fatalf("rollback incomplete: website=%#v", restored)
	}
	var active model.AppDeploy
	if err := database.Where("website_id = ? AND is_active = ?", website.ID, true).First(&active).Error; err != nil {
		t.Fatal(err)
	}
	if active.ID != oldDeploy.ID {
		t.Fatalf("active deploy = %d, want %d", active.ID, oldDeploy.ID)
	}
}

func restoreFlowHealthTestHooks(t *testing.T) {
	t.Helper()
	oldCheck := flowHealthCheck
	oldPoll := flowHealthPollInterval
	oldStabilizationPoll := flowStabilizationPollInterval
	oldStabilizationDuration := flowStabilizationDuration
	oldRetentionDuration := flowRetentionDuration
	oldCleanup := flowCleanupContainer
	oldPrepare := flowPreparePreviousContainer
	oldPrepareTarget := flowPrepareTargetContainer
	t.Cleanup(func() {
		flowHealthCheck = oldCheck
		flowHealthPollInterval = oldPoll
		flowStabilizationPollInterval = oldStabilizationPoll
		flowStabilizationDuration = oldStabilizationDuration
		flowRetentionDuration = oldRetentionDuration
		flowCleanupContainer = oldCleanup
		flowPreparePreviousContainer = oldPrepare
		flowPrepareTargetContainer = oldPrepareTarget
	})
}

func prepareFlowRuntimeHealthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := global.DB
	oldApply := applyContainerWebsiteCaddy
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "flow-runtime-health.db")), &gorm.Config{})
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
