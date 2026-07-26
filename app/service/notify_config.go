package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/mailer"
)

// NotifyConfigView 出接口的配置。密码只回一个"是否已设置"的标记，
// 明文永远不出后端——与节点令牌同样的处理。
type NotifyConfigView struct {
	model.NotifyConfig
	HasPassword bool `json:"hasPassword"`
}

// GetNotifyConfig 读取配置（不含密码明文）
func GetNotifyConfig() NotifyConfigView {
	cfg, _ := repo.NewNotify().GetConfig()
	view := NotifyConfigView{NotifyConfig: cfg, HasPassword: strings.TrimSpace(cfg.SMTPPassword) != ""}
	view.SMTPPassword = ""
	return view
}

// SaveNotifyConfig 保存配置。
// password 为空表示"不修改"——前端拿不到明文，只能留空提交，
// 这时必须沿用库里已有的密码，否则用户改个端口就把密码清空了。
func SaveNotifyConfig(cfg model.NotifyConfig, password string) error {
	if err := validateNotifyConfig(cfg); err != nil {
		return err
	}
	existing, _ := repo.NewNotify().GetConfig()

	if strings.TrimSpace(password) == "" {
		cfg.SMTPPassword = existing.SMTPPassword
	} else {
		cipher, err := encrypt.StringEncrypt(password)
		if err != nil {
			return fmt.Errorf("加密 SMTP 密码失败: %w", err)
		}
		cfg.SMTPPassword = cipher
	}
	return repo.NewNotify().SaveConfig(&cfg)
}

func validateNotifyConfig(cfg model.NotifyConfig) error {
	if !cfg.Enabled {
		return nil // 没启用就不校验，允许先存半份配置
	}
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return errors.New("SMTP 服务器地址不能为空")
	}
	if cfg.SMTPPort <= 0 || cfg.SMTPPort > 65535 {
		return errors.New("SMTP 端口不合法")
	}
	if strings.TrimSpace(cfg.SMTPFrom) == "" && strings.TrimSpace(cfg.SMTPUser) == "" {
		return errors.New("发件人和登录账号至少填一个")
	}
	if len(parseReceivers(cfg.Receivers)) == 0 {
		return errors.New("请至少填写一个收件人")
	}
	switch cfg.SMTPTLSMode {
	case model.SMTPTLSNone, model.SMTPTLSStartTLS, model.SMTPTLSSSL:
	default:
		return errors.New("加密方式不合法")
	}
	return nil
}

// parseReceivers 收件人支持逗号、分号、空白、换行分隔——
// 用户从各种地方粘贴过来的格式五花八门，全都认下来
func parseReceivers(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t' || r == '，' || r == '；'
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || !strings.Contains(f, "@") {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// SendNotifyMail 按当前配置发一封邮件
func SendNotifyMail(cfg model.NotifyConfig, subject, body string) error {
	receivers := parseReceivers(cfg.Receivers)
	if len(receivers) == 0 {
		return errors.New("没有配置收件人")
	}
	password := ""
	if strings.TrimSpace(cfg.SMTPPassword) != "" {
		plain, err := encrypt.StringDecrypt(cfg.SMTPPassword)
		if err != nil {
			return fmt.Errorf("解密 SMTP 密码失败: %w", err)
		}
		password = plain
	}
	return mailer.Send(mailer.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUser,
		Password: password,
		From:     cfg.SMTPFrom,
		TLSMode:  cfg.SMTPTLSMode,
		Timeout:  20 * time.Second,
	}, mailer.Message{
		To:      receivers,
		Subject: subject,
		Body:    body,
	})
}

// TestNotifyMail 发一封测试邮件。
// 配置页必须有这个按钮：SMTP 配错的方式太多（端口/加密/授权码/发件人不一致），
// 没有即时反馈的话用户根本不知道配没配对，直到某天出事故才发现通知没发出来。
func TestNotifyMail(cfg model.NotifyConfig, password string) error {
	if strings.TrimSpace(password) == "" {
		existing, _ := repo.NewNotify().GetConfig()
		cfg.SMTPPassword = existing.SMTPPassword
	} else {
		cipher, err := encrypt.StringEncrypt(password)
		if err != nil {
			return fmt.Errorf("加密 SMTP 密码失败: %w", err)
		}
		cfg.SMTPPassword = cipher
	}
	if len(parseReceivers(cfg.Receivers)) == 0 {
		return errors.New("请先填写收件人")
	}
	body := fmt.Sprintf("这是一封来自 %s 的测试邮件。\n\n发送时间：%s\n\n收到它说明 SMTP 配置正确，磁盘、容器、节点离线等告警可以正常送达。\n",
		constant.AppBrand, time.Now().Format("2006-01-02 15:04:05"))
	return SendNotifyMail(cfg, fmt.Sprintf("[%s] 通知配置测试", constant.AppBrand), body)
}
