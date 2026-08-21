package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/i18n"
	"github.com/aihop/gopanel/utils/gpc"
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
		req.Limit,
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
		req.Limit,
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
	seen := make(map[string]struct{})
	if err := filepath.Walk(logDir, func(pathItem string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "gopanel") {
			if info.Name() == "gopanel.log" {
				name := time.Now().Format("2006-01-02")
				if _, exists := seen[name]; !exists {
					files = append(files, name)
					seen[name] = struct{}{}
				}
				return nil
			}
			itemFileName := strings.TrimPrefix(info.Name(), "gopanel-")
			itemFileName = strings.TrimSuffix(itemFileName, ".gz")
			itemFileName = strings.TrimSuffix(itemFileName, ".log")
			if _, exists := seen[itemFileName]; !exists {
				files = append(files, itemFileName)
				seen[itemFileName] = struct{}{}
			}
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

func (u *LogService) ReadSystemLog(name string, maxBytes int64) (*dto.SystemLogContent, error) {
	return readSystemLogTail(global.CONF.System.LogPath, name, maxBytes)
}

func (u *LogService) PageSSHLoginLog(req dto.SearchSSHLogWithPage) (*dto.SSHLoginLogResult, error) {
	resp, err := gpc.Do(context.Background(), "SSH_LOGIN_LOG_LIST", map[string]interface{}{
		"page":     req.Page,
		"limit":    req.Limit,
		"ip":       req.IP,
		"status":   req.Status,
		"username": req.Username,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unsupported platform") {
			return &dto.SSHLoginLogResult{
				Supported: false,
				Warning:   i18n.GetMsg(constant.ErrLogSshLoginUnsupported),
				Items:     []dto.SSHLoginLog{},
			}, nil
		}
		return nil, err
	}

	var result dto.SSHLoginLogResult
	if err := json.Unmarshal([]byte(resp.Output), &result); err != nil {
		return nil, err
	}
	if result.Items == nil {
		result.Items = []dto.SSHLoginLog{}
	}
	return &result, nil
}

func (u *LogService) CleanLogs(logtype string) error {
	logRepo := repo.NewILogRepo()
	if logtype == "operation" {
		return logRepo.CleanOperation()
	}
	return logRepo.CleanLogin()
}
