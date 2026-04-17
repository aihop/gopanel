package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
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
	req, err := http.NewRequest("GET", checkUrl+"?versionCode="+strconv.FormatInt(upgradeVersion.VersionCode, 10)+"&version="+upgradeVersion.VersionName+"&os="+upgradeVersion.OS+"&arch="+upgradeVersion.Lang+"&appBrand="+upgradeVersion.AppBrand, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GoPanel API 返回异常状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	var release *dto.AppUpdateData
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析 GoPanel 响应失败: %v", err)
	}
	return release, nil
}

// 写入更新锁文件
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

// 读取更新锁文件,如果文件不存在，返回空字符串
func (a *AppVersionService) ReadUploadLock(installPath string) (int64, error) {
	// 读取更新锁文件
	uploadLockFile := filepath.Join(installPath, "update.lock")
	if _, err := os.Stat(uploadLockFile); err != nil {
		return 0, nil
	}
	// 读取更新锁文件
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
	return &dto.SettingAppVersion{
		VersionName: constant.AppVersion,
		BuildTime:   constant.BuildTime,
		VersionCode: versionCode,
		InstallPath: installPath,
	}, nil
}

func (a *AppVersionService) GoPanelUpload(downloadUrl string, installPath string, versionCode int64, writeLog func(string, interface{})) error {
	var err error
	filesUtil := files.NewFileOp()

	saveDirName := ""
	sourcePath, err := a.FileDownloadAndExtract(downloadUrl, saveDirName, writeLog)
	if err != nil {
		return err
	}
	tmpFolder := filepath.Dir(sourcePath)
	defer filesUtil.DeleteDir(tmpFolder)

	//  gopanel 的更新现在只需要对二进制文件更新，前端 public 等资源已编译进单文件
	writeLog("start replace file", sourcePath)

	// 列出解压目录内容，方便排查
	if entries, err := os.ReadDir(sourcePath); err == nil {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		writeLog("extracted dir entries", names)
	} else {
		writeLog("read extracted dir failed", err)
	}

	// helper：在 base 下按优先级查找候选路径
	findPath := func(base string, candidates ...string) string {
		for _, c := range candidates {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return ""
	}

	// 自动探测二进制文件路径（兼容不同布局）
	gopanelFile := findPath(sourcePath,
		filepath.Join("gopanel", "gopanel"),
		"gopanel",
		filepath.Join("gopanel", "gopanel.exe"),
		"gopanel.exe",
		filepath.Join("bin", "gopanel"),
	)
	if gopanelFile != "" {
		writeLog("detected binary file", gopanelFile)
		targetBin := filepath.Join(installPath, filepath.Base(gopanelFile))

		// 在安装目录创建临时文件以保证同一挂载点
		tmpBinFile, err := os.CreateTemp(installPath, ".gopanel_tmp_*")
		if err != nil {
			tmpBinFile, err = os.CreateTemp("", ".gopanel_tmp_*")
		}
		if err != nil {
			writeLog("failed create tmp file for binary", err)
			return err
		}
		tmpBinPath := tmpBinFile.Name()
		tmpBinFile.Close()

		// 直接复制文件内容到 tmpBinPath，避免 filesUtil.Copy 对目标做为目录的假设
		inF, err := os.Open(gopanelFile)
		if err != nil {
			writeLog("open source binary failed", err)
			_ = os.Remove(tmpBinPath)
			return err
		}
		outF, err := os.OpenFile(tmpBinPath, os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			inF.Close()
			writeLog("open tmp bin file failed", err)
			_ = os.Remove(tmpBinPath)
			return err
		}
		if _, err := io.Copy(outF, inF); err != nil {
			inF.Close()
			outF.Close()
			writeLog("copy binary content failed", err)
			_ = os.Remove(tmpBinPath)
			return err
		}
		inF.Close()
		if err := outF.Close(); err != nil {
			writeLog("close tmp bin file failed", err)
			_ = os.Remove(tmpBinPath)
			return err
		}

		if err := os.Chmod(tmpBinPath, 0755); err != nil {
			writeLog("failed chmod tmp binary", err)
			_ = os.Remove(tmpBinPath)
			return err
		}

		// On Windows, replacing a running executable directly fails with Access is denied.
		// We need to move the existing binary out of the way first.
		if runtime.GOOS == "windows" {
			if _, err := os.Stat(targetBin); err == nil {
				oldBin := targetBin + ".old"
				os.Remove(oldBin) // remove any previous old binary
				if err := os.Rename(targetBin, oldBin); err != nil {
					writeLog("failed to move existing binary on windows", err)
				}
			}
		}

		if err := os.Rename(tmpBinPath, targetBin); err != nil {
			writeLog("rename tmp->target failed, try copy fallback", err)
			if err2 := filesUtil.Copy(tmpBinPath, targetBin); err2 != nil {
				writeLog("fallback copy to target failed", err2)
				_ = os.Remove(tmpBinPath)
				return err2
			}
			_ = os.Remove(tmpBinPath)
		}

		if fi, err := os.Stat(targetBin); err != nil {
			writeLog("target stat failed after replace", err)
			return err
		} else {
			writeLog("target replaced", map[string]interface{}{"path": targetBin, "size": fi.Size(), "mode": fi.Mode().String()})
		}

		if err := os.Chmod(targetBin, 0755); err != nil {
			writeLog("chmod failed on target", err)
			return err
		}
	} else {
		writeLog("no gopanel binary found in extracted package", sourcePath)
	}

	// 替换成功后，尝试把文件属主设置为 installPath 的属主（若可行）
	if info, err := os.Stat(installPath); err == nil {
		uid, gid := files.GetUidGid(info)
		if uid >= 0 && gid >= 0 {
			// chown binary
			targetBin := filepath.Join(installPath, "gopanel")
			_ = os.Chown(targetBin, uid, gid)
			_ = os.Chmod(targetBin, 0755)
			writeLog("chown/chmod applied", map[string]interface{}{"uid": uid, "gid": gid})
		} else {
			writeLog("skip chown: cannot determine uid/gid for installPath", installPath)
		}
	} else {
		writeLog("stat installPath failed for chown", err)
	}

	// 使用接收者写更新锁
	a.WriteUploadLock(installPath, versionCode)

	writeLog("-------------------------------", "successful update to version_code "+fmt.Sprintf("%d", versionCode))
	// 尝试跨平台重启面板
	writeLog("restart panel", runtime.GOOS)
	if err := cmd.RestartGoPanel(); err != nil {
		writeLog("restart error", err.Error())
	} else {
		writeLog("restart scheduled", "gopanel restart has been triggered")
	}
	return nil
}

// 步骤
// 1. 从 downloadUrl 下载文件
// 2. 解压文件到临时文件夹
// 3. 重命名解压的文件夹为 saveDirName
// 4. 返回解压后的(临时文件夹路径/saveDirName)
func (a *AppVersionService) FileDownloadAndExtract(downloadUrl string, saveDirName string, writeLog func(string, interface{})) (string, error) {
	// 参数与路径准备
	suffix := strings.ToLower(common.GetFileExt(path.Base(downloadUrl)))
	if suffix == ".tgz" {
		suffix = ".tar.gz"
	}
	if suffix == "" {
		return "", fmt.Errorf("unknown archive type for url: %s", downloadUrl)
	}

	tmpFolder := filepath.Join(global.CONF.System.TmpDir, common.RandStr(10))
	if err := os.MkdirAll(tmpFolder, 0o755); err != nil {
		writeLog("create tmp folder failed", err)
		return "", err
	}

	tmpFile := filepath.Join(tmpFolder, common.RandStr(10)+suffix)
	filesUtil := files.NewFileOp()

	writeLog("create tmp folder", tmpFolder)

	// 发起异步下载（使用已有的 Wget 接口）
	writeLog("start download file", downloadUrl)
	key, err := NewIFileService().Wget(request.FileWget{
		Url:               downloadUrl,
		Path:              tmpFolder,
		Name:              filepath.Base(tmpFile),
		IgnoreCertificate: false,
	})
	if err != nil {
		writeLog("download start failed", err)
		return "", err
	}

	// 等待下载完成，使用单一超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			writeLog("failed download timed out", "")
			return "", fmt.Errorf("download timed out")
		case <-ticker.C:
			progressBytes, getErr := global.CACHE.Get(key)
			if getErr != nil {
				// 仍然轮询，直到超时或成功；记录日志但不立即失败
				writeLog("waiting download cache key", getErr)
				continue
			}
			var p files.Process
			if err := json.Unmarshal(progressBytes, &p); err != nil {
				writeLog("failed parse progress", err)
				continue
			}
			writeLog("download", fmt.Sprintf("%s/%s  %.2f%%", common.FormatBytes(p.Written), common.FormatBytes(p.Total), p.Percent))
			if p.Percent >= 100 {
				writeLog("download finished", "")
				goto DOWNLOAD_DONE
			}
		}
	}
DOWNLOAD_DONE:

	// 确认文件存在
	if _, err := os.Stat(tmpFile); err != nil {
		writeLog("downloaded file not found", tmpFile)
		return "", fmt.Errorf("downloaded file not found: %s", tmpFile)
	}

	// 解压并清理压缩包（extract 会删除压缩包）
	writeLog("start extract file", tmpFile)
	extractedDir, err := extract(tmpFile, suffix)
	if err != nil {
		writeLog("failed extract file", err)
		// 保留 tmpFolder 清理给调用方 defer
		return "", err
	}

	// 目标路径处理：如果没有传入 saveDirName 则直接使用解压出的顶层目录
	var sourcePath string
	if saveDirName == "" {
		sourcePath = extractedDir
	} else {
		sourcePath = filepath.Join(filepath.Dir(extractedDir), saveDirName)
		if filepath.Base(extractedDir) != saveDirName {
			// 若目标已存在，先删除或备份（这里选择删除）
			if _, statErr := os.Stat(sourcePath); statErr == nil {
				if rmErr := filesUtil.DeleteDir(sourcePath); rmErr != nil {
					writeLog("failed remove existing sourcePath before rename", rmErr)
					// 继续尝试重命名，可能失败
				}
			}
			writeLog("rename folder", map[string]string{"from": extractedDir, "to": sourcePath})
			if err = filesUtil.Rename(extractedDir, sourcePath); err != nil {
				writeLog("failed rename folder", err)
				return "", err
			}
		} else {
			sourcePath = extractedDir
		}
	}
	return sourcePath, nil
}

func extract(tmpFile string, suffix string) (string, error) {
	var compressType files.CompressType
	switch suffix {
	case ".zip":
		compressType = files.Zip
	case ".tar.gz":
		compressType = files.TarGz
	case ".tar":
		compressType = files.Tar
	default:
		return "", fmt.Errorf("unsupported archive type: %s", suffix)
	}

	archiver, err := files.NewShellArchiver(compressType)
	if err != nil {
		return "", err
	}

	dstDir := filepath.Dir(tmpFile)
	if err = archiver.Extract(tmpFile, dstDir, ""); err != nil {
		return "", err
	}

	// 删除压缩包
	if err = files.NewFileOp().DeleteFile(tmpFile); err != nil {
		// 记录但不阻止返回解压路径
		return "", fmt.Errorf("remove archive failed: %w", err)
	}

	entries, err := os.ReadDir(dstDir)
	if err != nil {
		return "", fmt.Errorf("read dir failed: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(dstDir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no directory found after extract")
}
