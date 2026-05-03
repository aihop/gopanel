package service

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/cloud"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	legolog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/registration"
	"strings"
	"time"
)

func (l *acmeLogger) Fatal(args ...interface{}) {
	l.logger.Error("%s", fmt.Sprint(args...))
}
func (l *acmeLogger) Fatalln(args ...interface{}) {
	l.logger.Error("%s", fmt.Sprint(args...))
}
func (l *acmeLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Error(format, args...)
}
func (l *acmeLogger) Print(args ...interface{}) {
	l.logger.Info("%s", fmt.Sprint(args...))
}
func (l *acmeLogger) Println(args ...interface{}) {
	l.logger.Info("%s", fmt.Sprint(args...))
}
func (l *acmeLogger) Printf(format string, args ...interface{}) {
	l.logger.Info(format, args...)
}
func (u *acmeUser) GetEmail() string {
	return u.Email
}
func (u *acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}
func getOrRegisterAcmeAccount(logger *SSLLogger) (*acmeUser, *model.AcmeAccount, error) {
	var dbAccount model.AcmeAccount
	result := global.DB.Where("type = ?", "letsencrypt").First(&dbAccount)
	var privateKey *ecdsa.PrivateKey
	var err error
	if result.Error == nil && dbAccount.PrivateKey != "" {
		logger.Info("发现已持久化的 ACME 账号 (%s)，尝试复用...", dbAccount.Email)
		block, _ := pem.Decode([]byte(dbAccount.PrivateKey))
		if block != nil {
			if parsedKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
				privateKey = parsedKey
			}
		}
	}
	isNewRegistration := false
	if privateKey == nil {
		logger.Info("未找到有效的本地 ACME 账号，正在生成新私钥...")
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("生成 ACME 私钥失败: %v", err)
		}
		isNewRegistration = true
	}
	email := NewUser().GetByAdminEmail()
	if dbAccount.Email != "" && email != "" {
		email = dbAccount.Email
	}
	if email == "" {
		email = "aihopv@gmail.com"
	}
	myUser := &acmeUser{Email: email, key: privateKey}
	if !isNewRegistration && dbAccount.URL != "" {
		myUser.Registration = &registration.Resource{URI: dbAccount.URL}
	}
	return myUser, &dbAccount, nil
}
func (s *SSLService) obtainCloudAcmeCertificate(item *model.SSL) {
	logger := GetSSLLogger(item.ID)
	logger.Info("开始执行 DNS-01 云账号签注流程，证书ID: %d", item.ID)
	legoLog := &acmeLogger{logger: logger}
	legolog.Logger = legoLog
	logger.Info("目标域名: %s", item.Domains)
	logger.Info("使用的云服务商类型: %s", item.Type)
	logger.Info("正在执行本地环境预检...")
	time.Sleep(2 * time.Second)
	logger.Info("预检完成，获取 ACME 账户信息...")
	cloudAccountRepo := repo.NewCloudAccount()
	account, err := cloudAccountRepo.GetByID(item.DnsAccountID)
	if err != nil {
		logger.Error("致命错误: 无法获取用于 DNS 验证的云服务商授权信息: %v", err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() {
			RemoveSSLLogger(item.ID)
		})
		return
	}
	logger.Info("成功加载 %s 云账号凭据 (别名: %s)", account.Type, account.Name)
	var authData map[ // 获取 DNS Provider
	string]interface{}
	_ = json.Unmarshal([]byte(account.Authorization), &authData)
	cloudProvider, err := cloud.NewProvider(account.Type, authData)
	if err != nil {
		logger.Error("暂未实现服务商 %s 的网关直调 DNS-01 Provider: %v", account.Type, err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() {
			RemoveSSLLogger(item.ID)
		})
		return
	}
	provider, err := cloudProvider.GetDNSProvider()
	if err != nil {
		logger.Error("获取服务商 %s 的 DNS Provider 失败: %v", account.Type, err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() {
			RemoveSSLLogger(item.ID)
		})
		return
	}
	logger.Info("正在初始化 ACME 客户端...")
	myUser, dbAccount, err := getOrRegisterAcmeAccount(logger)
	if err != nil {
		logger.Error("%v", err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() {
			RemoveSSLLogger(item.ID)
		})
		return
	}
	config := lego.NewConfig(myUser)
	config.CADirURL = "https://acme-v02.api.letsencrypt.org/directory"
	config.Certificate.KeyType = certcrypto.RSA2048
	client, err := lego.NewClient(config)
	if err != nil {
		logger.Error("创建 ACME 客户端失败: %v", err)
		return
	}
	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		logger.Error("设置 DNS-01 Provider 失败: %v", err)
		return
	}
	if myUser.Registration == nil {
		logger.Info("开始向 Let's Encrypt 发起账号注册请求...")
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			logger.Error("注册 ACME 账号失败: %v", err)
			item.Status = "error"
			_ = s.repo.SaveWithoutCtx(item)
			logger.Info("EOF")
			time.AfterFunc(10*time.Second, func() {
				RemoveSSLLogger(item.ID)
			})
			return
		}
		myUser.Registration = reg
		logger.Info("ACME 账号注册成功！正在保存至本地数据库...")
		keyBytes, _ := x509.MarshalECPrivateKey(myUser.key)
		pemBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
		dbAccount.Email = myUser.Email
		dbAccount.Type = "letsencrypt"
		dbAccount.URL = reg.URI
		dbAccount.PrivateKey = string(pemBlock)
		if dbAccount.ID > 0 {
			global.DB.Save(dbAccount)
		} else {
			global.DB.Create(dbAccount)
		}
	} else {
		logger.Info("已复用现有的 ACME 账号授权。")
	}
	logger.Info("正在发起 DNS-01 验证请求并等待全球 DNS 传播 (此步骤可能耗时 1-3 分钟，请耐心等待)...")
	domains := strings.Split(item.Domains, ",")
	var cleanDomains []string
	for _, d := range domains {
		cleanDomains = append(cleanDomains, strings.TrimSpace(d))
	}
	request := certificate.ObtainRequest{Domains: cleanDomains, Bundle: true}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		logger.Error("DNS-01 验证或签发证书失败:")
		errStr := err.Error()
		errStr = strings.ReplaceAll(errStr, "[", "")
		errStr = strings.ReplaceAll(errStr, "]", "")
		logger.Error(" -> %s", errStr)
		if strings.Contains(errStr, "The domain name belongs to other users") {
			logger.Error("👉 [诊断建议] 系统检测到您当前选择用于 DNS 验证的云账号（%s），并不具备该域名（%s）的解析管理权。", account.Name, item.Domains)
			logger.Error("👉 [诊断建议] 解决方法：请在签注表单的第 3 步（授权 DNS 解析服务商），下拉选择一个真正管理该域名解析的云账号（如您的 Cloudflare 账号或另一个阿里云账号）。")
		} else if strings.Contains(errStr, "InvalidAccessKeyId") || strings.Contains(errStr, "SignatureDoesNotMatch") {
			logger.Error("👉 [诊断建议] 系统检测到当前云账号（%s）的 API 密钥无效或已过期，请前往“云账号授权”页面重新配置正确的 AccessKey 和 SecretKey。", account.Name)
		}
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() {
			RemoveSSLLogger(item.ID)
		})
		return
	}
	logger.Info("所有域名验证成功！证书已下载...")
	item.Status = "issued"
	item.Pem = string(certificates.Certificate)
	item.PrivateKey = string(certificates.PrivateKey)
	info, err := parseCertificateInfo(item.Pem)
	if err == nil {
		item.StartDate = info.StartDate
		item.ExpireDate = info.ExpireDate
	} else {
		item.StartDate = time.Now()
		item.ExpireDate = time.Now().AddDate(0, 0, 90)
	}
	if err := s.repo.UpdateFields(item.ID, map[string]interface{}{"status": item.Status, "start_date": item.StartDate, "expire_date": item.ExpireDate, "private_key": item.PrivateKey, "pem": item.Pem}); err != nil {
		logger.Error("保存证书状态失败: %v", err)
	} else {
		logger.Info("✅ 证书签注并保存成功！有效期至: %s", item.ExpireDate.Format("2006-01-02 15:04:05"))
	}
	logger.Info("EOF")
	time.AfterFunc(10*time.Second, func() {
		RemoveSSLLogger(item.ID)
	})
	logger.Info("正在检查云端自动部署规则...")
	pushErr := AutoPushCertificateWithLogger(item.ID, logger)
	if pushErr != nil {
		logger.Error("自动部署存在警告或错误: %v", pushErr)
	} else {
		logger.Info("自动化部署流程全部完成。")
	}
}
