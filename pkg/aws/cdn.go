package aws

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// ImportCertificateToACM uploads a new certificate to AWS ACM and returns the CertificateArn
func ImportCertificateToACM(ak, sk, sslPub, sslPri string) (string, error) {
	// AWS ACM uses JSON for its API payloads, unlike Route53/CloudFront which use XML mostly.
	// For ACM to be used with CloudFront, it MUST be imported into the us-east-1 region.
	region := "us-east-1"
	
	payloadMap := map[string]interface{}{
		"Certificate": sslPub,
		"PrivateKey":  sslPri,
	}
	
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "CertificateManager.ImportCertificate",
	}

	bodyBytes, err := doAWSSigV4Request(
		ak, sk,
		region,
		"acm",
		"POST",
		"acm.us-east-1.amazonaws.com",
		"/",
		"",
		payloadBytes,
		headers,
	)

	if err != nil {
		return "", err
	}

	var result struct {
		CertificateArn string `json:"CertificateArn"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if result.CertificateArn == "" {
		return "", fmt.Errorf("failed to import certificate to ACM, arn is empty: %s", string(bodyBytes))
	}

	return result.CertificateArn, nil
}

// SetCdnDomainSSLCertificate updates a CloudFront Distribution to use the new ACM Certificate
func SetCdnDomainSSLCertificate(ak, sk, domainName, certName, sslPub, sslPri string) error {
	// 1. Upload certificate to ACM in us-east-1
	certArn, err := ImportCertificateToACM(ak, sk, sslPub, sslPri)
	if err != nil {
		return fmt.Errorf("failed to upload cert to AWS ACM: %v", err)
	}

	// 2. Find the CloudFront distribution that has this domain as an alias
	bodyBytes, err := doAWSSigV4Request(ak, sk, "us-east-1", "cloudfront", "GET", "cloudfront.amazonaws.com", "/2020-05-31/distribution", "", nil, nil)
	if err != nil {
		return err
	}

	var distList struct {
		Items []struct {
			Id      string `xml:"Id"`
			Aliases struct {
				Items []string `xml:"Items>CNAME"`
			} `xml:"Aliases"`
		} `xml:"Items>DistributionSummary"`
	}

	if err := xml.Unmarshal(bodyBytes, &distList); err != nil {
		return err
	}

	var targetDistId string
	for _, item := range distList.Items {
		for _, alias := range item.Aliases.Items {
			if alias == domainName {
				targetDistId = item.Id
				break
			}
		}
		if targetDistId != "" {
			break
		}
	}

	if targetDistId == "" {
		return fmt.Errorf("could not find CloudFront distribution for domain %s", domainName)
	}

	// 3. Get the current distribution config to update it
	getConfigPath := fmt.Sprintf("/2020-05-31/distribution/%s/config", targetDistId)
	_, err = doAWSSigV4Request(ak, sk, "us-east-1", "cloudfront", "GET", "cloudfront.amazonaws.com", getConfigPath, "", nil, nil)
	if err != nil {
		return err
	}

	// For AWS CloudFront updates, we MUST pass the ETag from the GET request as the If-Match header in the PUT request.
	// Since our doAWSSigV4Request currently doesn't return headers, we need to extract ETag.
	// We will update the doAWSSigV4Request or write a specialized one if needed, but for simplicity we can parse the XML and inject the Arn.
	
	// Note: Fully updating CloudFront requires passing the exact same XML back with modifications.
	// This is a simplified placeholder as full CloudFront XML manipulation is very complex.
	return fmt.Errorf("CloudFront distribution update with ACM Arn %s requires If-Match ETag implementation", certArn)
}

// ListCdnDomains 获取 AWS CloudFront 分发列表 (Distribution) 中的所有 CNAMEs
func ListCdnDomains(ak, sk string) ([]string, error) {
	// AWS CloudFront API endpoints are usually on cloudfront.amazonaws.com (global service)
	bodyBytes, err := doAWSSigV4Request(
		ak, sk,
		"us-east-1",
		"cloudfront",
		"GET",
		"cloudfront.amazonaws.com",
		"/2020-05-31/distribution",
		"",
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var distList struct {
		Items []struct {
			Id      string `xml:"Id"`
			Aliases struct {
				Items []string `xml:"Items>CNAME"`
			} `xml:"Aliases"`
		} `xml:"Items>DistributionSummary"`
	}

	if err := xml.Unmarshal(bodyBytes, &distList); err != nil {
		return nil, err
	}

	var domains []string
	for _, item := range distList.Items {
		domains = append(domains, item.Aliases.Items...)
	}

	return domains, nil
}
