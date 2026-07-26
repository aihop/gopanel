package service

import (
	"testing"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
)

func TestShouldRemindRespectsSilence(t *testing.T) {
	now := time.Now()

	// 静默期内不重复提醒——这是"磁盘满每分钟一封信"的唯一防线
	cfg := model.NotifyConfig{SilenceHours: 6}
	if shouldRemind(cfg, now.Add(-5*time.Hour), now) {
		t.Error("静默期未过不应重复提醒")
	}
	if !shouldRemind(cfg, now.Add(-7*time.Hour), now) {
		t.Error("超过静默期应再次提醒")
	}
	// 从未通知过时应当提醒
	if !shouldRemind(cfg, time.Time{}, now) {
		t.Error("从未通知过应提醒")
	}
	// 0 表示只发一次，不再重复
	if shouldRemind(model.NotifyConfig{SilenceHours: 0}, now.Add(-100*time.Hour), now) {
		t.Error("静默期为 0 表示不重复提醒")
	}
}

func TestDebounceTimesFloor(t *testing.T) {
	if got := debounceTimes(model.NotifyConfig{DebounceTimes: 0}); got != 1 {
		t.Errorf("去抖次数至少为 1，got %d", got)
	}
	if got := debounceTimes(model.NotifyConfig{DebounceTimes: 3}); got != 3 {
		t.Errorf("got %d", got)
	}
}

func TestAlertTypeEnabled(t *testing.T) {
	cfg := model.NotifyConfig{EnableDisk: true, EnableContainer: false, EnableOffline: true, EnableCert: false}
	cases := map[string]bool{
		"disk": true, "container": false, "offline": true, "unauthorized": true,
		"cert": false, "unknown-type": false,
	}
	for typ, want := range cases {
		if got := alertTypeEnabled(cfg, typ); got != want {
			t.Errorf("%s = %v, want %v", typ, got, want)
		}
	}
}

func TestParseReceivers(t *testing.T) {
	got := parseReceivers("a@x.com, b@x.com;c@x.com\n d@x.com，e@x.com  a@x.com  垃圾内容")
	want := []string{"a@x.com", "b@x.com", "c@x.com", "d@x.com", "e@x.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(parseReceivers("   ")) != 0 {
		t.Error("空输入应返回空列表")
	}
}

func TestBuildAlertSubject(t *testing.T) {
	one := []model.AlertEvent{{SourceName: "web-1", Type: "disk"}}
	two := []model.AlertEvent{{SourceName: "web-1", Type: "disk"}, {SourceName: "db-1", Type: "container"}}

	// 单条告警把主体写进标题，手机通知栏一眼能看清
	if got := buildAlertSubject(one, nil); got == "" || !contains(got, "web-1") || !contains(got, "磁盘告警") {
		t.Errorf("单条告警标题应含主体和类型: %s", got)
	}
	if got := buildAlertSubject(two, nil); !contains(got, "2 项告警") {
		t.Errorf("多条应聚合计数: %s", got)
	}
	if got := buildAlertSubject(nil, one); !contains(got, "已恢复") {
		t.Errorf("恢复通知标题不对: %s", got)
	}
	if got := buildAlertSubject(one, one); !contains(got, "告警") || !contains(got, "恢复") {
		t.Errorf("同时有告警和恢复时都要体现: %s", got)
	}
}

func TestBuildAlertBodyAggregates(t *testing.T) {
	now := time.Now()
	fired := []model.AlertEvent{
		{SourceName: "web-1", Type: "disk", Level: "warn", Detail: "/ 使用率 86.0%", FirstSeenAt: now.Add(-time.Hour)},
		{SourceName: "db-1", Type: "disk", Level: "danger", Detail: "/data 使用率 95.0%", FirstSeenAt: now.Add(-2 * time.Hour)},
	}
	resolved := []model.AlertEvent{
		{SourceName: "cache-1", Type: "container", FirstSeenAt: now.Add(-3 * time.Hour), ResolvedAt: now},
	}
	body := buildAlertBody(fired, resolved)

	// 10 个节点同时出问题要发 1 封列 10 行，而不是 10 封
	for _, want := range []string{"web-1", "db-1", "cache-1", "/data 使用率 95.0%", "【告警】2 项", "【恢复】1 项"} {
		if !contains(body, want) {
			t.Errorf("正文缺少 %q:\n%s", want, body)
		}
	}
	// danger 排在 warn 前面
	if indexOf(body, "db-1") > indexOf(body, "web-1") {
		t.Errorf("严重级别应排在前面:\n%s", body)
	}
}

func TestDescribeWarning(t *testing.T) {
	target := alertTarget{Node: model.Node{Summary: model.NodeSummary{DiskMaxPath: "/data"}}}
	cases := []struct {
		w    dto.NodeWarning
		want string
	}{
		{dto.NodeWarning{Type: "disk", Value: 93.5}, "/data 使用率 93.5%"},
		{dto.NodeWarning{Type: "container", Value: 3}, "3 个容器处于异常状态（dead/restarting/paused）"},
		{dto.NodeWarning{Type: "offline", Value: 2.5}, "节点已离线，最后在线于 2.5 小时前"},
		{dto.NodeWarning{Type: "cert", Value: -3}, "已有证书过期 3 天"},
	}
	for _, c := range cases {
		if got := describeWarning(target, c.w); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestValidateNotifyConfig(t *testing.T) {
	ok := model.NotifyConfig{
		Enabled: true, SMTPHost: "smtp.x.com", SMTPPort: 587,
		SMTPUser: "u@x.com", Receivers: "ops@x.com", SMTPTLSMode: model.SMTPTLSStartTLS,
	}
	if err := validateNotifyConfig(ok); err != nil {
		t.Fatalf("合法配置不应报错: %v", err)
	}
	// 未启用时不校验，允许先存半份配置
	if err := validateNotifyConfig(model.NotifyConfig{Enabled: false}); err != nil {
		t.Fatalf("未启用不应校验: %v", err)
	}

	bad := []model.NotifyConfig{
		{Enabled: true, SMTPPort: 587, SMTPUser: "u@x.com", Receivers: "a@b.c", SMTPTLSMode: "starttls"},        // 缺 host
		{Enabled: true, SMTPHost: "h", SMTPPort: 0, SMTPUser: "u", Receivers: "a@b.c", SMTPTLSMode: "starttls"}, // 端口非法
		{Enabled: true, SMTPHost: "h", SMTPPort: 587, SMTPUser: "u", Receivers: "", SMTPTLSMode: "starttls"},    // 无收件人
		{Enabled: true, SMTPHost: "h", SMTPPort: 587, SMTPUser: "u", Receivers: "a@b.c", SMTPTLSMode: "weird"},  // 加密方式非法
		{Enabled: true, SMTPHost: "h", SMTPPort: 587, Receivers: "a@b.c", SMTPTLSMode: "starttls"},              // 无发件人也无账号
	}
	for i, c := range bad {
		if err := validateNotifyConfig(c); err == nil {
			t.Errorf("第 %d 组非法配置应报错", i)
		}
	}
}

func contains(s, sub string) bool { return indexOf(s, sub) >= 0 }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
