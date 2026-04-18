package huaweicloud

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ListCdnDomains 获取华为云 CDN 域名列表
func ListCdnDomains(ak, sk string) ([]string, error) {
	// API: https://support.huaweicloud.com/api-cdn/cdn_02_0006.html
	bodyBytes, err := doHuaweiCloudRequest(
		ak,
		sk,
		"GET",
		"cdn.myhuaweicloud.com",
		"/v1.0/cdn/domains",
		"page_size=100",
		nil,
	)
	if err != nil {
		return nil, err
	}

	var result struct {
		Domains []struct {
			DomainName string `json:"domain_name"`
		} `json:"domains"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	var domains []string
	for _, d := range result.Domains {
		domains = append(domains, d.DomainName)
	}

	return domains, nil
}

// SetCdnDomainSSLCertificate 更新华为云 CDN 域名的 SSL 证书
func SetCdnDomainSSLCertificate(ak, sk, domainName, certName, sslPub, sslPri string) error {
	// API: https://support.huaweicloud.com/api-cdn/cdn_02_0026.html
	reqMap := map[string]interface{}{
		"https": map[string]interface{}{
			"https_status":   2,
			"certificate_name": certName,
			"certificate_value": sslPub,
			"private_key":      sslPri,
		},
	}
	
	payload, err := json.Marshal(reqMap)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1.0/cdn/domains/%s/https-info", url.QueryEscape(domainName))
	_, err = doHuaweiCloudRequest(
		ak,
		sk,
		"PUT",
		"cdn.myhuaweicloud.com",
		path,
		"",
		payload,
	)
	
	if err != nil {
		return fmt.Errorf("huaweicloud cdn push failed: %v", err)
	}
	
	return nil
}
