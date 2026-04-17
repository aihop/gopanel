package tencentcloud

import (
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type DNSProvider struct {
	SecretId  string
	SecretKey string
}

func NewDNSProvider(secretId, secretKey string) challenge.Provider {
	return &DNSProvider{
		SecretId:  secretId,
		SecretKey: secretKey,
	}
}

func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("could not find zone for fqdn %s: %v", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)
	name := dns01.UnFqdn(fqdn)
	name = strings.TrimSuffix(name, "."+zone)

	return AddTxtRecord(d.SecretId, d.SecretKey, zone, name, value)
}

func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("could not find zone for fqdn %s: %v", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)
	name := dns01.UnFqdn(fqdn)
	name = strings.TrimSuffix(name, "."+zone)

	return DeleteSubDomainRecords(d.SecretId, d.SecretKey, zone, name)
}
