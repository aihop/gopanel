package tencentcloud

import (
	"encoding/json"
	"fmt"
)

// ListCdnDomains 获取腾讯云 CDN 域名列表
func ListCdnDomains(secretId, secretKey string) ([]string, error) {
	// API 参数: https://cloud.tencent.com/document/api/228/41118 (DescribeDomainsConfig)
	payload := []byte(`{"Limit":100}`)

	bodyBytes, err := doTencentCloudRequestv3(
		secretId,
		secretKey,
		"cdn",
		"2018-06-06",
		"DescribeDomainsConfig",
		"cdn.tencentcloudapi.com",
		"",
		payload,
	)
	if err != nil {
		return nil, err
	}

	var result struct {
		Response struct {
			Domains []struct {
				Domain string `json:"Domain"`
			} `json:"Domains"`
		} `json:"Response"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	var domains []string
	for _, d := range result.Response.Domains {
		domains = append(domains, d.Domain)
	}

	return domains, nil
}

// SetCdnDomainSSLCertificate 更新腾讯云 CDN 域名的 SSL 证书
func SetCdnDomainSSLCertificate(secretId, secretKey, domainName, certName, sslPub, sslPri string) error {
	// 先调用 UploadCdnCertificate (如果不需要复用，或者直接 UpdateDomainConfig)
	// 在腾讯云 CDN API 3.0 中，使用 UpdateDomainConfig 接口可以直接配置 Https 证书
	// 参考: https://cloud.tencent.com/document/api/228/41116
	
	// 注意：由于是私有证书（未托管在 SSL 证书管理服务中），可以直接通过 UpdateDomainConfig 传 CertInfo
	
	reqMap := map[string]interface{}{
		"Domain": domainName,
		"Https": map[string]interface{}{
			"Switch": "on",
			"CertInfo": map[string]interface{}{
				"Message":      sslPri,
				"Certificate":  sslPub,
				"CertType":     "upload",
				"CertName":     certName,
			},
		},
	}
	
	payload, err := json.Marshal(reqMap)
	if err != nil {
		return err
	}

	_, err = doTencentCloudRequestv3(
		secretId,
		secretKey,
		"cdn",
		"2018-06-06",
		"UpdateDomainConfig",
		"cdn.tencentcloudapi.com",
		"",
		payload,
	)
	
	if err != nil {
		return fmt.Errorf("tencentcloud cdn push failed: %v", err)
	}
	
	return nil
}
