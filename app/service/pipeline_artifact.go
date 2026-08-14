package service

import (
	"archive/zip"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func pipelineDirName(p *model.Pipeline) string {
	if p == nil {
		return "project"
	}
	if key := strings.TrimSpace(p.PipelineKey); key != "" {
		return key
	}
	if p.ID > 0 {
		return fmt.Sprintf("project_%d", p.ID)
	}
	return "project"
}
func pipelineBaseDir(p *model.Pipeline) string {
	return filepath.Join(global.CONF.System.BaseDir, "pipelines", pipelineDirName(p))
}
func pipelineWorkspaceDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "workspace")
}
func pipelineReleaseDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "release")
}
func pipelineArchiveDir(p *model.Pipeline) string {
	return filepath.Join(pipelineBaseDir(p), "archive")
}
func preparePipelineReleaseDir(logger *PipelineLogger, workspaceDir, releaseDir string) error {
	workspaceDir = strings.TrimSpace(workspaceDir)
	releaseDir = strings.TrimSpace(releaseDir)
	if workspaceDir == "" || releaseDir == "" {
		return fmt.Errorf("工作区目录或发布目录为空")
	}
	if err := ensurePipelineSyncPathsSafe(workspaceDir, releaseDir); err != nil {
		return err
	}
	if err := os.RemoveAll(releaseDir); err != nil {
		return err
	}
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		return err
	}
	logger.Info("正在同步工作区到发布目录...")
	if err := copyPipelineTree(workspaceDir, releaseDir, releaseExcludedNames); err != nil {
		return err
	}
	logger.Info("发布目录同步完成: %s", releaseDir)
	return nil
}

func resetPipelineReleaseSyncMarker(releaseDir string) error {
	releaseDir = strings.TrimSpace(releaseDir)
	if releaseDir == "" || releaseDir == string(filepath.Separator) {
		return fmt.Errorf("发布目录非法")
	}
	err := os.Remove(filepath.Join(releaseDir, ".gopanel_release_synced"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ensurePipelineSyncPathsSafe(workspaceDir, releaseDir string) error {
	absWorkspace, err := filepath.Abs(filepath.Clean(workspaceDir))
	if err != nil {
		return fmt.Errorf("解析工作区目录失败: %w", err)
	}
	absRelease, err := filepath.Abs(filepath.Clean(releaseDir))
	if err != nil {
		return fmt.Errorf("解析发布目录失败: %w", err)
	}
	baseDir := filepath.Dir(absWorkspace)
	if absWorkspace == string(filepath.Separator) || absRelease == string(filepath.Separator) {
		return fmt.Errorf("工作区目录或发布目录非法")
	}
	if !pathWithinBaseDir(absWorkspace, baseDir) || !pathWithinBaseDir(absRelease, baseDir) {
		return fmt.Errorf("工作区目录或发布目录超出流水线工作目录范围")
	}
	if absRelease == baseDir || absRelease == absWorkspace {
		return fmt.Errorf("发布目录非法: %s", absRelease)
	}
	return nil
}

func pathWithinBaseDir(target, base string) bool {
	target = filepath.Clean(target)
	base = filepath.Clean(base)
	if target == base {
		return true
	}
	return strings.HasPrefix(target, base+string(filepath.Separator))
}
func copyPipelineTree(srcDir, dstDir string, excluded map[string]struct{}) error {
	return filepath.Walk(srcDir, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipPipelineReleaseEntry(rel, info, excluded) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dstDir, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(current)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return copyPipelineFile(current, target, info.Mode())
	})
}
func shouldSkipPipelineReleaseEntry(rel string, info os.FileInfo, excluded map[string]struct{}) bool {
	cleanRel := filepath.ToSlash(rel)
	name := info.Name()
	if name == ".DS_Store" || strings.HasPrefix(name, "._") {
		return true
	}
	if cleanRel == "node_modules" {
		return true
	}
	if _, ok := excluded[name]; ok {
		return true
	}
	for _, part := range strings.Split(cleanRel, "/") {
		if _, ok := excluded[part]; ok {
			return true
		}
	}
	return false
}
func copyPipelineFile(srcPath, dstPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
func createFilteredZipArchive(srcPath, archivePath string, preserveNestedDependencies bool) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(archivePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	rootName := filepath.Base(filepath.Clean(srcPath))
	if !info.IsDir() {
		return addArchiveFileToZip(zw, srcPath, filepath.ToSlash(rootName), info)
	}
	rootHeader := &zip.FileHeader{Name: filepath.ToSlash(rootName) + "/", Method: zip.Deflate, Modified: info.ModTime()}
	rootHeader.SetMode(info.Mode())
	if _, err := zw.CreateHeader(rootHeader); err != nil {
		return err
	}
	return filepath.Walk(srcPath, func(current string, currentInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcPath, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipArchiveEntry(rel, currentInfo, preserveNestedDependencies) {
			if currentInfo.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		nameInArchive := filepath.ToSlash(filepath.Join(rootName, rel))
		if currentInfo.IsDir() {
			header := &zip.FileHeader{Name: nameInArchive + "/", Method: zip.Deflate, Modified: currentInfo.ModTime()}
			header.SetMode(currentInfo.Mode())
			_, err = zw.CreateHeader(header)
			return err
		}
		return addArchiveFileToZip(zw, current, nameInArchive, currentInfo)
	})
}
func shouldSkipArchiveEntry(rel string, info os.FileInfo, preserveNestedDependencies bool) bool {
	name := info.Name()
	if name == ".DS_Store" || strings.HasPrefix(name, "._") {
		return true
	}
	cleanRel := filepath.ToSlash(filepath.Clean(rel))
	if _, ok := archiveExcludedNames[name]; ok && !(preserveNestedDependencies && name == "node_modules" && cleanRel != "node_modules") {
		return true
	}
	for _, part := range strings.Split(cleanRel, "/") {
		if preserveNestedDependencies && part == "node_modules" && cleanRel != "node_modules" && !strings.HasPrefix(cleanRel, "node_modules/") {
			continue
		}
		if _, ok := archiveExcludedNames[part]; ok {
			return true
		}
	}
	return false
}
func addArchiveFileToZip(zw *zip.Writer, diskPath, nameInArchive string, info os.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = nameInArchive
	header.Method = zip.Deflate
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}
