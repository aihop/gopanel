package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	legolog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/registration"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	caddyinit "github.com/aihop/gopanel/init/caddy"
	"github.com/aihop/gopanel/pkg/aliyun"
	"github.com/aihop/gopanel/pkg/cloud"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
)

type acmeLogger struct {
	logger *SSLLogger
}

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

type acmeUser struct {
	Email        string
	Registration *registration.Resource
	key          *ecdsa.PrivateKey
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
	// 简单起见，这里直接查询第一条作为全局 Let's Encrypt 账号
	result := global.DB.Where("type = ?", "letsencrypt").First(&dbAccount)

	var privateKey *ecdsa.PrivateKey
	var err error

	if result.Error == nil && dbAccount.PrivateKey != "" {
		// 已有账号，从数据库加载私钥
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
	myUser := &acmeUser{
		Email: email,
		key:   privateKey,
	}

	if !isNewRegistration && dbAccount.URL != "" {
		myUser.Registration = &registration.Resource{URI: dbAccount.URL}
	}

	return myUser, &dbAccount, nil
}

func NewSSL() *SSLService {
	return &SSLService{
		repo: repo.NewSSL(),
	}
}

type SSLService struct {
	repo *repo.SSLRepo
}

func (s *SSLService) Create(req *request.SSLCreate) (*model.SSL, error) {
	domains, primary, _ := normalizeDomains(req.PrimaryDomain, req.OtherDomains)
	if primary == "" {
		return nil, errors.New("主域名不能为空")
	}

	item := &model.SSL{
		PrimaryDomain:  primary,
		Domains:        strings.Join(domains, ","),
		Type:           req.Type,
		Description:    strings.TrimSpace(req.Description),
		KeyType:        normalizeKeyType(req.KeyType),
		AcmeAccountID:  req.AcmeAccountID,
		CloudAccountID: req.CloudAccountID,
		DnsAccountID:   req.DnsAccountID,
		Status:         "issued",
		AutoRenew:      false,
	}

	if item.Type == "" {
		item.Type = "upload"
	}

	if item.Type == "upload" {
		if strings.TrimSpace(req.Pem) == "" || strings.TrimSpace(req.PrivateKey) == "" {
			return nil, errors.New("请填写完整证书内容和私钥")
		}
		item.Pem = strings.TrimSpace(req.Pem)
		item.PrivateKey = strings.TrimSpace(req.PrivateKey)
		info, err := parseCertificateInfo(item.Pem)
		if err != nil {
			return nil, err
		}
		if len(info.Domains) > 0 {
			domains = info.Domains
			item.PrimaryDomain = domains[0]
			item.Domains = strings.Join(domains, ",")
		}
		item.StartDate = info.StartDate
		item.ExpireDate = info.ExpireDate
		item.Organization = info.IssuerName

		if item.Organization == "" {
			item.Organization = "GoPanel"
		}
		item.Provider = "custom"
	} else if item.Type == "dns" {
		item.Status = "pending"
		item.Provider = "acme-dns"
		item.Organization = "Let's Encrypt" // 默认占位，后续由签发流程更新
		item.AutoRenew = true

		// 查询 DNS 账号以标记来源
		if req.DnsAccountID > 0 {
			cloudAccountRepo := repo.NewCloudAccount()
			if account, err := cloudAccountRepo.GetByID(req.DnsAccountID); err == nil {
				item.Type = "dns-" + account.Type // 例如: dns-aliyun
			}
		} else if req.CloudAccountID > 0 {
			cloudAccountRepo := repo.NewCloudAccount()
			if account, err := cloudAccountRepo.GetByID(req.CloudAccountID); err == nil {
				item.Type = "dns-" + account.Type // 兼容仅选择拉取域名的账号
			}
		}

		if err := s.repo.Create(context.Background(), item); err != nil {
			return nil, err
		}

		// 获取最新实例并传递指针，确保异步任务内的更新能作用在最新数据上
		refreshedItem, _ := s.repo.GetFirst(s.repo.WithID(item.ID))
		go s.obtainCloudAcmeCertificate(&refreshedItem)
		return item, nil
	}

	if item.Type == "upload" {
		if err := s.repo.Create(context.Background(), item); err != nil {
			return nil, err
		}

		dir, err := s.persistCertificateFiles(item)
		if err != nil {
			return nil, err
		}
		item.Dir = dir
		return item, s.repo.SaveWithoutCtx(item)
	}

	return item, nil
}

func (s *SSLService) Renew(id uint) error {
	item, err := s.repo.GetFirst(s.repo.WithID(id))
	if err != nil {
		return err
	}

	if item.Type == "upload" {
		return errors.New("手动上传的证书不支持自动签发")
	}

	if item.Type == "caddy" {
		// 这里暂不实现 caddy 域名的主动重签逻辑，它由 caddy 接管
		return errors.New("Caddy 管理的证书不支持手动重签")
	}

	if strings.HasPrefix(item.Type, "dns-") {
		item.Status = "pending"
		if err := s.repo.SaveWithoutCtx(&item); err != nil {
			return err
		}
		// 获取最新实例并传递指针，确保异步任务内的更新能作用在最新数据上
		refreshedItem, _ := s.repo.GetFirst(s.repo.WithID(item.ID))
		go s.obtainCloudAcmeCertificate(&refreshedItem)
		return nil
	}

	return errors.New("不支持的证书类型")
}

func (s *SSLService) obtainCloudAcmeCertificate(item *model.SSL) {
	logger := GetSSLLogger(item.ID)
	logger.Info("开始执行 DNS-01 云账号签注流程，证书ID: %d", item.ID)

	// 接管 lego 底层日志
	legoLog := &acmeLogger{logger: logger}
	legolog.Logger = legoLog
	logger.Info("目标域名: %s", item.Domains)
	logger.Info("使用的云服务商类型: %s", item.Type)

	logger.Info("正在执行本地环境预检...")
	// TODO: 这里填写真实的 ACME 流程
	// 模拟耗时过程
	time.Sleep(2 * time.Second)
	logger.Info("预检完成，获取 ACME 账户信息...")

	cloudAccountRepo := repo.NewCloudAccount()
	account, err := cloudAccountRepo.GetByID(item.DnsAccountID)
	if err != nil {
		logger.Error("致命错误: 无法获取用于 DNS 验证的云服务商授权信息: %v", err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
		return
	}
	logger.Info("成功加载 %s 云账号凭据 (别名: %s)", account.Type, account.Name)

	// 获取 DNS Provider
	var authData map[string]interface{}
	_ = json.Unmarshal([]byte(account.Authorization), &authData)

	cloudProvider, err := cloud.NewProvider(account.Type, authData)
	if err != nil {
		logger.Error("暂未实现服务商 %s 的网关直调 DNS-01 Provider: %v", account.Type, err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
		return
	}

	provider, err := cloudProvider.GetDNSProvider()
	if err != nil {
		logger.Error("获取服务商 %s 的 DNS Provider 失败: %v", account.Type, err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
		return
	}

	logger.Info("正在初始化 ACME 客户端...")
	myUser, dbAccount, err := getOrRegisterAcmeAccount(logger)
	if err != nil {
		logger.Error("%v", err)
		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
		return
	}

	config := lego.NewConfig(myUser)
	// Let's Encrypt 生产环境 URL
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

	// 注册流程：如果 myUser.Registration 为空，说明是新号或未注册
	if myUser.Registration == nil {
		logger.Info("开始向 Let's Encrypt 发起账号注册请求...")
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			logger.Error("注册 ACME 账号失败: %v", err)
			item.Status = "error"
			_ = s.repo.SaveWithoutCtx(item)
			logger.Info("EOF")
			time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
			return
		}
		myUser.Registration = reg
		logger.Info("ACME 账号注册成功！正在保存至本地数据库...")

		// 将新生成的私钥序列化并存入数据库
		keyBytes, _ := x509.MarshalECPrivateKey(myUser.key)
		pemBlock := pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		})

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

	request := certificate.ObtainRequest{
		Domains: cleanDomains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		logger.Error("DNS-01 验证或签发证书失败:")
		// lego 返回的 err 通常是复合错误，包含了具体的 DNS 验证失败原因
		errStr := err.Error()
		errStr = strings.ReplaceAll(errStr, "[", "")
		errStr = strings.ReplaceAll(errStr, "]", "")
		logger.Error(" -> %s", errStr)

		// 翻译并拦截常见的云厂商特定报错，给用户更清晰的中文指引
		if strings.Contains(errStr, "The domain name belongs to other users") {
			logger.Error("👉 [诊断建议] 系统检测到您当前选择用于 DNS 验证的云账号（%s），并不具备该域名（%s）的解析管理权。", account.Name, item.Domains)
			logger.Error("👉 [诊断建议] 解决方法：请在签注表单的第 3 步（授权 DNS 解析服务商），下拉选择一个真正管理该域名解析的云账号（如您的 Cloudflare 账号或另一个阿里云账号）。")
		} else if strings.Contains(errStr, "InvalidAccessKeyId") || strings.Contains(errStr, "SignatureDoesNotMatch") {
			logger.Error("👉 [诊断建议] 系统检测到当前云账号（%s）的 API 密钥无效或已过期，请前往“云账号授权”页面重新配置正确的 AccessKey 和 SecretKey。", account.Name)
		}

		item.Status = "error"
		_ = s.repo.SaveWithoutCtx(item)
		logger.Info("EOF")
		time.AfterFunc(10*time.Second, func() { RemoveSSLLogger(item.ID) })
		return
	}

	logger.Info("所有域名验证成功！证书已下载...")

	// 保存真实的证书内容
	item.Status = "issued"
	item.Pem = string(certificates.Certificate)
	item.PrivateKey = string(certificates.PrivateKey)

	// 解析证书有效期
	info, err := parseCertificateInfo(item.Pem)
	if err == nil {
		item.StartDate = info.StartDate
		item.ExpireDate = info.ExpireDate
	} else {
		// fallback
		item.StartDate = time.Now()
		item.ExpireDate = time.Now().AddDate(0, 0, 90)
	}

	// 在后台协程中，使用 DB.Model().Updates() 强制更新指定字段，绕过 GORM 零值判断或状态覆盖
	if err := s.repo.UpdateFields(item.ID, map[string]interface{}{
		"status":      item.Status,
		"start_date":  item.StartDate,
		"expire_date": item.ExpireDate,
		"private_key": item.PrivateKey,
		"pem":         item.Pem,
	}); err != nil {
		logger.Error("保存证书状态失败: %v", err)
	} else {
		logger.Info("✅ 证书签注并保存成功！有效期至: %s", item.ExpireDate.Format("2006-01-02 15:04:05"))
	}

	logger.Info("EOF") // 结束标记

	// 稍后清理日志通道
	time.AfterFunc(10*time.Second, func() {
		RemoveSSLLogger(item.ID)
	})

	// 触发自动分发部署
	logger.Info("正在检查云端自动部署规则...")
	pushErr := AutoPushCertificateWithLogger(item.ID, logger)
	if pushErr != nil {
		logger.Error("自动部署存在警告或错误: %v", pushErr)
	} else {
		logger.Info("自动化部署流程全部完成。")
	}
}

func (s *SSLService) Get(id uint) (res *model.SSL, err error) {
	item, err := s.repo.GetFirst(s.repo.WithID(id))
	if err != nil {
		return nil, err
	}
	res = &item
	if err = s.attachWebsiteRelations([]*model.SSL{res}); err != nil {
		return nil, err
	}
	return
}

func (s *SSLService) GetByWebsiteID(websiteID uint) (res *model.SSL, err error) {
	website, err := repo.NewWebsite().GetFirst(repo.NewWebsite().WithID(websiteID))
	if err != nil {
		return nil, err
	}
	items, err := s.repo.GetBy(s.repo.WithDomain(website.PrimaryDomain))
	if err == nil && len(items) > 0 {
		res = &items[0]
		return
	}
	return s.Obtain(websiteID)
}

func (s *SSLService) Update(req *request.SSLUpdate) error {
	item, err := s.repo.GetFirst(s.repo.WithID(req.ID))
	if err != nil {
		return err
	}
	item.Description = strings.TrimSpace(req.Description)
	item.AutoRenew = req.AutoRenew
	return s.repo.SaveWithoutCtx(&item)
}

func (s *SSLService) Delete(id uint) error {
	item, err := s.repo.GetFirst(s.repo.WithID(id))
	if err != nil {
		return err
	}
	if err = s.repo.DeleteBy(context.Background(), s.repo.WithID(id)); err != nil {
		return err
	}
	if item.Type == "upload" && item.Dir != "" {
		_ = os.RemoveAll(item.Dir)
	}
	return nil
}

func (s *SSLService) Apply(req *request.SSLApply) error {
	website, err := repo.NewWebsite().GetFirst(repo.NewWebsite().WithID(req.WebsiteID))
	if err != nil {
		return err
	}
	item, err := s.repo.GetFirst(s.repo.WithID(req.SSLID))
	if err != nil {
		return err
	}
	if item.Type == "caddy" {
		return errors.New("Caddy 自动 HTTPS 无需手动应用证书")
	}
	certPath, keyPath, err := s.ensureCertificateFiles(&item)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(caddyinit.CaddyFilePath())
	if err != nil {
		return err
	}
	domainValues := make([]string, 0, 1+len(website.Domains))
	domainValues = append(domainValues, website.PrimaryDomain)
	for _, domain := range website.Domains {
		domainValues = append(domainValues, domain.Domain)
	}
	domains, _, _ := normalizeDomains(domainValues[0], strings.Join(domainValues[1:], ","))
	if len(domains) == 0 {
		return errors.New("网站未配置域名")
	}
	next := string(content)
	for _, domain := range domains {
		next, err = upsertTLSDirective(next, domain, certPath, keyPath)
		if err != nil {
			return err
		}
	}
	return NewCaddy().SaveContent([]byte(next))
}

func (s *SSLService) PushCDN(ctx context.Context, req request.SSLPushCDN) error {
	ssl, err := s.repo.GetFirst(repo.NewSSL().WithID(req.SSLID))
	if err != nil {
		return fmt.Errorf("certificate not found: %v", err)
	}

	targetDomain := req.TargetDomain
	if targetDomain == "" {
		targetDomain = ssl.PrimaryDomain
	}

	cloudAccountRepo := repo.NewCloudAccount()
	cloudAccount, err := cloudAccountRepo.GetByID(req.CloudAccountID)
	if err != nil {
		return fmt.Errorf("cloud account not found: %v", err)
	}

	var authData map[string]interface{}
	_ = json.Unmarshal([]byte(cloudAccount.Authorization), &authData)

	switch cloudAccount.Type {
	case "aliyun":
		ak, _ := authData["accessKey"].(string)
		sk, _ := authData["secretKey"].(string)
		err = aliyun.SetCdnDomainSSLCertificate(
			ak,
			sk,
			targetDomain,
			fmt.Sprintf("gopanel-%s-%d", strings.ReplaceAll(targetDomain, ".", "-"), ssl.ID),
			ssl.Pem,
			ssl.PrivateKey,
		)
		if err != nil {
			return fmt.Errorf("aliyun cdn push failed: %v", err)
		}
	default:
		return fmt.Errorf("unsupported provider: %s", cloudAccount.Type)
	}

	return nil
}

func (s *SSLService) Obtain(websiteID uint) (res *model.SSL, err error) {
	website, err := repo.NewWebsite().GetFirst(repo.NewWebsite().WithID(websiteID))
	if err != nil {
		return nil, err
	}
	certPath, keyPath, err := findManagedCertificateFiles(website.PrimaryDomain)
	if err != nil {
		return nil, err
	}
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	info, err := parseCertificateInfo(string(certBytes))
	if err != nil {
		return nil, err
	}
	item, findErr := s.repo.GetFirst(s.repo.WithDomain(website.PrimaryDomain))
	if findErr != nil {
		item = model.SSL{}
	}
	item.PrimaryDomain = website.PrimaryDomain
	if len(info.Domains) > 0 {
		item.PrimaryDomain = info.Domains[0]
		item.Domains = strings.Join(info.Domains, ",")
	} else {
		item.Domains = website.PrimaryDomain
	}
	item.Type = "caddy"
	item.Provider = "caddy"
	item.Pem = string(certBytes)
	item.PrivateKey = string(keyBytes)
	item.StartDate = info.StartDate
	item.ExpireDate = info.ExpireDate
	item.Organization = info.IssuerName
	item.Status = "issued"
	item.AutoRenew = true
	item.Dir = filepath.Dir(certPath)
	if item.Organization == "" {
		item.Organization = "Caddy Managed"
	}
	if item.ID == 0 {
		if err = s.repo.Create(context.Background(), &item); err != nil {
			return nil, err
		}
	} else {
		if err = s.repo.SaveWithoutCtx(&item); err != nil {
			return nil, err
		}
	}

	// 每次同步/获取到新证书后，尝试触发自动推送部署规则
	go func() {
		_ = AutoPushCertificate(item.ID)
	}()

	res = &item
	if err = s.attachWebsiteRelations([]*model.SSL{res}); err != nil {
		return nil, err
	}
	return
}

func (s *SSLService) List(ctx *gormx.Contextx) (res []*model.SSL, err error) {
	res, err = s.repo.Search(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.attachWebsiteRelations(res); err != nil {
		return nil, err
	}
	return
}

func (s *SSLService) CountByWhere(where *gormx.Wherex) (res int64, err error) {
	return s.repo.CountByWhere(where)
}

type certificateInfo struct {
	Domains    []string
	StartDate  time.Time
	ExpireDate time.Time
	IssuerName string
}

func normalizeKeyType(value string) string {
	switch strings.TrimSpace(value) {
	case "P384", "2048", "3072", "4096":
		return strings.TrimSpace(value)
	default:
		return "P256"
	}
}

func normalizeDomains(primary, others string) ([]string, string, string) {
	raw := append([]string{primary}, strings.FieldsFunc(others, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' '
	})...)
	seen := make(map[string]struct{})
	domains := make([]string, 0, len(raw))
	for _, item := range raw {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		domains = append(domains, value)
	}
	primaryDomain := ""
	if len(domains) > 0 {
		primaryDomain = domains[0]
	}
	return domains, primaryDomain, strings.Join(domains[1:], ",")
}

func parseCertificateInfo(certPEM string) (*certificateInfo, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("证书 PEM 内容无效")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	domainSet := append([]string{}, cert.DNSNames...)
	if cert.Subject.CommonName != "" {
		exists := false
		for _, item := range domainSet {
			if item == cert.Subject.CommonName {
				exists = true
				break
			}
		}
		if !exists {
			domainSet = append([]string{cert.Subject.CommonName}, domainSet...)
		}
	}
	return &certificateInfo{
		Domains:    domainSet,
		StartDate:  cert.NotBefore,
		ExpireDate: cert.NotAfter,
		IssuerName: cert.Issuer.CommonName,
	}, nil
}

func (s *SSLService) certificateDir(id uint) string {
	return filepath.Join(global.CONF.System.BaseDir, "data", "ssl", fmt.Sprintf("%d", id))
}

func (s *SSLService) persistCertificateFiles(item *model.SSL) (string, error) {
	dir := s.certificateDir(item.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), []byte(item.Pem), 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "privkey.pem"), []byte(item.PrivateKey), 0o600); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *SSLService) ensureCertificateFiles(item *model.SSL) (string, string, error) {
	dir := item.Dir
	if dir == "" {
		var err error
		dir, err = s.persistCertificateFiles(item)
		if err != nil {
			return "", "", err
		}
		item.Dir = dir
		_ = s.repo.SaveWithoutCtx(item)
	}
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	if _, err := os.Stat(certPath); err != nil {
		if _, err = s.persistCertificateFiles(item); err != nil {
			return "", "", err
		}
	}
	return certPath, keyPath, nil
}

func (s *SSLService) attachWebsiteRelations(items []*model.SSL) error {
	if len(items) == 0 {
		return nil
	}

	websiteRepo := repo.NewWebsite()
	// 查询出所有的 Website 及其关联的 Domains
	websites, err := websiteRepo.ListBy()
	if err != nil {
		return err
	}

	for _, item := range items {
		// 构建当前 SSL 证书涵盖的域名集合
		domainSet := make(map[string]struct{})
		for _, domain := range strings.Split(item.Domains, ",") {
			value := strings.TrimSpace(domain)
			if value != "" {
				domainSet[value] = struct{}{}
			}
		}

		linked := make([]model.Website, 0)
		for _, website := range websites {
			// 构建当前网站的所有域名（主域名 + 附加域名）
			relatedDomains := make([]string, 0, 1+len(website.Domains))
			relatedDomains = append(relatedDomains, website.PrimaryDomain)
			for _, domain := range website.Domains {
				relatedDomains = append(relatedDomains, domain.Domain)
			}

			// 获取网站规范化后的域名列表（如处理通配符等）
			websiteDomains, _, _ := normalizeDomains(relatedDomains[0], strings.Join(relatedDomains[1:], ","))

			// 检查证书的域名集合是否覆盖了该网站的任意一个域名
			for _, domain := range websiteDomains {
				if _, ok := domainSet[domain]; ok {
					linked = append(linked, website)
					break // 只要命中一个域名就认为该网站使用了此证书
				}
			}
		}
		item.Websites = linked
	}

	return nil
}

func upsertTLSDirective(content, domain, certPath, keyPath string) (string, error) {
	pattern := regexp.MustCompile(`(?ms)(^` + regexp.QuoteMeta(domain) + `\s*\{\n)(.*?)(^\})`)
	match := pattern.FindStringSubmatch(content)
	if len(match) != 4 {
		return content, fmt.Errorf("未找到域名 %s 的站点配置", domain)
	}
	body := regexp.MustCompile(`(?m)^\s*tls\s+.+\n?`).ReplaceAllString(match[2], "")
	body = strings.TrimLeft(body, "\n")
	replacement := match[1] + "\ttls " + certPath + " " + keyPath + "\n" + body + match[3]
	return pattern.ReplaceAllString(content, replacement), nil
}

func findManagedCertificateFiles(domain string) (string, string, error) {
	safeDomain := certmagic.StorageKeys.Safe(domain)
	certPattern := filepath.Join(caddy.AppDataDir(), "certificates", "*", safeDomain, safeDomain+".crt")
	matches, err := filepath.Glob(certPattern)
	if err != nil {
		return "", "", err
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("未找到域名 %s 的 Caddy 自动证书，请确认域名解析和 80/443 访问已生效", domain)
	}
	certPath := matches[0]
	keyPath := filepath.Join(filepath.Dir(certPath), safeDomain+".key")
	if _, err = os.Stat(keyPath); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}
