package cloud

import (
	"github.com/aihop/gopanel/pkg/aliyun"
	"github.com/go-acme/lego/v4/challenge"
)

type AliyunProvider struct {
	AccessKey string
	SecretKey string
}

func NewAliyunProvider(accessKey, secretKey string) Provider {
	return &AliyunProvider{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

func (p *AliyunProvider) GetDNSProvider() (challenge.Provider, error) {
	return aliyun.NewDNSProvider(p.AccessKey, p.SecretKey), nil
}

func (p *AliyunProvider) ListCdnDomains() ([]string, error) {
	return aliyun.ListCdnDomains(p.AccessKey, p.SecretKey)
}

func (p *AliyunProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return aliyun.SetCdnDomainSSLCertificate(p.AccessKey, p.SecretKey, domainName, certName, sslPub, sslPri)
}
