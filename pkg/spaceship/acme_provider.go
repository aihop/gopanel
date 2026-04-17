package spaceship

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type DNSProvider struct {
	client *Client
}

func NewDNSProvider(apiKey, apiSecret string) challenge.Provider {
	client, _ := NewClient(apiKey, apiSecret)
	return &DNSProvider{
		client: client,
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	record := Record{
		Type:  "TXT",
		Name:  name,
		Value: value,
		TTL:   120, // 2 minutes
	}

	return d.client.AddRecord(ctx, zone, record)
}

func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return fmt.Errorf("could not find zone for fqdn %s: %v", fqdn, err)
	}

	zone = dns01.UnFqdn(zone)
	name := dns01.UnFqdn(fqdn)
	name = strings.TrimSuffix(name, "."+zone)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	record := Record{
		Type:  "TXT",
		Name:  name,
		Value: value,
	}

	return d.client.DeleteRecord(ctx, zone, record)
}
