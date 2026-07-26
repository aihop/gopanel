// Package mailer 一个够用的 SMTP 发信实现。
//
// 不引第三方库，但也不能直接用 net/smtp.SendMail，原因有二：
//
//  1. net/smtp 只会 STARTTLS，不支持隐式 TLS（465 端口连上就要握手）。
//     而国内 QQ 邮箱、163、腾讯企业邮箱大量只开 465，用 SendMail 连上去
//     会一直等一个永远不来的明文问候，表现为"卡住直到超时"。
//  2. smtp.PlainAuth 在非 TLS 连接上会主动拒绝发送（防明文密码泄露），
//     报错是 "unencrypted connection"，跟"密码错"完全不是一回事，
//     必须把这种情况单独提示出来，否则用户会一直去改密码。
package mailer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const (
	TLSNone     = "none"
	TLSStartTLS = "starttls"
	TLSSSL      = "ssl"
)

// Config 发信配置
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLSMode  string
	Timeout  time.Duration
}

// Message 一封邮件
type Message struct {
	To      []string
	Subject string
	Body    string // 纯文本。告警邮件不需要 HTML，纯文本在任何客户端都不会变形
}

// Send 发送邮件。任何一步失败都带上足够定位问题的上下文——
// SMTP 配置错误的排查成本很高，错误信息含糊等于让用户瞎猜。
func Send(cfg Config, msg Message) error {
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("SMTP 服务器地址为空")
	}
	if cfg.Port <= 0 {
		return errors.New("SMTP 端口无效")
	}
	if len(msg.To) == 0 {
		return errors.New("收件人为空")
	}
	from := strings.TrimSpace(cfg.From)
	if from == "" {
		from = cfg.Username
	}
	if from == "" {
		return errors.New("发件人为空")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := dial(cfg, addr, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	if cfg.TLSMode == TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("服务器 %s 不支持 STARTTLS，请改用 SSL(465) 或不加密", addr)
		}
		if err := client.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
			return fmt.Errorf("STARTTLS 握手失败: %w", err)
		}
	}

	if cfg.Username != "" {
		if err := auth(client, cfg); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人 %s 失败（多数邮箱要求发件人与登录账号一致）: %w", from, err)
	}
	for _, to := range msg.To {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人 %s 失败: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("进入数据发送阶段失败: %w", err)
	}
	if _, err := w.Write(build(from, msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("提交邮件失败: %w", err)
	}
	return client.Quit()
}

func dial(cfg Config, addr string, timeout time.Duration) (*smtp.Client, error) {
	if cfg.TLSMode == TLSSSL {
		// 隐式 TLS：连上就握手，不能先发明文
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, &tls.Config{ServerName: cfg.Host})
		if err != nil {
			return nil, fmt.Errorf("连接 %s 失败（465 端口需要 SSL 模式，端口与加密方式不匹配会一直卡到超时）: %w", addr, err)
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP 握手失败: %w", err)
		}
		return client, nil
	}

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP 握手失败（若服务器是 465 端口请选 SSL 模式）: %w", err)
	}
	return client, nil
}

func auth(client *smtp.Client, cfg Config) error {
	a := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(a); err != nil {
		// PlainAuth 在未加密连接上会拒发，这个错和"密码错误"要分开提示
		if strings.Contains(err.Error(), "unencrypted") {
			return fmt.Errorf("当前连接未加密，SMTP 认证被拒绝，请把加密方式改为 STARTTLS(587) 或 SSL(465): %w", err)
		}
		// 很多邮箱要求用「授权码」而不是登录密码，这一点必须写在错误里
		return fmt.Errorf("SMTP 认证失败（QQ/163 等邮箱需使用授权码而非登录密码）: %w", err)
	}
	return nil
}

func build(from string, msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	// 主题里有中文，不做 encoded-word 编码的话大量客户端会显示乱码
	b.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", msg.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	// 正文里以点开头的行必须转义，否则单独一行的 "." 会被当成数据结束符
	b.WriteString(escapeDots(msg.Body))
	b.WriteString("\r\n")
	return []byte(b.String())
}

func escapeDots(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, ".") {
			lines[i] = "." + line
		}
	}
	return strings.Join(lines, "\r\n")
}
