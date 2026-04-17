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

	// 流水线
	if err := repo.NewPipeline(global.DB).MigrateTable(); err != nil {
		sysLog.Println("Pipeline table error", err)
		return
	}

	if err := repo.NewPipelineRecord(global.DB).MigrateTable(); err != nil {
		sysLog.Println("PipelineRecord table error", err)
		return
	}

	if err := global.DB.AutoMigrate(&model.Firewall{}, &model.Forward{}, &model.AITask{}, &model.AIMessage{}, &model.OperationLog{}, &model.LoginLog{}, &model.WebsiteDeploy{}); err != nil {
		sysLog.Println("AutoMigrate additional tables error", err)
		return
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
