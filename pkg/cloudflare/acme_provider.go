package cloudflare

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type DNSProvider struct {
	Token string
}

func NewDNSProvider(token string) challenge.Provider {
	return &DNSProvider{
		Token: token,
	}
}

func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return 5 * time.Minute, 5 * time.Second
}

func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("cloudflare dns-01 present failed: could not find zone for fqdn %s: %w", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)
	name := dns01.UnFqdn(fqdn)
	name = strings.TrimSuffix(name, "."+zone)

	if err := AddTxtRecord(d.Token, zone, fqdn, value); err != nil {
		return fmt.Errorf("cloudflare dns-01 present failed: zone=%s record=%s fqdn=%s value_length=%d err=%w", zone, name, fqdn, len(value), err)
	}
	return nil
}

func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("cloudflare dns-01 cleanup failed: could not find zone for fqdn %s: %w", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)

	if err := DeleteSubDomainRecords(d.Token, zone, fqdn); err != nil {
		return fmt.Errorf("cloudflare dns-01 cleanup failed: zone=%s fqdn=%s err=%w", zone, fqdn, err)
	}
	return nil
}

func AddTxtRecord(token, zoneName, fqdn, value string) error {
	zoneId, err := getZoneIdByName(token, zoneName)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/zones/%s/dns_records", zoneId)
	name := dns01.UnFqdn(fqdn)
	reqMap := map[string]interface{}{
		"type":    "TXT",
		"name":    name,
		"content": value,
		"ttl":     120,
	}
	payload, _ := json.Marshal(reqMap)

	_, err = doCloudflareRequest(token, "POST", path, payload)
	return err
}

func DeleteSubDomainRecords(token, zoneName, fqdn string) error {
	zoneId, err := getZoneIdByName(token, zoneName)
	if err != nil {
		return err
	}

	name := dns01.UnFqdn(fqdn)
	path := fmt.Sprintf("/zones/%s/dns_records?type=TXT&name=%s", zoneId, name)
	bodyBytes, err := doCloudflareRequest(token, "GET", path, nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			Id string `json:"id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return err
	}

	for _, record := range result.Result {
		delPath := fmt.Sprintf("/zones/%s/dns_records/%s", zoneId, record.Id)
		_, err := doCloudflareRequest(token, "DELETE", delPath, nil)
		if err != nil {
			return fmt.Errorf("failed to delete cloudflare record %s: %v", record.Id, err)
		}
	}

	return nil
}
