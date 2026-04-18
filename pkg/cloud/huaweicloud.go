package cloud

import (
	"github.com/aihop/gopanel/pkg/huaweicloud"
	"github.com/go-acme/lego/v4/challenge"
)

type HuaweiCloudProvider struct {
	AccessKey string
	SecretKey string
}

func NewHuaweiCloudProvider(ak, sk string) Provider {
	return &HuaweiCloudProvider{
		AccessKey: ak,
		SecretKey: sk,
	}
}

func (p *HuaweiCloudProvider) GetDNSProvider() (challenge.Provider, error) {
	return huaweicloud.NewDNSProvider(p.AccessKey, p.SecretKey), nil
}

func (p *HuaweiCloudProvider) ListCdnDomains() ([]string, error) {
	return huaweicloud.ListCdnDomains(p.AccessKey, p.SecretKey)
}

func (p *HuaweiCloudProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return huaweicloud.SetCdnDomainSSLCertificate(p.AccessKey, p.SecretKey, domainName, certName, sslPub, sslPri)
}
