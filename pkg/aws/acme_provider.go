package aws

import (
	"encoding/xml"
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

	return AddTxtRecord(d.AccessKey, d.SecretKey, zone, fqdn, value)
}

func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("could not find zone for fqdn %s: %v", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)

	return DeleteSubDomainRecords(d.AccessKey, d.SecretKey, zone, fqdn)
}

func getHostedZoneIdByName(ak, sk, domain string) (string, error) {
	// AWS Route 53 API uses global region "us-east-1" for endpoint
	query := "name=" + domain + "&maxitems=1"
	bodyBytes, err := doAWSSigV4Request(ak, sk, "us-east-1", "route53", "GET", "route53.amazonaws.com", "/2013-04-01/hostedzonesbyname", query, nil, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		HostedZones []struct {
			Id   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZones>HostedZone"`
	}

	if err := xml.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.HostedZones) == 0 {
		return "", fmt.Errorf("could not find hosted zone for domain %s", domain)
	}

	// Clean up /hostedzone/ prefix if exists
	zoneId := strings.TrimPrefix(result.HostedZones[0].Id, "/hostedzone/")
	return zoneId, nil
}

func changeResourceRecordSets(ak, sk, zoneId, action, fqdn, value string) error {
	name := dns01.UnFqdn(fqdn)

	payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
   <ChangeBatch>
      <Changes>
         <Change>
            <Action>%s</Action>
            <ResourceRecordSet>
               <Name>%s</Name>
               <Type>TXT</Type>
               <TTL>120</TTL>
               <ResourceRecords>
                  <ResourceRecord>
                     <Value>"%s"</Value>
                  </ResourceRecord>
               </ResourceRecords>
            </ResourceRecordSet>
         </Change>
      </Changes>
   </ChangeBatch>
</ChangeResourceRecordSetsRequest>`, action, name, value)

	path := fmt.Sprintf("/2013-04-01/hostedzone/%s/rrset/", zoneId)
	headers := map[string]string{
		"Content-Type": "application/xml",
	}

	_, err := doAWSSigV4Request(ak, sk, "us-east-1", "route53", "POST", "route53.amazonaws.com", path, "", []byte(payload), headers)
	return err
}

func AddTxtRecord(ak, sk, zoneName, fqdn, value string) error {
	zoneId, err := getHostedZoneIdByName(ak, sk, zoneName)
	if err != nil {
		return err
	}

	return changeResourceRecordSets(ak, sk, zoneId, "UPSERT", fqdn, value)
}

func DeleteSubDomainRecords(ak, sk, zoneName, fqdn string) error {
	zoneId, err := getHostedZoneIdByName(ak, sk, zoneName)
	if err != nil {
		return err
	}

	// For deletion, we need to know the exact value.
	// In a real robust implementation, we would query the existing TXT record first.
	// For simplicity in this acme cleanup context, we might skip deletion if value is unknown
	// or we can implement a read-before-delete logic.
	
	// Simplification: We attempt to delete with a dummy value (Route53 might reject this if it doesn't match).
	// To make it reliable, we should fetch the existing record first.
	query := fmt.Sprintf("name=%s&type=TXT&maxitems=1", fqdn)
	path := fmt.Sprintf("/2013-04-01/hostedzone/%s/rrset", zoneId)
	bodyBytes, err := doAWSSigV4Request(ak, sk, "us-east-1", "route53", "GET", "route53.amazonaws.com", path, query, nil, nil)
	if err != nil {
		return err
	}

	var result struct {
		ResourceRecordSets []struct {
			Name            string `xml:"Name"`
			Type            string `xml:"Type"`
			ResourceRecords []struct {
				Value string `xml:"Value"`
			} `xml:"ResourceRecords>ResourceRecord"`
		} `xml:"ResourceRecordSets>ResourceRecordSet"`
	}

	if err := xml.Unmarshal(bodyBytes, &result); err != nil {
		return err
	}

	for _, rrset := range result.ResourceRecordSets {
		if rrset.Type == "TXT" {
			for _, record := range rrset.ResourceRecords {
				// Clean up quotes from the value
				cleanValue := strings.Trim(record.Value, "\"")
				err = changeResourceRecordSets(ak, sk, zoneId, "DELETE", fqdn, cleanValue)
				if err != nil {
					return fmt.Errorf("failed to delete AWS Route53 record: %v", err)
				}
			}
		}
	}

	return nil
}
