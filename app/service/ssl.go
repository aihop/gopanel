package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/pkg/aliyun"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/go-acme/lego/v4/registration"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type acmeLogger struct{ logger *SSLLogger }
type acmeUser struct {
	Email        string
	Registration *registration.Resource
	key          *ecdsa.PrivateKey
}

func NewSSL() *SSLService {
	return &SSLService{repo: repo.NewSSL()}
}

type SSLService struct{ repo *repo.SSLRepo }

func (s *SSLService) Create(req *request.SSLCreate) (*model.SSL, error) {
	domains, primary, _ := normalizeDomains(req.PrimaryDomain, req.OtherDomains)
	if primary == "" {
		return nil, errors.New("主域名不能为空")
	}
	item := &model.SSL{PrimaryDomain: primary, Domains: strings.Join(domains, ","), Type: req.Type, Description: strings.TrimSpace(req.Description), KeyType: normalizeKeyType(req.KeyType), AcmeAccountID: req.AcmeAccountID, CloudAccountID: req.CloudAccountID, DnsAccountID: req.DnsAccountID, Status: "issued", AutoRenew: false}
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
		item.Organization = "Let's Encrypt"
		item.AutoRenew = true
		if req.DnsAccountID > 0 {
			cloudAccountRepo := repo.NewCloudAccount()
			if account, err := cloudAccountRepo.GetByID(req.DnsAccountID); err == nil {
				item.Type = "dns-" + account.Type
			}
		} else if req.CloudAccountID > 0 {
			cloudAccountRepo := repo.NewCloudAccount()
			if account, err := cloudAccountRepo.GetByID(req.CloudAccountID); err == nil {
				item.Type = "dns-" + account.Type
			}
		}
		if err := s.repo.Create(context.Background(), item); err != nil {
			return nil, err
		}
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
		return errors.New("Caddy 管理的证书不支持手动重签")
	}
	if strings.HasPrefix(item.Type, "dns-") {
		item.Status = "pending"
		if err := s.repo.SaveWithoutCtx(&item); err != nil {
			return err
		}
		refreshedItem, _ := s.repo.GetFirst(s.repo.WithID(item.ID))
		go s.obtainCloudAcmeCertificate(&refreshedItem)
		return nil
	}
	return errors.New("不支持的证书类型")
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
	domainValues := make([]string, 0, 1+len(website.Domains))
	domainValues = append(domainValues, website.PrimaryDomain)
	for _, domain := range website.Domains {
		domainValues = append(domainValues, domain.Domain)
	}
	domains, _, _ := normalizeDomains(domainValues[0], strings.Join(domainValues[1:], ","))
	if len(domains) == 0 {
		return errors.New("网站未配置域名")
	}
	return nil
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
		err = aliyun.SetCdnDomainSSLCertificate(ak, sk, targetDomain, fmt.Sprintf("gopanel-%s-%d", strings.ReplaceAll(targetDomain, ".", "-"), ssl.ID), ssl.Pem, ssl.PrivateKey)
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
