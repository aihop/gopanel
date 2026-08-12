package api

import (
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

// 单条错误信息的留档上限。交付失败时质量门禁会把整段测试输出塞进来，
// 原样存下去会让这张表迅速膨胀，而定位问题只需要开头那段。
const codeDeliveryAttemptErrorLimit = 2000

// warnCodeDelivery 是 nil 安全的告警日志。
//
// global.LOG 只在面板正常启动时才装配，测试和早期初始化阶段是 nil。
// 直接调用会让「记一条降级告警」这种本该无害的动作反而 panic ——
// 诊断信息的写入路径不该比它诊断的东西更容易崩。
func warnCodeDelivery(format string, args ...any) {
	if global.LOG == nil {
		return
	}
	global.LOG.Warnf(format, args...)
}

// recordCodeDeliveryAttempt 把一次交付尝试的结果追加留档。
//
// 作业表按会话唯一，重新交付会覆盖上一次的结果；没有这张表的话，
// 失败在面板里就是彻底不可见的——用户只能靠「感觉总出问题」，
// 而任何针对失败率的改进也没法验证效果。
//
// 留档失败只记日志：交付本身已经结束了，不能因为写不进一条统计
// 就把用户已经拿到的结果推翻。
func recordCodeDeliveryAttempt(
	job *model.AICodeDeliveryJob,
	status, stage, failureCode string,
	result codeGitDeliveryResult,
	runErr error,
	duration time.Duration,
) {
	if global.DB == nil || job == nil || job.ID == 0 {
		return
	}
	errorMessage := ""
	if runErr != nil {
		errorMessage = runErr.Error()
	} else if result.ErrorMessage != "" {
		errorMessage = result.ErrorMessage
	}
	if len(errorMessage) > codeDeliveryAttemptErrorLimit {
		errorMessage = errorMessage[:codeDeliveryAttemptErrorLimit]
	}
	attempt := model.AICodeDeliveryAttempt{
		SessionID: job.SessionID, JobID: job.ID, TaskID: job.TaskID,
		ProjectID: job.ProjectID, UserID: job.UserID, Attempt: job.Attempt,
		Status: status, Stage: stage, FailureCode: failureCode,
		ResultCommit: result.Commit, ErrorMessage: errorMessage,
		DurationMS: duration.Milliseconds(),
	}
	if err := global.DB.Create(&attempt).Error; err != nil {
		warnCodeDelivery("Record Code delivery attempt for session %d failed: %v", job.SessionID, err)
	}
}

// loadCodeDeliveryAttempts 取某个会话的交付尝试历史，最近的在前。
func loadCodeDeliveryAttempts(sessionID uint, limit int) ([]model.AICodeDeliveryAttempt, error) {
	if global.DB == nil || sessionID == 0 {
		return nil, nil
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	var attempts []model.AICodeDeliveryAttempt
	err := global.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").Limit(limit).Find(&attempts).Error
	return attempts, err
}
