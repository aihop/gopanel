package huaweicloud

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type DNSProvider struct {
	AccessKey string
	SecretKey string
}

func NewDNSProvider(ak, sk string) challenge.Provider {
	return &DNSProvider{
		AccessKey: ak,
		SecretKey: sk,
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

func getZoneIdByName(ak, sk, domain string) (string, error) {
	// DNS API: https://support.huaweicloud.com/api-dns/dns_api_64004.html
	query := fmt.Sprintf("name=%s", domain)
	bodyBytes, err := doHuaweiCloudRequest(ak, sk, "GET", "dns.myhuaweicloud.com", "/v2/zones", query, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Zones []struct {
			Id string `json:"id"`
		} `json:"zones"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Zones) == 0 {
		return "", fmt.Errorf("could not find zone id for domain %s", domain)
	}

	return result.Zones[0].Id, nil
}

func AddTxtRecord(ak, sk, domain, rr, value string) error {
	zoneId, err := getZoneIdByName(ak, sk, domain)
	if err != nil {
		return err
	}

	reqMap := map[string]interface{}{
		"name":    fmt.Sprintf("%s.%s.", rr, domain),
		"type":    "TXT",
		"records": []string{fmt.Sprintf("\"%s\"", value)},
		"ttl":     300,
	}
	payload, _ := json.Marshal(reqMap)

	path := fmt.Sprintf("/v2/zones/%s/recordsets", zoneId)
	_, err = doHuaweiCloudRequest(ak, sk, "POST", "dns.myhuaweicloud.com", path, "", payload)
	return err
}

func DeleteSubDomainRecords(ak, sk, domain, rr string) error {
	zoneId, err := getZoneIdByName(ak, sk, domain)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s.%s.", rr, domain)
	query := fmt.Sprintf("name=%s&type=TXT", name)
	path := fmt.Sprintf("/v2/zones/%s/recordsets", zoneId)
	
	bodyBytes, err := doHuaweiCloudRequest(ak, sk, "GET", "dns.myhuaweicloud.com", path, query, nil)
	if err != nil {
		return err
	}

	var result struct {
		Recordsets []struct {
			Id string `json:"id"`
		} `json:"recordsets"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return err
	}

	for _, record := range result.Recordsets {
		delPath := fmt.Sprintf("/v2/zones/%s/recordsets/%s", zoneId, record.Id)
		_, err := doHuaweiCloudRequest(ak, sk, "DELETE", "dns.myhuaweicloud.com", delPath, "", nil)
		if err != nil {
			return fmt.Errorf("failed to delete huaweicloud record %s: %v", record.Id, err)
		}
	}

	return nil
}
