package service

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/jinzhu/copier"
)

type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

func (u *LogService) CreateLoginLog(operation model.LoginLog) error {
	logRepo := repo.NewILogRepo()
	return logRepo.CreateLoginLog(&operation)
}

func (u *LogService) PageLoginLog(req dto.SearchLgLogWithPage) (int64, interface{}, error) {
	logRepo := repo.NewILogRepo()
	options := []repo.DBOption{}
	if len(req.IP) != 0 {
		options = append(options, logRepo.WithByIP(req.IP))
	}
	if len(req.Status) != 0 {
		options = append(options, logRepo.WithByStatus(req.Status))
	}
	total, ops, err := logRepo.PageLoginLog(
		req.Page,
		req.PageSize,
		options...,
	)
	var dtoOps []dto.LoginLog
	for _, op := range ops {
		var item dto.LoginLog
		if err := copier.Copy(&item, &op); err != nil {
			return 0, nil, err
		}
		dtoOps = append(dtoOps, item)
	}
	return total, dtoOps, err
}

func (u *LogService) CreateOperationLog(operation *model.OperationLog) error {
	logRepo := repo.NewILogRepo()
	return logRepo.CreateOperationLog(operation)
}

func (u *LogService) PageOperationLog(req dto.SearchOpLogWithPage) (int64, interface{}, error) {
	logRepo := repo.NewILogRepo()
	options := []repo.DBOption{
		logRepo.WithByLikeOperation(req.Operation),
	}
	if len(req.Source) != 0 {
		options = append(options, logRepo.WithBySource(req.Source))
	}
	if len(req.Status) != 0 {
		options = append(options, logRepo.WithByStatus(req.Status))
	}

	total, ops, err := logRepo.PageOperationLog(
		req.Page,
		req.PageSize,
		options...,
	)
	var dtoOps []dto.OperationLog
	for _, op := range ops {
		var item dto.OperationLog
		if err := copier.Copy(&item, &op); err != nil {
			return 0, nil, err
		}
		dtoOps = append(dtoOps, item)
	}
	return total, dtoOps, err
}

func (u *LogService) ListSystemLogFile() ([]string, error) {
	logDir := global.CONF.System.LogPath
	var files []string
	if err := filepath.Walk(logDir, func(pathItem string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "gopanel") {
			if info.Name() == "gopanel.log" {
				files = append(files, time.Now().Format("2006-01-02"))
				return nil
			}
			itemFileName := strings.TrimPrefix(info.Name(), "gopanel-")
			itemFileName = strings.TrimSuffix(itemFileName, ".gz")
			itemFileName = strings.TrimSuffix(itemFileName, ".log")
			files = append(files, itemFileName)
			return nil
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if len(files) < 2 {
		return files, nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	return files, nil
}

func (u *LogService) ReadSystemLog(name string) (string, error) {
	logDir := global.CONF.System.LogPath
	fileName := ""
	if name == time.Now().Format("2006-01-02") {
		fileName = "gopanel.log"
	} else {
		fileName = fmt.Sprintf("gopanel-%s.log.gz", name)
	}

	fullPath := filepath.Join(logDir, fileName)

	if strings.HasSuffix(fileName, ".gz") {
		f, err := os.Open(fullPath)
		if err != nil {
			return "", err
		}
		defer f.Close()

		gr, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gr.Close()

		data, err := io.ReadAll(gr)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (u *LogService) CleanLogs(logtype string) error {
	logRepo := repo.NewILogRepo()
	if logtype == "operation" {
		return logRepo.CleanOperation()
	}
	return logRepo.CleanLogin()
}
