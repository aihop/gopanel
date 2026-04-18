package api

import (
	"github.com/aihop/gopanel/app/service"
)

type ApiGroup struct {
}

var ApiGroupApp = new(ApiGroup)

var (
	dashboardService = service.NewIDashboardService()
	imageService     = service.NewIImageService()
	dockerService    = service.NewIDockerService()
	imageRepoService = service.NewIImageRepoService()
	containerService = service.NewIContainerService()
	fileService      = service.NewIFileService()
	appService       = service.NewIAppService()
	processService   = service.NewIProcessService()

	appVersionService = service.NewIAppVersionService()
)
