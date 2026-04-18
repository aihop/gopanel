package cloud

import (
	"github.com/aihop/gopanel/pkg/aws"
	"github.com/go-acme/lego/v4/challenge"
)

type AWSProvider struct {
	AccessKey string
	SecretKey string
}

func NewAWSProvider(ak, sk string) Provider {
	return &AWSProvider{
		AccessKey: ak,
		SecretKey: sk,
	}
}

func (p *AWSProvider) GetDNSProvider() (challenge.Provider, error) {
	return aws.NewDNSProvider(p.AccessKey, p.SecretKey), nil
}

func (p *AWSProvider) ListCdnDomains() ([]string, error) {
	return aws.ListCdnDomains(p.AccessKey, p.SecretKey)
}

func (p *AWSProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return aws.SetCdnDomainSSLCertificate(p.AccessKey, p.SecretKey, domainName, certName, sslPub, sslPri)
}
