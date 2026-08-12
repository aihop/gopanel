package api

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func withDeliveryAttemptDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "attempts.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.AICodeDeliveryAttempt{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

// 同一会话反复交付时，每次结果都要留下来。
// 作业表按会话唯一、后一次覆盖前一次——没有这张表，49 次失败就是不可见的。
func TestDeliveryAttemptsAccumulateAcrossRetries(t *testing.T) {
	withDeliveryAttemptDB(t)
	job := &model.AICodeDeliveryJob{ID: 5, SessionID: 77, ProjectID: 3, UserID: 9}

	job.Attempt = 1
	recordCodeDeliveryAttempt(job, codeDeliveryJobFailed, codeDeliveryStageQueued, "capacity_busy",
		codeGitDeliveryResult{}, errCodeDeliveryCapacityBusy, 1500*time.Millisecond)
	job.Attempt = 2
	recordCodeDeliveryAttempt(job, codeDeliveryJobConflict, codeDeliveryStageMerging, "conflict",
		codeGitDeliveryResult{}, nil, time.Second)
	job.Attempt = 3
	recordCodeDeliveryAttempt(job, codeDeliveryJobCompleted, codeDeliveryStageCompleted, "",
		codeGitDeliveryResult{Commit: "abc1234"}, nil, 2*time.Second)

	attempts, err := loadCodeDeliveryAttempts(77, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 3 {
		t.Fatalf("三次尝试都应留档，实际 %d 条", len(attempts))
	}
	statuses := map[string]string{}
	for _, attempt := range attempts {
		statuses[attempt.Status] = attempt.FailureCode
	}
	if statuses[codeDeliveryJobFailed] != "capacity_busy" {
		t.Fatalf("失败尝试应保留失败码：%#v", statuses)
	}
	if _, ok := statuses[codeDeliveryJobConflict]; !ok {
		t.Fatalf("冲突尝试应留档：%#v", statuses)
	}
	if _, ok := statuses[codeDeliveryJobCompleted]; !ok {
		t.Fatalf("成功尝试应留档：%#v", statuses)
	}
}

// 质量门禁失败会把整段测试输出塞进错误信息，原样存下去会让这张表爆掉。
func TestDeliveryAttemptTruncatesOversizedErrorMessage(t *testing.T) {
	withDeliveryAttemptDB(t)
	job := &model.AICodeDeliveryJob{ID: 6, SessionID: 78, ProjectID: 3, UserID: 9, Attempt: 1}
	huge := errors.New(strings.Repeat("x", codeDeliveryAttemptErrorLimit*3))
	recordCodeDeliveryAttempt(job, codeDeliveryJobFailed, codeDeliveryStageQualityCheck, "quality_failed",
		codeGitDeliveryResult{}, huge, time.Second)

	attempts, err := loadCodeDeliveryAttempts(78, 20)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("应留档一条：%d, %v", len(attempts), err)
	}
	if len(attempts[0].ErrorMessage) != codeDeliveryAttemptErrorLimit {
		t.Fatalf("错误信息应被截断到 %d，实际 %d",
			codeDeliveryAttemptErrorLimit, len(attempts[0].ErrorMessage))
	}
}

// 留档是诊断信息，写不进去不能把已经完成的交付推翻。
func TestDeliveryAttemptSurvivesUnavailableDatabase(t *testing.T) {
	previous := global.DB
	global.DB = nil
	t.Cleanup(func() { global.DB = previous })
	recordCodeDeliveryAttempt(
		&model.AICodeDeliveryJob{ID: 1, SessionID: 1}, codeDeliveryJobCompleted,
		codeDeliveryStageCompleted, "", codeGitDeliveryResult{}, nil, time.Second,
	)
	attempts, err := loadCodeDeliveryAttempts(1, 20)
	if err != nil || len(attempts) != 0 {
		t.Fatalf("数据库不可用时应静默返回：%d, %v", len(attempts), err)
	}
}
