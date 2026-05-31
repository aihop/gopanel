package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

func NewIAppVersionService() IAppVersionService {
	return &AppVersionService{}
}

type AppVersionService struct{}
type IAppVersionService interface {
	GetUpdateInfo(checkUrl string, upgradeVersion *dto.SettingUpgradeVersion) (*dto.AppUpdateData, error)
	GoPanelVersion() (*dto.SettingAppVersion, error)
	GoPanelUpload(downloadUrl string, installPath string, versionCode int64, writeLog func(string, interface{})) error
	WriteUploadLock(installPath string, version_code int64)
	ReadUploadLock(installPath string) (int64, error)
	FileDownloadAndExtract(downloadUrl string, saveDirName string, writeLog func(string, interface{})) (string, error)
}

func (a *AppVersionService) GetUpdateInfo(checkUrl string, upgradeVersion *dto.SettingUpgradeVersion) (*dto.AppUpdateData, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", checkUrl+"?versionCode="+strconv.FormatInt(upgradeVersion.VersionCode, 10)+"&version="+upgradeVersion.VersionName+"&os="+upgradeVersion.OS+"&arch="+upgradeVersion.Arch+"&appBrand="+upgradeVersion.AppBrand+"&package="+upgradeVersion.Package+"&source="+upgradeVersion.Source, nil)
	if err != nil {
		return nil, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}
	var release *dto.AppUpdateData
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, nil
	}
	return release, nil
}
func (a *AppVersionService) WriteUploadLock(installPath string, versionCode int64) {
	uploadLockFile := filepath.Join(installPath, "update.lock")
	var content []byte
	if versionCode == 0 {
		content = []byte(time.Now().Format("200601021504"))
	} else {
		content = []byte(fmt.Sprintf("%d", versionCode))
	}
	if err := os.WriteFile(uploadLockFile, content, 0644); err != nil {
		global.LOG.Errorf("write upload lock file error, err %s", err.Error())
	}
}
func (a *AppVersionService) ReadUploadLock(installPath string) (int64, error) {
	uploadLockFile := filepath.Join(installPath, "update.lock")
	if _, err := os.Stat(uploadLockFile); err != nil {
		return 0, nil
	}
	uploadLockFileContent, err := os.ReadFile(uploadLockFile)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(string(uploadLockFileContent), 10, 64)
}
func (a *AppVersionService) GoPanelVersion() (*dto.SettingAppVersion, error) {
	installPath := global.CONF.System.BaseDir
	versionCode := int64(0)
	if strings.TrimSpace(constant.BuildVersionCode) != "" {
		parsedCode, err := strconv.ParseInt(strings.TrimSpace(constant.BuildVersionCode), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse build version code error, err %s", err.Error())
		}
		versionCode = parsedCode
	} else {
		var err error
		versionCode, err = a.ReadUploadLock(installPath)
		if err != nil {
			return nil, fmt.Errorf("read upload lock file error, err %s", err.Error())
		}
	}
	return &dto.SettingAppVersion{VersionName: constant.AppVersion, BuildTime: constant.BuildTime, VersionCode: versionCode, InstallPath: installPath}, nil
}
