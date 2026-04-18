package service

import "github.com/aihop/gopanel/app/repo"

var (
	commonRepo = repo.NewCommonRepo()

	appRepo        = repo.NewIAppRepo()
	appDetailRepo  = repo.NewAppDetail()
	imageRepoRepo  = repo.NewIImageRepoRepo()
	composeRepo    = repo.NewIComposeTemplateRepo()
	settingRepo    = repo.NewISettingRepo()
	appInstallRepo = repo.NewIAppInstallRepo()
)
