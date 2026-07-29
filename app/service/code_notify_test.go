package service

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestBuildCodeNotifyMessage(t *testing.T) {
	session := &model.AIDevSession{ID: 12, Title: "登录修复", AgentName: "codex", WorkDir: "/srv/app"}
	task := &model.AITask{Title: "修复登录接口"}
	subject, body := buildCodeNotifyMessage(session, task, CodeNotifyApproval, "需要执行数据库迁移")
	for _, expected := range []string{"等待确认", "修复登录接口"} {
		if !strings.Contains(subject, expected) {
			t.Fatalf("subject %q missing %q", subject, expected)
		}
	}
	for _, expected := range []string{"codex", "/srv/app", "需要执行数据库迁移", "会话 ID：12"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body missing %q: %s", expected, body)
		}
	}
}
