package aliyun

import (
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type DNSProvider struct {
	AccessKey string
	SecretKey string
}

// NewDNSProvider 返回基于原始 HTTP 签名的轻量级阿里云 DNS 验证提供者，避免引入庞大的官方 SDK
func NewDNSProvider(accessKey, secretKey string) challenge.Provider {
	return &DNSProvider{
		AccessKey: accessKey,
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

	return AddTxtRecord(d.AccessKey, d.SecretKey, zone, name, value)
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

	return DeleteSubDomainRecords(d.AccessKey, d.SecretKey, zone, name)
}
