package service

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
	return &certificateInfo{Domains: domainSet, StartDate: cert.NotBefore, ExpireDate: cert.NotAfter, IssuerName: cert.Issuer.CommonName}, nil
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
	websites, err := websiteRepo.ListBy()
	if err != nil {
		return err
	}
	for _, item := range items {
		domainSet := make(map[string]struct{})
		for _, domain := range strings.Split(item.Domains, ",") {
			value := strings.TrimSpace(domain)
			if value != "" {
				domainSet[value] = struct{}{}
			}
		}
		linked := make([]model.Website, 0)
		for _, website := range websites {
			relatedDomains := make([]string, 0, 1+len(website.Domains))
			relatedDomains = append(relatedDomains, website.PrimaryDomain)
			for _, domain := range website.Domains {
				relatedDomains = append(relatedDomains, domain.Domain)
			}
			websiteDomains, _, _ := normalizeDomains(relatedDomains[0], strings.Join(relatedDomains[1:], ","))
			for _, domain := range websiteDomains {
				if _, ok := domainSet[domain]; ok {
					linked = append(linked, website)
					break
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
	return "", "", nil
}
