package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/compose"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/aihop/gopanel/utils/files"
	"github.com/docker/docker/api/types"
	"gorm.io/gorm"
)

type AppUninstall struct {
	ContainerName string `json:"containerName" validate:"required"`
	DeleteDir     bool   `json:"deleteDir"`
}

type AppInstallService struct {
	tx   *gorm.DB
	repo *repo.AppInstallRepo
	db   *gorm.DB
}

type IAppInstallService interface {
	GetInstallList() ([]dto.AppInstallInfo, error)
	SearchForWebsite(req request.AppInstalledSearch) (int64, []model.AppInstall, error)
	SyncAll() error
	Uninstall(req AppUninstall) error
}

func NewAppInstall() *AppInstallService {
	return &AppInstallService{
		db:   global.DB,
		repo: repo.NewAppInstall(),
	}
}

func (a *AppInstallService) SearchForWebsite(req request.AppInstalledSearch) (int64, []model.AppInstall, error) {
	var (
		opts     []repo.DBOption
		total    int64
		installs []model.AppInstall
		err      error
	)

	if req.Name != "" {
		opts = append(opts, commonRepo.WithLikeName(req.Name))
	}
	// 过滤指定app_id
	if req.Key != "" {
		var app *model.App
		app, err = repo.NewApp(a.db).GetByKey(req.Key)
		if err != nil {
			return 0, nil, err
		}
		fmt.Printf("app: %v\n", app)
		if app.ID == 0 {
			return 0, nil, fmt.Errorf("app key not exist")
		}
		opts = append(opts, appInstallRepo.WithAppId(app.ID))
	}

	// installs, err = appInstallRepo.ListBy(opts...)
	total, installs, err = appInstallRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	return total, installs, nil
}

func (a *AppInstallService) GetInstallList() ([]dto.AppInstallInfo, error) {
	var datas []dto.AppInstallInfo
	appInstalls, err := appInstallRepo.ListBy()
	if err != nil {
		return nil, err
	}
	for _, install := range appInstalls {
		datas = append(datas, dto.AppInstallInfo{
			ID:            install.ID,
			Key:           install.App.Key,
			Name:          install.Name,
			HttpPort:      install.HttpPort,
			HttpsPort:     install.HttpsPort,
			ContainerName: install.ContainerName,
		})
	}
	return datas, nil
}

func (a *AppInstallService) SyncAll() error {
	installs, err := appInstallRepo.ListBy()
	if err != nil {
		return err
	}
	for i := range installs {
		_ = syncAppInstallStatus(&installs[i], true)
	}
	return nil
}

func (a *AppInstallService) Get(ID uint) (*model.AppInstall, error) {
	var appInstall model.AppInstall
	if err := a.db.Where("id = ?", ID).First(&appInstall).Error; err != nil {
		return nil, err
	}
	return &appInstall, nil
}

func (a *AppInstallService) GetByAppId(appId uint) *[]model.AppInstall {
	var appInstalls []model.AppInstall
	// 有可能存在多条记录
	a.db.Where("app_id = ?", appId).Find(&appInstalls)
	return &appInstalls
}

func (a *AppInstallService) GetByName(name string) *model.AppInstall {
	var appInstalls model.AppInstall
	a.db.Where("name = ?", name).First(&appInstalls)
	return &appInstalls
}

func (a *AppInstallService) GetByContainerName(containerName string) (*model.AppInstall, error) {
	var appInstalls model.AppInstall
	if err := a.db.Where("container_name = ?", containerName).First(&appInstalls).Error; err != nil {
		return nil, err
	}
	return &appInstalls, nil
}

func (a *AppInstallService) Create(appInstall *model.AppInstall) error {
	if err := a.db.Create(&appInstall).Error; err != nil {
		return err
	}
	return nil
}

func (s *AppInstallService) Update(appInstall *model.AppInstall) error {
	if err := s.db.Model(appInstall).
		Where("id = ?", appInstall.ID).
		Updates(appInstall).Error; err != nil {
		return err
	}
	return nil
}

func (s *AppInstallService) Delete(id uint) error {
	if err := s.db.Where("id = ?", id).
		Delete(&model.AppInstall{}).Error; err != nil {
		return err
	}
	return nil
}

func (s *AppInstallService) CreateOrUpdate(appInstall *model.AppInstall) error {
	if appInstall.ID != 0 {
		return s.Update(appInstall)
	}
	app := s.GetByName(appInstall.Name)
	// fmt.Printf("app: %v\n", app)
	var err error
	if app != nil && app.ID != 0 {
		appInstall.ID = (*app).ID
		err = s.Update(appInstall)
	} else {
		err = s.Create(appInstall)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *AppInstallService) Uninstall(req AppUninstall) error {
	appInstall, err := s.GetByContainerName(req.ContainerName)
	if err != nil {
		return err
	}
	if appInstall.ID == 0 {
		return fmt.Errorf("app install not exist")
	}

	var path string = ""
	// 找到这个 compose yml文件的路径
	var compose model.Compose
	compose, err = composeRepo.GetRecord(commonRepo.WithByName(appInstall.Name))
	if err != nil {
		return err
	}
	path = compose.Path

	composeOperation := dto.ComposeOperation{
		Name:      appInstall.Name,
		Path:      path,
		Operation: "delete",
		WithFile:  req.DeleteDir,
	}

	if err = NewIContainerService().ComposeOperation(&composeOperation); err != nil {
		return err
	}

	// 移除数据库记录
	err = s.Delete(appInstall.ID)
	if err != nil {
		return err
	}

	return nil
}

func (a *AppInstallService) Operate(req request.AppInstalledOperate) error {
	install, err := appInstallRepo.GetFirstByCtx(context.Background(), commonRepo.WithByID(req.InstallId))
	if err != nil {
		return err
	}
	if !req.ForceDelete && !files.NewFileOp().Stat(install.GetPath()) {
		return errors.New(constant.ErrInstallDirNotFound)
	}
	dockerComposePath := install.GetComposePath()
	switch req.Operate {
	case constant.Rebuild:
		return rebuildApp(install)
	case constant.Start:
		out, err := compose.Start(dockerComposePath)
		if err != nil {
			return handleErr(install, err, out)
		}
		return syncAppInstallStatus(&install, false)
	case constant.Stop:
		out, err := compose.Stop(dockerComposePath)
		if err != nil {
			return handleErr(install, err, out)
		}
		return syncAppInstallStatus(&install, false)
	case constant.Restart:
		out, err := compose.Restart(dockerComposePath)
		if err != nil {
			return handleErr(install, err, out)
		}
		return syncAppInstallStatus(&install, false)
	case constant.Delete:
		if err := a.Uninstall(AppUninstall{
			ContainerName: install.ContainerName,
			DeleteDir:     req.ForceDelete,
		}); err != nil && !req.ForceDelete {
			return err
		}
		return nil
	case constant.Sync:
		return syncAppInstallStatus(&install, true)
	default:
		return errors.New("operate not support")
	}
}

func syncAppInstallStatus(appInstall *model.AppInstall, force bool) error {
	if appInstall.Status == constant.Installing || appInstall.Status == constant.Rebuilding || appInstall.Status == constant.Upgrading {
		return nil
	}
	cli, err := docker.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	var (
		containers     []types.Container
		containersMap  map[string]types.Container
		containerNames = strings.Split(appInstall.ContainerName, ",")
	)
	containers, err = cli.ListContainersByName(containerNames)
	if err != nil {
		return err
	}
	containersMap = make(map[string]types.Container)
	for _, con := range containers {
		containersMap[con.Names[0]] = con
	}
	synAppInstall(containersMap, appInstall, force)
	return nil
}
