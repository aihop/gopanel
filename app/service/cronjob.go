package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
	"github.com/robfig/cron/v3"
)

type CronjobService struct {
	repo *repo.CronjobRepo
}

func NewCronjobService() *CronjobService {
	return &CronjobService{repo: repo.NewCronjob()}
}

func validateCronjobTypeFields(jobType, script, dbType, dbName, logType string, serverID uint) error {
	switch jobType {
	case "shell":
		if strings.TrimSpace(script) == "" {
			return errors.New("脚本内容不能为空")
		}
	case "db_backup":
		if serverID == 0 {
			return errors.New("请选择数据库服务器")
		}
		if dbType == "" || dbName == "" {
			return errors.New("请选择数据库类型和数据库")
		}
	case "log_clean":
		if logType == "" {
			return errors.New("请选择要清理的日志类型")
		}
	case "ssl_renew":
		// 不需要额外字段
	default:
		return fmt.Errorf("不支持的任务类型: %s", jobType)
	}
	return nil
}

// scheduleJob 把任务加入正在运行的调度器，返回新的 EntryID；Status 非 Enable 时直接返回 0
func scheduleJob(job *model.Cronjob) (int, error) {
	if job.Status != constant.StatusEnable {
		return 0, nil
	}
	if global.Cron == nil {
		return 0, errors.New("调度器尚未初始化")
	}
	entryID, err := global.Cron.AddFunc(job.Spec, func() {
		NewCronjobService().Run(job.ID)
	})
	if err != nil {
		return 0, err
	}
	return int(entryID), nil
}

// unscheduleJob 从调度器里移除一个已注册的任务
func unscheduleJob(entryID int) {
	if entryID == 0 || global.Cron == nil {
		return
	}
	global.Cron.Remove(cron.EntryID(entryID))
}

func (s *CronjobService) Create(req *request.CronjobCreate) (*model.Cronjob, error) {
	if err := validateCronjobTypeFields(req.Type, req.Script, req.DBType, req.DBName, req.LogType, req.ServerID); err != nil {
		return nil, err
	}
	job := &model.Cronjob{
		Name:         req.Name,
		Type:         req.Type,
		Spec:         req.Spec,
		Status:       constant.StatusEnable,
		Script:       req.Script,
		ServerID:     req.ServerID,
		DBType:       req.DBType,
		DBName:       req.DBName,
		RetainCopies: req.RetainCopies,
		LogType:      req.LogType,
	}
	if err := s.repo.Create(job); err != nil {
		return nil, err
	}
	entryID, err := scheduleJob(job)
	if err != nil {
		global.LOG.Errorf("[Cronjob] schedule job %d failed: %v", job.ID, err)
	} else if entryID != 0 {
		job.EntryID = entryID
		_ = s.repo.UpdateEntryID(job.ID, entryID)
	}
	return job, nil
}

func (s *CronjobService) Update(req *request.CronjobUpdate) error {
	if err := validateCronjobTypeFields(req.Type, req.Script, req.DBType, req.DBName, req.LogType, req.ServerID); err != nil {
		return err
	}
	job, err := s.repo.Get(req.ID)
	if err != nil {
		return err
	}
	unscheduleJob(job.EntryID)

	job.Name = req.Name
	job.Type = req.Type
	job.Spec = req.Spec
	job.Script = req.Script
	job.ServerID = req.ServerID
	job.DBType = req.DBType
	job.DBName = req.DBName
	job.RetainCopies = req.RetainCopies
	job.LogType = req.LogType
	job.EntryID = 0

	entryID, err := scheduleJob(job)
	if err != nil {
		global.LOG.Errorf("[Cronjob] schedule job %d failed: %v", job.ID, err)
	} else {
		job.EntryID = entryID
	}
	return s.repo.Update(job)
}

func (s *CronjobService) SetStatus(id uint, status string) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	unscheduleJob(job.EntryID)
	job.Status = status
	job.EntryID = 0
	entryID, err := scheduleJob(job)
	if err != nil {
		global.LOG.Errorf("[Cronjob] schedule job %d failed: %v", job.ID, err)
	} else {
		job.EntryID = entryID
	}
	return s.repo.Update(job)
}

func (s *CronjobService) Get(id uint) (*model.Cronjob, error) {
	return s.repo.Get(id)
}

func (s *CronjobService) Delete(id uint) error {
	job, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	unscheduleJob(job.EntryID)
	if err := s.repo.DeleteRecords(id); err != nil {
		global.LOG.Errorf("[Cronjob] delete records of job %d failed: %v", id, err)
	}
	return s.repo.Delete(id)
}

func (s *CronjobService) List(ctx *gormx.Contextx) ([]*model.Cronjob, error) {
	jobs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}
	latest, err := s.repo.ListLatestRecords(ids)
	if err != nil {
		global.LOG.Errorf("[Cronjob] load latest records failed: %v", err)
		return jobs, nil
	}
	for _, job := range jobs {
		job.LastRecord = latest[job.ID]
	}
	return jobs, nil
}

func (s *CronjobService) CountByWhere(where *gormx.Wherex) (int64, error) {
	return s.repo.CountByWhere(where)
}

func (s *CronjobService) RecordList(cronjobID uint, limit int) ([]*model.JobRecords, error) {
	return s.repo.ListRecords(cronjobID, limit)
}

func (s *CronjobService) RecordDelete(cronjobID uint) error {
	return s.repo.DeleteRecords(cronjobID)
}

// Run 执行一次任务，调度器触发和"立即执行"共用这个入口
func (s *CronjobService) Run(cronjobID uint) {
	job, err := s.repo.Get(cronjobID)
	if err != nil {
		global.LOG.Errorf("[Cronjob] load job %d failed: %v", cronjobID, err)
		return
	}

	record := &model.JobRecords{CronjobID: job.ID, StartTime: time.Now(), Status: constant.StatusRunning}
	if err := s.repo.CreateRecord(record); err != nil {
		global.LOG.Errorf("[Cronjob] create record for job %d failed: %v", job.ID, err)
	}

	var message string
	var runErr error
	switch job.Type {
	case "shell":
		message, runErr = s.runShell(job)
	case "db_backup":
		message, runErr = s.runDBBackup(job)
	case "log_clean":
		message, runErr = s.runLogClean(job)
	case "ssl_renew":
		message, runErr = s.runSSLRenew(job)
	default:
		runErr = fmt.Errorf("不支持的任务类型: %s", job.Type)
	}

	status := constant.StatusSuccess
	if runErr != nil {
		status = constant.StatusFailed
		if message != "" {
			message += "\n"
		}
		message += "错误: " + runErr.Error()
	}
	if record.ID != 0 {
		_ = s.repo.UpdateRecord(record.ID, map[string]interface{}{
			"end_time": time.Now(),
			"status":   status,
			"message":  message,
		})
	}
}

func (s *CronjobService) runShell(job *model.Cronjob) (string, error) {
	return cmd.ExecWithTimeOut(job.Script, time.Hour)
}

func (s *CronjobService) runDBBackup(job *model.Cronjob) (string, error) {
	server, err := repo.NewDatabaseServer().Get(job.ServerID)
	if err != nil {
		return "", fmt.Errorf("获取数据库服务器失败: %v", err)
	}

	key := "cron_" + common.RandStrAndNum(20)
	logger := GetBackupLogger(key)
	defer RemoveBackupLogger(key)

	req := &dto.CommonBackup{
		Type:       job.DBType,
		Name:       server.Name,
		DetailName: job.DBName,
		DetailId:   job.ServerID,
	}
	backupService := NewBackup()
	switch job.DBType {
	case "mysql", "mariadb":
		err = backupService.MysqlBackup(req, logger)
	case constant.AppPostgresql:
		err = backupService.PostgresqlBackup(req, logger)
	default:
		err = fmt.Errorf("不支持的数据库类型: %s", job.DBType)
	}
	message := strings.Join(logger.GetLogs(), "\n")
	if err != nil {
		return message, err
	}

	if job.RetainCopies > 0 {
		if pruneErr := s.pruneBackupRecords(job.DBType, server.Name, job.DBName, job.RetainCopies); pruneErr != nil {
			message += "\n清理旧备份失败: " + pruneErr.Error()
		}
	}
	return message, nil
}

func (s *CronjobService) pruneBackupRecords(dbType, serverName, dbName string, retain int) error {
	backupRecordRepo := repo.NewBackupRecord()
	records, err := backupRecordRepo.ListRecord(
		backupRecordRepo.WithByType(dbType),
		backupRecordRepo.WithByDetailName(dbName),
	)
	if err != nil {
		return err
	}
	filtered := make([]model.BackupRecord, 0, len(records))
	for _, r := range records {
		if r.Name == serverName {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) <= retain {
		return nil
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })
	overflow := filtered[retain:]
	ids := make([]uint, 0, len(overflow))
	for _, r := range overflow {
		ids = append(ids, r.ID)
	}
	return NewBackup().BatchDeleteRecord(ids)
}

func (s *CronjobService) runLogClean(job *model.Cronjob) (string, error) {
	logSvc := NewLogService()
	logType := job.LogType
	if logType == "" {
		logType = "all"
	}
	var cleaned []string
	if logType == "operation" || logType == "all" {
		if err := logSvc.CleanLogs("operation"); err != nil {
			return strings.Join(cleaned, "\n"), fmt.Errorf("清理操作日志失败: %v", err)
		}
		cleaned = append(cleaned, "操作日志已清理")
	}
	if logType == "login" || logType == "all" {
		if err := logSvc.CleanLogs("login"); err != nil {
			return strings.Join(cleaned, "\n"), fmt.Errorf("清理登录日志失败: %v", err)
		}
		cleaned = append(cleaned, "登录日志已清理")
	}
	return strings.Join(cleaned, "\n"), nil
}

// runSSLRenew 续签所有开启了自动续签且临近到期的证书，逻辑与原 init/cron 里硬编码的每日任务一致
func (s *CronjobService) runSSLRenew(_ *model.Cronjob) (string, error) {
	sslService := NewSSL()

	var certs []model.SSL
	if err := global.DB.Where("auto_renew = ?", true).Find(&certs).Error; err != nil {
		return "", fmt.Errorf("查询需要自动续签的证书失败: %v", err)
	}

	now := time.Now()
	var lines []string
	var lastErr error
	for _, certItem := range certs {
		if certItem.Type == "upload" || certItem.Type == "caddy" {
			continue
		}
		if !certItem.ExpireDate.AddDate(0, 0, -7).Before(now) {
			continue
		}
		if err := sslService.Renew(certItem.ID); err != nil {
			lastErr = err
			lines = append(lines, fmt.Sprintf("证书 %s 自动续签请求失败: %v", certItem.PrimaryDomain, err))
			continue
		}
		lines = append(lines, fmt.Sprintf("证书 %s 已提交续签", certItem.PrimaryDomain))
	}
	if len(lines) == 0 {
		lines = append(lines, "没有需要续签的证书")
	}
	return strings.Join(lines, "\n"), lastErr
}
