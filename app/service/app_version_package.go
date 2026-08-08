package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpc"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (a *AppVersionService) GoPanelUpload(downloadUrl, checksum, installPath string, versionCode int64, versionName string, writeLog func(string, interface{})) error {
	var err error
	if err := validateUpdateDownloadURL(downloadUrl); err != nil {
		return err
	}
	checksum, err = resolveUpdateChecksum(downloadUrl, checksum)
	if err != nil {
		return err
	}
	filesUtil := files.NewFileOp()
	saveDirName := ""
	sourcePath, err := a.FileDownloadAndExtract(downloadUrl, checksum, saveDirName, writeLog)
	if err != nil {
		return err
	}
	tmpFolder := filepath.Dir(sourcePath)
	defer filesUtil.DeleteDir(tmpFolder)
	writeLog("start replace file", sourcePath)
	if entries, err := os.ReadDir(sourcePath); err == nil {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		writeLog("extracted dir entries", names)
	} else {
		writeLog("read extracted dir failed", err)
	}
	findPath := func(base string, candidates ...string) string {
		for _, c := range candidates {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return ""
	}
	gopanelFile := findPath(sourcePath, filepath.Join("gopanel", "gopanel"), "gopanel", filepath.Join("gopanel", "gopanel.exe"), "gopanel.exe", filepath.Join("bin", "gopanel"))
	gpcFile := findPath(sourcePath, filepath.Join("gpc", "gpc"), "gpc", filepath.Join("bin", "gpc"))
	gpcUpdateScheduled := false
	if gopanelFile == "" {
		writeLog("no gopanel binary found in extracted package", sourcePath)
		return fmt.Errorf("update package does not contain gopanel binary")
	}
	{
		writeLog("detected binary file", gopanelFile)
		targetBin := filepath.Join(installPath, filepath.Base(gopanelFile))
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
		if runtime.GOOS == "windows" {
			if _, err := os.Stat(targetBin); err == nil {
				oldBin := targetBin + ".old"
				os.Remove(oldBin)
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
	}
	if runtime.GOOS != "windows" {
		if gpcFile != "" {
			writeLog("detected gpc binary file", gpcFile)
			helperCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			resp, err := gpc.Do(helperCtx, "GOPANEL_GPC_INSTALL", map[string]interface{}{"source_path": gpcFile})
			if err != nil {
				writeLog("gpc helper update skipped", err)
				if resp != nil && strings.TrimSpace(resp.Output) != "" {
					writeLog("gpc helper update output", resp.Output)
				}
			} else {
				if strings.TrimSpace(resp.Output) != "" {
					writeLog("gpc helper update output", resp.Output)
				}
				gpcUpdateScheduled = true
				writeLog("gpc helper update scheduled", "/usr/local/bin/gpc")
			}
		} else {
			writeLog("no gpc binary found in extracted package", sourcePath)
		}
	}
	if info, err := os.Stat(installPath); err == nil {
		uid, gid := files.GetUidGid(info)
		if uid >= 0 && gid >= 0 {
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
	writeLog("restart panel", runtime.GOOS)
	restartDelay := time.Second
	if gpcUpdateScheduled {
		restartDelay = 5 * time.Second
		writeLog("wait for gpc helper restart", restartDelay)
	}
	if err := cmd.RestartGoPanelWithDelay(restartDelay); err != nil {
		writeLog("restart error", err.Error())
		return fmt.Errorf("schedule gopanel restart: %w", err)
	}
	a.WriteUploadLock(installPath, versionCode)
	writeLog("restart scheduled", "gopanel restart has been triggered")
	writeLog("-------------------------------", "successful update to version_code "+fmt.Sprintf("%d", versionCode))
	TrackEvent(TrackEventUpgradeSuccess, versionName)
	return nil
}
func (a *AppVersionService) FileDownloadAndExtract(downloadUrl, checksum, saveDirName string, writeLog func(string, interface{})) (string, error) {
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
	writeLog("start download file", downloadUrl)
	key, err := NewIFileService().Wget(request.FileWget{Url: downloadUrl, Path: tmpFolder, Name: filepath.Base(tmpFile), IgnoreCertificate: false})
	if err != nil {
		writeLog("download start failed", err)
		return "", err
	}
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
	if _, err := os.Stat(tmpFile); err != nil {
		writeLog("downloaded file not found", tmpFile)
		return "", fmt.Errorf("downloaded file not found: %s", tmpFile)
	}
	if err := verifyFileSHA256(tmpFile, checksum); err != nil {
		writeLog("update package checksum mismatch", err)
		return "", err
	}
	writeLog("update package checksum verified", checksum)
	writeLog("start extract file", tmpFile)
	extractedDir, err := extract(tmpFile, suffix)
	if err != nil {
		writeLog("failed extract file", err)
		return "", err
	}
	var sourcePath string
	if saveDirName == "" {
		sourcePath = extractedDir
	} else {
		sourcePath = filepath.Join(filepath.Dir(extractedDir), saveDirName)
		if filepath.Base(extractedDir) != saveDirName {
			if _, statErr := os.Stat(sourcePath); statErr == nil {
				if rmErr := filesUtil.DeleteDir(sourcePath); rmErr != nil {
					writeLog("failed remove existing sourcePath before rename", rmErr)
				}
			}
			writeLog("rename folder", map[ // 目标路径处理：如果没有传入 saveDirName 则直接使用解压出的顶层目录
			string]string{"from": extractedDir, "to": sourcePath})
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

func resolveUpdateChecksum(downloadURL, supplied string) (string, error) {
	if checksum, err := normalizeSHA256(supplied); err == nil && checksum != "" {
		return checksum, nil
	} else if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(downloadURL + ".sha256")
	if err != nil {
		return "", fmt.Errorf("download update checksum: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update checksum unavailable: HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read update checksum: %w", err)
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return "", fmt.Errorf("update checksum is empty")
	}
	return normalizeSHA256(fields[0])
}

func normalizeSHA256(checksum string) (string, error) {
	checksum = strings.ToLower(strings.TrimSpace(checksum))
	if checksum == "" {
		return "", nil
	}
	decoded, err := hex.DecodeString(checksum)
	if err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("invalid update SHA-256 checksum")
	}
	return checksum, nil
}

func verifyFileSHA256(filePath, expected string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("update SHA-256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
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
	if err = files.NewFileOp().DeleteFile(tmpFile); err != nil {
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
