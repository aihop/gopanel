package cloud

import (
	"github.com/aihop/gopanel/pkg/spaceship"
	"github.com/go-acme/lego/v4/challenge"
)

type SpaceshipProvider struct {
	AccessKey string
	SecretKey string
}

func NewSpaceshipProvider(ak, sk string) Provider {
	return &SpaceshipProvider{
		AccessKey: ak,
		SecretKey: sk,
	}
}

func (p *SpaceshipProvider) GetDNSProvider() (challenge.Provider, error) {
	return spaceship.NewDNSProvider(p.AccessKey, p.SecretKey), nil
}

func (p *SpaceshipProvider) ListCdnDomains() ([]string, error) {
	return spaceship.ListCdnDomains(p.AccessKey, p.SecretKey)
}

func (p *SpaceshipProvider) SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error {
	return spaceship.SetCdnDomainSSLCertificate(p.AccessKey, p.SecretKey, domainName, certName, sslPub, sslPri)
}
