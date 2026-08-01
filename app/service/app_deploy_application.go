package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"gorm.io/gorm"
)

type AppDeployApplicationService struct {
	db          *gorm.DB
	releaseRepo *repo.ReleaseRepo
}

type AppDeployTriggerOptions struct {
	WebsiteID uint
	ZipPath   string
	ImageTag  string
	ReleaseID uint
}

func NewAppDeployApplication(db *gorm.DB) *AppDeployApplicationService {
	return &AppDeployApplicationService{
		db:          db,
		releaseRepo: repo.NewRelease(db),
	}
}

func (s *AppDeployApplicationService) PageWebsiteReleases(websiteID uint, page, limit int) (int64, []model.Release, error) {
	var website model.Website
	if err := s.db.Select("id", "pipeline_id").First(&website, websiteID).Error; err != nil {
		return 0, nil, fmt.Errorf("网站不存在")
	}
	if website.PipelineID == 0 {
		return 0, []model.Release{}, nil
	}
	return s.releaseRepo.PageByPipeline(website.PipelineID, page, limit)
}

func (s *AppDeployApplicationService) ListByWebsite(websiteID uint) ([]model.AppDeploy, error) {
	var list []model.AppDeploy
	if err := s.db.Where("website_id = ?", websiteID).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *AppDeployApplicationService) Switch(deployID uint, exposePort int) error {
	var targetDeploy model.AppDeploy
	if err := s.db.First(&targetDeploy, deployID).Error; err != nil {
		return fmt.Errorf("部署记录不存在")
	}
	if targetDeploy.SourceType == "container_bind" {
		return fmt.Errorf("容器端口绑定记录不能作为部署版本切换，请从容器列表重新发布")
	}
	var website model.Website
	if err := s.db.Preload("Domains").First(&website, targetDeploy.WebsiteID).Error; err != nil {
		return fmt.Errorf("网站不存在")
	}

	if targetDeploy.SourceType == "pipeline" || targetDeploy.SourceType == "git" || targetDeploy.ArchiveFile != "" || targetDeploy.ImageTag != "" {
		releaseDir := targetDeploy.ReleaseDir
		if releaseDir == "" {
			releaseDir = filepath.Join(global.CONF.System.BaseDir, "www", website.Alias, "releases", targetDeploy.Version)
		}
		archiveFile := targetDeploy.ArchiveFile
		if archiveFile == "" {
			archiveFile = targetDeploy.SourceUrl
		}
		targetDeploy.ArchiveFile = archiveFile
		targetDeploy.SourceUrl = archiveFile
		targetDeploy.ReleaseDir = releaseDir
		if _, err := ReuseAppDeployment(website, &targetDeploy, exposePort); err != nil {
			return fmt.Errorf("回滚发布失败: %w", err)
		}
		return nil
	}

	if website.Type == constant.Proxy {
		var appInstall model.AppInstall
		if err := s.db.First(&appInstall, website.AppInstallID).Error; err == nil {
			envPath := appInstall.GetEnvPath()
			composePath := appInstall.GetComposePath()
			_ = os.WriteFile(envPath, []byte(targetDeploy.Env), 0644)
			_ = os.WriteFile(composePath, []byte(targetDeploy.DockerCompose), 0644)

			_ = NewAppInstall().Operate(request.AppInstalledOperate{
				InstallId: appInstall.ID,
				Operate:   constant.OperateUp,
			})

			appInstall.Env = targetDeploy.Env
			appInstall.DockerCompose = targetDeploy.DockerCompose
			s.db.Save(&appInstall)

			s.db.Model(&model.AppDeploy{}).Where("website_id = ?", website.ID).Update("is_active", false)
			targetDeploy.IsActive = true
			s.db.Save(&targetDeploy)
			return nil
		}
	}

	return fmt.Errorf("当前版本缺少可回滚快照")
}

func (s *AppDeployApplicationService) Delete(deployID uint) error {
	var deploy model.AppDeploy
	if err := s.db.First(&deploy, deployID).Error; err != nil {
		return fmt.Errorf("部署记录不存在")
	}
	if deploy.IsActive {
		return fmt.Errorf("线上运行中的版本不允许删除")
	}
	if err := s.db.Delete(&deploy).Error; err != nil {
		return fmt.Errorf("删除部署记录失败: %w", err)
	}
	return nil
}

func (s *AppDeployApplicationService) Trigger(opts AppDeployTriggerOptions, exposePort int) error {
	var website model.Website
	if err := s.db.Preload("Domains").First(&website, opts.WebsiteID).Error; err != nil {
		return fmt.Errorf("网站不存在")
	}

	if opts.ReleaseID > 0 {
		release, err := s.releaseRepo.Get(opts.ReleaseID)
		if err != nil {
			return fmt.Errorf("正式版本不存在")
		}
		if website.PipelineID == 0 || website.PipelineID != release.PipelineID {
			return fmt.Errorf("该网站未绑定对应流水线，无法部署此正式版本")
		}
		if release.Status != "ready" {
			return fmt.Errorf("该正式版本当前不可部署")
		}
		go ProcessReleaseAppDeployment(website, release, exposePort)
		return nil
	}

	version := fmt.Sprintf("v%d", time.Now().Unix())
	releaseDir := filepath.Join(global.CONF.System.BaseDir, "www", website.Alias, "releases", version)
	sourceType := "manual"
	switch {
	case strings.TrimSpace(opts.ImageTag) != "":
		sourceType = "image"
	case strings.TrimSpace(opts.ZipPath) != "":
		sourceType = "upload"
	}
	go ProcessAppDeployment(website, 0, version, opts.ZipPath, releaseDir, "", opts.ImageTag, sourceType, exposePort)
	return nil
}

func (s *AppDeployApplicationService) Snapshot(websiteID uint) error {
	var website model.Website
	if err := s.db.First(&website, websiteID).Error; err != nil {
		return fmt.Errorf("网站不存在")
	}
	if !(website.Type == constant.Proxy && website.AppInstallID > 0) {
		return fmt.Errorf("仅容器应用支持快照")
	}

	var appInstall model.AppInstall
	if err := s.db.First(&appInstall, website.AppInstallID).Error; err != nil {
		return fmt.Errorf("关联应用不存在")
	}

	version := fmt.Sprintf("v%d", time.Now().Unix())
	deploy := model.AppDeploy{
		WebsiteID:     website.ID,
		Version:       version,
		SourceType:    "compose",
		Status:        "Running",
		LogText:       "已创建配置快照，记录了当时的 docker-compose 和环境变量配置。",
		DockerCompose: appInstall.DockerCompose,
		Env:           appInstall.Env,
		AppInstallID:  appInstall.ID,
		Port:          appInstall.HttpPort,
	}

	s.db.Model(&model.AppDeploy{}).Where("website_id = ?", website.ID).Update("is_active", false)
	deploy.IsActive = true
	if err := s.db.Create(&deploy).Error; err != nil {
		return fmt.Errorf("创建快照失败: %v", err)
	}
	return nil
}
