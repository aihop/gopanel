package cloudflare

import (
	"encoding/json"
	"fmt"
)

// ListCdnDomains 获取 Cloudflare 的 Zone 列表 (视为 CDN 域名)
func ListCdnDomains(token string) ([]string, error) {
	path := "/zones?per_page=100"
	bodyBytes, err := doCloudflareRequest(token, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("cloudflare list zones failed: %s", string(bodyBytes))
	}

	var domains []string
	for _, z := range result.Result {
		domains = append(domains, z.Name)
	}

	return domains, nil
}

// SetCdnDomainSSLCertificate 上传自定义证书到 Cloudflare (需要 Business/Enterprise Plan 或 Advanced Certificate Manager)
func SetCdnDomainSSLCertificate(token, domainName, certName, sslPub, sslPri string) error {
	zoneId, err := getZoneIdByName(token, domainName)
	if err != nil {
		return err
	}

	// Cloudflare 上传 Custom SSL 的接口 (POST /zones/:zone_identifier/custom_certificates)
	path := fmt.Sprintf("/zones/%s/custom_certificates", zoneId)
	
	reqMap := map[string]interface{}{
		"certificate": sslPub,
		"private_key": sslPri,
		"bundle_method": "ubiquitous",
	}

	payload, err := json.Marshal(reqMap)
	if err != nil {
		return err
	}

	_, err = doCloudflareRequest(token, "POST", path, payload)
	if err != nil {
		return fmt.Errorf("cloudflare custom certificate push failed: %v", err)
	}

	return nil
}
