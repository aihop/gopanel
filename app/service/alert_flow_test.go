package service

import (
	"bufio"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/zlog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 收信用的假 SMTP 服务端，用来验证"到底发没发、发了几封"
type mailCatcher struct {
	addr string
	mu   sync.Mutex
	got  []string
}

func newMailCatcher(t *testing.T) *mailCatcher {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &mailCatcher{addr: ln.Addr().String()}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go m.serve(conn)
		}
	}()
	return m
}

func (m *mailCatcher) serve(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	reply := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
	reply("220 catcher")
	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				m.mu.Lock()
				m.got = append(m.got, data.String())
				m.mu.Unlock()
				data.Reset()
				reply("250 ok")
				continue
			}
			data.WriteString(line + "\n")
			continue
		}
		up := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(up, "EHLO"):
			reply("250 catcher")
		case strings.HasPrefix(up, "DATA"):
			inData = true
			reply("354 go")
		case strings.HasPrefix(up, "QUIT"):
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func (m *mailCatcher) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.got)
}

func (m *mailCatcher) last() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.got) == 0 {
		return ""
	}
	return m.got[len(m.got)-1]
}

func setupAlertDB(t *testing.T) {
	t.Helper()
	global.LOG = zlog.New(io.Discard, zlog.ErrorLevel, &zlog.TextFormatter{})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.NotifyConfig{}, &model.AlertEvent{}, &model.Node{}); err != nil {
		t.Fatal(err)
	}
	old := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = old })
}

func saveTestConfig(t *testing.T, catcher *mailCatcher, mutate func(*model.NotifyConfig)) {
	t.Helper()
	host, portStr, _ := net.SplitHostPort(catcher.addr)
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	cfg := model.NotifyConfig{
		Enabled: true, SMTPHost: host, SMTPPort: port,
		SMTPFrom: "panel@test.local", SMTPTLSMode: model.SMTPTLSNone,
		Receivers: "ops@test.local", DebounceTimes: 2, SilenceHours: 6,
		NotifyResolved: true, EnableDisk: true, EnableContainer: true, EnableOffline: true,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	if err := repo.NewNotify().SaveConfig(&cfg); err != nil {
		t.Fatal(err)
	}
}

// 造一个磁盘 95% 的离线节点之外的"在线但磁盘满"节点
func seedFullDiskNode(t *testing.T, name string, percent float64) *model.Node {
	t.Helper()
	node := &model.Node{
		Name: name, Addr: "https://127.0.0.1:5470", Status: NodeStatusOnline,
		Summary: model.NodeSummary{DiskMaxPercent: percent, DiskMaxPath: "/data"},
	}
	if err := repo.NewNode().Create(node); err != nil {
		t.Fatal(err)
	}
	return node
}

// 全链路：去抖 → 触发发信 → 静默期内不重发 → 恢复发信
func TestAlertLifecycle(t *testing.T) {
	setupAlertDB(t)
	catcher := newMailCatcher(t)
	saveTestConfig(t, catcher, nil)
	node := seedFullDiskNode(t, "db-1", 95)

	// 第一轮：命中但未达去抖次数（2 次），不应发信
	EvaluateAlerts()
	if catcher.count() != 0 {
		t.Fatalf("第一次命中不应发信（去抖），已发 %d 封", catcher.count())
	}

	// 第二轮：达到去抖次数，发一封
	EvaluateAlerts()
	if catcher.count() != 1 {
		t.Fatalf("达到去抖次数应发 1 封，实际 %d 封", catcher.count())
	}
	mail := catcher.last()
	if !strings.Contains(mail, "db-1") {
		t.Errorf("邮件应包含节点名:\n%s", mail)
	}

	// 第三、四轮：仍在告警中且未过静默期，一封都不能再发
	EvaluateAlerts()
	EvaluateAlerts()
	if catcher.count() != 1 {
		t.Fatalf("静默期内不应重复发信，实际共 %d 封（这正是「磁盘满每分钟一封」的坑）", catcher.count())
	}

	// 磁盘恢复正常
	node.Summary.DiskMaxPercent = 40
	if err := repo.NewNode().UpdateSummary(node.ID, model.Node{
		Status: NodeStatusOnline, Version: "1", LastSeenAt: time.Now(), Summary: node.Summary,
	}); err != nil {
		t.Fatal(err)
	}
	EvaluateAlerts()
	if catcher.count() != 2 {
		t.Fatalf("恢复应发 1 封通知，实际共 %d 封", catcher.count())
	}
	if !strings.Contains(catcher.last(), "恢复") {
		t.Errorf("恢复邮件内容不对:\n%s", catcher.last())
	}

	// 恢复之后不应再发任何东西
	EvaluateAlerts()
	EvaluateAlerts()
	if catcher.count() != 2 {
		t.Fatalf("恢复后不应继续发信，实际共 %d 封", catcher.count())
	}
}

// 多个节点同时出问题必须聚合成一封
func TestAlertAggregatesMultipleNodes(t *testing.T) {
	setupAlertDB(t)
	catcher := newMailCatcher(t)
	saveTestConfig(t, catcher, func(c *model.NotifyConfig) { c.DebounceTimes = 1 })
	seedFullDiskNode(t, "web-1", 92)
	seedFullDiskNode(t, "web-2", 96)
	seedFullDiskNode(t, "web-3", 91)

	EvaluateAlerts()
	if catcher.count() != 1 {
		t.Fatalf("3 个节点同时告警应聚合成 1 封，实际 %d 封", catcher.count())
	}
	mail := catcher.last()
	for _, name := range []string{"web-1", "web-2", "web-3"} {
		if !strings.Contains(mail, name) {
			t.Errorf("聚合邮件缺少 %s:\n%s", name, mail)
		}
	}
}

// 关掉的事件类型不应触发
func TestAlertRespectsTypeToggle(t *testing.T) {
	setupAlertDB(t)
	catcher := newMailCatcher(t)
	saveTestConfig(t, catcher, func(c *model.NotifyConfig) {
		c.DebounceTimes = 1
		c.EnableDisk = false
	})
	seedFullDiskNode(t, "db-1", 99)

	EvaluateAlerts()
	if catcher.count() != 0 {
		t.Fatalf("磁盘告警已关闭，不应发信，实际 %d 封", catcher.count())
	}
}

// 未启用通知时整个引擎不动
func TestAlertSkippedWhenDisabled(t *testing.T) {
	setupAlertDB(t)
	catcher := newMailCatcher(t)
	saveTestConfig(t, catcher, func(c *model.NotifyConfig) { c.Enabled = false; c.DebounceTimes = 1 })
	seedFullDiskNode(t, "db-1", 99)

	EvaluateAlerts()
	if catcher.count() != 0 {
		t.Fatalf("未启用不应发信，实际 %d 封", catcher.count())
	}
	events, _ := repo.NewNotify().ActiveEvents()
	if len(events) != 0 {
		t.Fatalf("未启用时不应写入事件，实际 %d 条", len(events))
	}
}

// 状态必须落库：重启后（重新读库）不能把仍在持续的告警重发一遍
func TestAlertStateSurvivesRestart(t *testing.T) {
	setupAlertDB(t)
	catcher := newMailCatcher(t)
	saveTestConfig(t, catcher, func(c *model.NotifyConfig) { c.DebounceTimes = 1 })
	seedFullDiskNode(t, "db-1", 97)

	EvaluateAlerts()
	if catcher.count() != 1 {
		t.Fatalf("应发 1 封，实际 %d", catcher.count())
	}
	// 模拟面板重启：内存态全部丢失，只剩数据库
	EvaluateAlerts()
	if catcher.count() != 1 {
		t.Fatalf("重启后不应重发仍在持续的告警，实际共 %d 封", catcher.count())
	}
	events, _ := repo.NewNotify().ActiveEvents()
	if len(events) == 0 {
		t.Fatal("活动事件应已落库")
	}
	if events[0].Status != model.AlertStatusFiring {
		t.Fatalf("事件状态应为 firing，实际 %s", events[0].Status)
	}
}

// 回归：布尔开关关闭、静默期设 0 必须真的存进去。
// GORM 会跳过「零值 + default 标签」的字段让数据库填默认值，
// 结果是用户关掉磁盘告警、把静默期设为 0，保存后又变回开启和 6 小时。
func TestNotifyConfigPersistsFalsyValues(t *testing.T) {
	setupAlertDB(t)
	cfg := model.NotifyConfig{
		Enabled: true, SMTPHost: "h", SMTPPort: 25, SMTPFrom: "a@b.c",
		SMTPTLSMode: model.SMTPTLSNone, Receivers: "x@y.z",
		DebounceTimes: 1, SilenceHours: 0,
		NotifyResolved: false, EnableDisk: false, EnableContainer: false, EnableOffline: false, EnableCert: false,
	}
	if err := repo.NewNotify().SaveConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	got, err := repo.NewNotify().GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.EnableDisk || got.EnableContainer || got.EnableOffline || got.NotifyResolved {
		t.Errorf("关闭的开关必须保持关闭: %+v", got)
	}
	if got.SilenceHours != 0 {
		t.Errorf("静默期 0（只发一次）被改写成了 %d", got.SilenceHours)
	}

	// 再存一次（走 Save 而不是 Create 分支），同样不能被改写
	got.EnableDisk = false
	if err := repo.NewNotify().SaveConfig(&got); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.NewNotify().GetConfig()
	if again.EnableDisk || again.SilenceHours != 0 {
		t.Errorf("二次保存后又被默认值改写: %+v", again)
	}
}
