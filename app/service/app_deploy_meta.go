package service

import (
	"archive/zip"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func BuildOtherDomains(w model.Website) string {
	var domains []string
	if w.Domains != nil {
		for _, d := range w.Domains {
			if normalizeWebsiteDomainForCompare(d.Domain) != normalizeWebsiteDomainForCompare(w.PrimaryDomain) {
				domains = append(domains, d.Domain)
			}
		}
	}
	return strings.Join(domains, "\n")
}
func buildDeployCaddyDomain(website model.Website) string {
	return BuildWebsiteCaddyDomain(website.PrimaryDomain, website.Protocol)
}
func appendPipelineDeployInfoLog(pipelineRecordID uint, websiteAlias, msg string) {
	if pipelineRecordID == 0 || !IsPipelineLoggerActive(pipelineRecordID) {
		return
	}
	GetPipelineLogger(pipelineRecordID).Info("[%s] %s", websiteAlias, msg)
}
func appendPipelineDeployErrorLog(pipelineRecordID uint, websiteAlias, msg string) {
	if pipelineRecordID == 0 || !IsPipelineLoggerActive(pipelineRecordID) {
		return
	}
	GetPipelineLogger(pipelineRecordID).Error("[%s] %s", websiteAlias, msg)
}
func UnzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}
	cleanDest := filepath.Clean(dest) + string(os.PathSeparator)
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		cleanPath := filepath.Clean(fpath)
		if !strings.HasPrefix(cleanPath, cleanDest) {
			return fmt.Errorf("非法压缩包路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(cleanPath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(cleanPath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(cleanPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
