package repo

import (
	sysLog "log"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"

	"github.com/aihop/gopanel/global"
)

func Init() {

	if err := repo.NewUser(global.DB).MigrateTable(); err != nil {
		sysLog.Println("AutoMigrate table error", err)
		return
	}
	if err := repo.NewUserNote(global.DB).MigrateTable(); err != nil {
		sysLog.Println("UserNote table error", err)
		return
	}
	if err := repo.NewImageRepo(global.DB).MigrateTable(); err != nil {
		sysLog.Println("ImageRepo table error", err)
		return
	}
	if err := repo.NewSetting(global.DB).MigrateTable(); err != nil {
		sysLog.Println("ImageRepo table error", err)
		return
	}
	if err := repo.NewComposeTemplate(global.DB).MigrateTable(); err != nil {
		sysLog.Println("ComposeTemplate table error", err)
		return
	}
	if err := repo.NewCompose(global.DB).MigrateTableCompose(); err != nil {
		sysLog.Println("Compose table error", err)
		return
	}
	if err := repo.NewApp(global.DB).MigrateTable(); err != nil {
		sysLog.Println("App table error", err)
		return
	}
	if err := repo.NewAppInstall().MigrateTable(); err != nil {
		sysLog.Println("AppInstall table error", err)
		return
	}
	if err := repo.NewAppDetail().MigrateTable(); err != nil {
		sysLog.Println("AppInstall table error", err)
		return
	}

	if err := repo.NewDatabaseUser().MigrateTable(); err != nil {
		sysLog.Println("DatabaseUser table error", err)
		return
	}

	if err := repo.NewDatabaseServer().MigrateTable(); err != nil {
		sysLog.Println("DatabaseServer table error", err)
		return
	}

	if err := repo.NewBackupRecord().MigrateTable(); err != nil {
		sysLog.Println("BackupRecord table error", err)
		return
	}

	if err := repo.NewWebsite().MigrateTable(); err != nil {
		sysLog.Println("NewWebsite table error", err)
		return
	}

	if err := repo.NewSSL().MigrateTable(); err != nil {
		sysLog.Println("SSL table error", err)
		return
	}

	if err := repo.NewCloudAccount().MigrateTable(); err != nil {
		sysLog.Println("CloudAccount table error", err)
		return
	}

	if err := repo.NewAcmeAccount().MigrateTable(); err != nil {
		sysLog.Println("AcmeAccount table error", err)
		return
	}

	// 邮件通知与告警事件
	if err := repo.NewNotify().MigrateTable(); err != nil {
		sysLog.Println("Notify table error", err)
		return
	}

	// 流水线
	if err := repo.NewPipeline(global.DB).MigrateTable(); err != nil {
		sysLog.Println("Pipeline table error", err)
		return
	}

	if err := repo.NewPipelineRecord(global.DB).MigrateTable(); err != nil {
		sysLog.Println("PipelineRecord table error", err)
		return
	}

	if err := repo.NewRelease(global.DB).MigrateTable(); err != nil {
		sysLog.Println("Release table error", err)
		return
	}

	if err := repo.NewAppDeploy(global.DB).MigrateTable(); err != nil {
		sysLog.Println("AppDeploy table error", err)
		return
	}

	if err := repo.NewCronjob().MigrateTable(); err != nil {
		sysLog.Println("Cronjob table error", err)
		return
	}

	if err := repo.NewNode().MigrateTable(); err != nil {
		sysLog.Println("Node table error", err)
		return
	}

	if err := global.DB.AutoMigrate(
		&model.Firewall{},
		&model.Forward{},
		&model.AIGroup{},
		&model.AITask{},
		&model.AIMessage{},
		&model.AIDevSession{},
		&model.AIDevSessionRepository{},
		&model.AIExecutionRun{},
		&model.AICodeDatabaseAccess{},
		&model.AICodeDelivery{},
		&model.AICodeAuditEvent{},
		&model.AIInstruction{},
		&model.AIPreview{},
		&model.AITimelineEvent{},
		&model.AIApproval{},
		&model.MobilePairing{},
		&model.MobileDevice{},
		&model.OperationLog{},
		&model.LoginLog{},
		&model.LegacyWebsiteDeploy{},
	); err != nil {
		sysLog.Println("AutoMigrate additional tables error", err)
		return
	}

	if err := repo.NewAppDeploy(global.DB).SyncFromLegacy(); err != nil {
		sysLog.Println("AppDeploy sync legacy data error", err)
		return
	}

	if err := repo.NewRelease(global.DB).FixSharedReleaseDirs(); err != nil {
		sysLog.Println("Release shared dir repair warning", err)
	}

	if err := repo.NewRelease(global.DB).EnsureUniquePipelineRecordIndex(); err != nil {
		sysLog.Println("Release unique index repair warning", err)
	}

	if global.MonitorDB != nil {
		global.MonitorDB.AutoMigrate(
			&model.MonitorBase{},
			&model.MonitorIO{},
			&model.MonitorNetwork{},
		)
	}

	sysLog.Println("AutoMigrate table success")
}
