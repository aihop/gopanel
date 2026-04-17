package cloud

import (
	"github.com/aihop/gopanel/pkg/cloudflare"
	"github.com/go-acme/lego/v4/challenge"
)

type CloudflareProvider struct {
	Token string
}

func NewCloudflareProvider(token string) Provider {
	return &CloudflareProvider{
		Token: token,
	}
}

func (p *CloudflareProvider) GetDNSProvider() (challenge.Provider, error) {
	return cloudflare.NewDNSProvider(p.Token), nil
}

func (p *CloudflareProvider) ListCdnDomains() ([]string, error) {
	return cloudflare.ListCdnDomains(p.Token)
}

func (p *CloudflareProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return cloudflare.SetCdnDomainSSLCertificate(p.Token, domainName, certName, sslPub, sslPri)
}
