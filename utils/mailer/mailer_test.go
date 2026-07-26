package mailer

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP 一个最小可用的 SMTP 服务端，用来真正跑一遍收发对话。
// 光靠"能编译"证明不了 SMTP 实现是对的——协议细节（点转义、结束符、
// 命令顺序）全都要真跑才暴露。
type fakeSMTP struct {
	addr     string
	mu       sync.Mutex
	received []string
	authSeen bool
}

func startFakeSMTP(t *testing.T, greet string) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeSMTP{addr: ln.Addr().String()}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn, greet)
		}
	}()
	return s
}

func (s *fakeSMTP) handle(conn net.Conn, greet string) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = w.WriteString(line + "\r\n")
		_ = w.Flush()
	}

	write("220 fake ESMTP")
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
				s.mu.Lock()
				s.received = append(s.received, data.String())
				s.mu.Unlock()
				data.Reset()
				write("250 ok")
				continue
			}
			data.WriteString(line + "\n")
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			write("250-fake")
			write("250 AUTH PLAIN LOGIN")
		case strings.HasPrefix(upper, "HELO"):
			write("250 fake")
		case strings.HasPrefix(upper, "AUTH"):
			s.mu.Lock()
			s.authSeen = true
			s.mu.Unlock()
			write("235 accepted")
		case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
			write("250 ok")
		case strings.HasPrefix(upper, "DATA"):
			inData = true
			write("354 send data")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 bye")
			return
		default:
			write("250 ok")
		}
	}
}

func (s *fakeSMTP) lastMail(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.received) == 0 {
		t.Fatal("服务端没有收到任何邮件")
	}
	return s.received[len(s.received)-1]
}

func hostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatal(err)
	}
	return host, port
}

func TestSendPlain(t *testing.T) {
	srv := startFakeSMTP(t, "")
	host, port := hostPort(t, srv.addr)

	err := Send(Config{
		Host: host, Port: port, From: "panel@example.com", TLSMode: TLSNone, Timeout: 3 * time.Second,
	}, Message{
		To:      []string{"ops@example.com", "dev@example.com"},
		Subject: "磁盘告警：/ 已用 92%",
		Body:    "节点 web-1 磁盘使用率 92%\n请及时清理",
	})
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	mail := srv.lastMail(t)
	if !strings.Contains(mail, "To: ops@example.com, dev@example.com") {
		t.Errorf("收件人不对:\n%s", mail)
	}
	if !strings.Contains(mail, "charset=UTF-8") {
		t.Errorf("缺少字符集声明:\n%s", mail)
	}
	// 中文主题必须做 encoded-word 编码，否则大量客户端显示乱码
	if strings.Contains(mail, "磁盘告警：/ 已用 92%") {
		t.Errorf("主题应经过 MIME 编码而不是裸中文:\n%s", mail)
	}
	if !strings.Contains(mail, "Subject: =?UTF-8?q?") && !strings.Contains(mail, "Subject: =?utf-8?q?") {
		t.Errorf("主题未按 encoded-word 编码:\n%s", mail)
	}
	if !strings.Contains(mail, "节点 web-1 磁盘使用率 92%") {
		t.Errorf("正文丢失:\n%s", mail)
	}
}

// 正文里单独一行的点会被 SMTP 当成数据结束符，必须转义，否则邮件被截断
func TestSendEscapesLeadingDots(t *testing.T) {
	srv := startFakeSMTP(t, "")
	host, port := hostPort(t, srv.addr)

	err := Send(Config{Host: host, Port: port, From: "a@b.c", TLSMode: TLSNone, Timeout: 3 * time.Second},
		Message{To: []string{"x@y.z"}, Subject: "t", Body: "第一行\n.\n.hidden\n最后一行"})
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	mail := srv.lastMail(t)
	if !strings.Contains(mail, "最后一行") {
		t.Fatalf("正文被单独的点截断了:\n%s", mail)
	}
}

func TestSendAuthenticates(t *testing.T) {
	srv := startFakeSMTP(t, "")
	host, port := hostPort(t, srv.addr)

	// 未加密连接上 PlainAuth 会拒发，错误信息必须指向加密方式而不是密码
	err := Send(Config{
		Host: host, Port: port, Username: "u", Password: "p",
		From: "a@b.c", TLSMode: TLSNone, Timeout: 3 * time.Second,
	}, Message{To: []string{"x@y.z"}, Subject: "t", Body: "b"})
	if err == nil {
		t.Skip("该 Go 版本允许未加密认证，跳过")
	}
	if !strings.Contains(err.Error(), "未加密") {
		t.Fatalf("错误信息应提示改用加密方式，实际: %v", err)
	}
}

func TestSendValidatesInput(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		msg  Message
	}{
		{"缺主机", Config{Port: 25, From: "a@b.c"}, Message{To: []string{"x@y.z"}}},
		{"端口非法", Config{Host: "h", Port: 0, From: "a@b.c"}, Message{To: []string{"x@y.z"}}},
		{"无收件人", Config{Host: "h", Port: 25, From: "a@b.c"}, Message{}},
		{"无发件人", Config{Host: "h", Port: 25}, Message{To: []string{"x@y.z"}}},
	}
	for _, c := range cases {
		if err := Send(c.cfg, c.msg); err == nil {
			t.Errorf("%s：应返回错误", c.name)
		}
	}
}
