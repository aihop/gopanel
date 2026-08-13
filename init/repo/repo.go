package repo

import (
	"fmt"
	sysLog "log"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"

	"github.com/aihop/gopanel/global"
)

func Init() error {
	migrations := []struct {
		name string
		run  func() error
	}{
		{"user", func() error { return repo.NewUser(global.DB).MigrateTable() }},
		{"user note", func() error { return repo.NewUserNote(global.DB).MigrateTable() }},
		{"image", func() error { return repo.NewImageRepo(global.DB).MigrateTable() }},
		{"setting", func() error { return repo.NewSetting(global.DB).MigrateTable() }},
		{"compose template", func() error { return repo.NewComposeTemplate(global.DB).MigrateTable() }},
		{"compose", func() error { return repo.NewCompose(global.DB).MigrateTableCompose() }},
		{"app", func() error { return repo.NewApp(global.DB).MigrateTable() }},
		{"app install", func() error { return repo.NewAppInstall().MigrateTable() }},
		{"app detail", func() error { return repo.NewAppDetail().MigrateTable() }},
		{"database user", func() error { return repo.NewDatabaseUser().MigrateTable() }},
		{"database server", func() error { return repo.NewDatabaseServer().MigrateTable() }},
		{"backup record", func() error { return repo.NewBackupRecord().MigrateTable() }},
		{"website", func() error { return repo.NewWebsite().MigrateTable() }},
		{"SSL", func() error { return repo.NewSSL().MigrateTable() }},
		{"cloud account", func() error { return repo.NewCloudAccount().MigrateTable() }},
		{"ACME account", func() error { return repo.NewAcmeAccount().MigrateTable() }},
		{"notify", func() error { return repo.NewNotify().MigrateTable() }},
		{"security monitoring", func() error { return repo.NewSecurityMonitoring().MigrateTable() }},
		{"pipeline", func() error { return repo.NewPipeline(global.DB).MigrateTable() }},
		{"pipeline record", func() error { return repo.NewPipelineRecord(global.DB).MigrateTable() }},
		{"release", func() error { return repo.NewRelease(global.DB).MigrateTable() }},
		{"flow", func() error { return repo.NewFlow(global.DB).MigrateTable() }},
		{"app deploy", func() error { return repo.NewAppDeploy(global.DB).MigrateTable() }},
		{"cronjob", func() error { return repo.NewCronjob().MigrateTable() }},
		{"node", func() error { return repo.NewNode().MigrateTable() }},
		{"legacy AI project data", func() error { return repo.MigrateLegacyAIProjects(global.DB) }},
	}
	for _, migration := range migrations {
		if err := migration.run(); err != nil {
			return fmt.Errorf("migrate %s: %w", migration.name, err)
		}
	}

	if err := global.DB.AutoMigrate(
		&model.Firewall{},
		&model.Forward{},
		&model.AIGitCredential{},
		&model.AIProject{},
		&model.AITask{},
		&model.AIMessage{},
		&model.AIDevSession{},
		&model.AIDevSessionRepository{},
		&model.AIExecutionRun{},
		&model.AICodeDatabaseAccess{},
		&model.AICodeDelivery{},
		&model.AICodeDeliveryJob{},
		&model.AICodeDeliveryLease{},
		&model.AICodeDeliveryAttempt{},
		&model.AICodeMemoryEntry{},
		&model.AICodeMemorySummary{},
		&model.AICodeMemorySetting{},
		&model.AIProviderAccount{},
		&model.AICodeMemoryExtractionState{},
		&model.AICodeMemoryAuditEvent{},
		&model.AICodeAuditEvent{},
		&model.AIInstruction{},
		&model.AIPreview{},
		&model.AITimelineEvent{},
		&model.AIApproval{},
		&model.MobilePairing{},
		&model.MobileDevice{},
		&model.OperationLog{},
		&model.LoginLog{},
		&model.HostTerminalSession{},
		&model.HostTerminalAuditEvent{},
	); err != nil {
		return fmt.Errorf("migrate additional tables: %w", err)
	}

	if err := repo.NewAppDeploy(global.DB).SyncFromLegacy(); err != nil {
		return fmt.Errorf("sync legacy app deploy data: %w", err)
	}

	if err := repo.NewRelease(global.DB).FixSharedReleaseDirs(); err != nil {
		sysLog.Println("Release shared dir repair warning", err)
	}

	if err := repo.NewRelease(global.DB).EnsureUniquePipelineRecordIndex(); err != nil {
		sysLog.Println("Release unique index repair warning", err)
	}

	if global.MonitorDB != nil {
		if err := global.MonitorDB.AutoMigrate(
			&model.MonitorBase{},
			&model.MonitorIO{},
			&model.MonitorNetwork{},
		); err != nil {
			return fmt.Errorf("migrate monitor tables: %w", err)
		}
	}

	sysLog.Println("AutoMigrate table success")
	return nil
}
