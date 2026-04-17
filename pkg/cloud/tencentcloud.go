package cloud

import (
	"github.com/aihop/gopanel/pkg/tencentcloud"
	"github.com/go-acme/lego/v4/challenge"
)

type TencentCloudProvider struct {
	SecretId  string
	SecretKey string
}

func NewTencentCloudProvider(secretId, secretKey string) Provider {
	return &TencentCloudProvider{
		SecretId:  secretId,
		SecretKey: secretKey,
	}
}

func (p *TencentCloudProvider) GetDNSProvider() (challenge.Provider, error) {
	return tencentcloud.NewDNSProvider(p.SecretId, p.SecretKey), nil
}

func (p *TencentCloudProvider) ListCdnDomains() ([]string, error) {
	return tencentcloud.ListCdnDomains(p.SecretId, p.SecretKey)
}

func (p *TencentCloudProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return tencentcloud.SetCdnDomainSSLCertificate(p.SecretId, p.SecretKey, domainName, certName, sslPub, sslPri)
}
