package aliyun

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SetCdnDomainSSLCertificate calls Aliyun CDN API to set SSL certificate
func SetCdnDomainSSLCertificate(accessKey, secretKey, domainName, certName, sslPub, sslPri string) error {
	params := map[string]string{
		"Action":      "SetCdnDomainSSLCertificate",
		"Version":     "2018-05-10",
		"Format":      "JSON",
		"AccessKeyId": accessKey,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.New().String(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"DomainName":       domainName,
		"CertName":         certName,
		"CertType":         "upload",
		"SSLProtocol":      "on",
		"SSLPub":           sslPub,
		"SSLPri":           sslPri,
	}

	// 1. Sort parameters by key
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. Build canonicalized query string
	var canonicalizedQueryString string
	for _, k := range keys {
		if canonicalizedQueryString != "" {
			canonicalizedQueryString += "&"
		}
		// Aliyun API 签名规范：对 + 和 * 和 ~ 进行特殊处理
		escapedKey := strings.Replace(url.QueryEscape(k), "+", "%20", -1)
		escapedKey = strings.Replace(escapedKey, "*", "%2A", -1)
		escapedKey = strings.Replace(escapedKey, "%7E", "~", -1)

		escapedValue := strings.Replace(url.QueryEscape(params[k]), "+", "%20", -1)
		escapedValue = strings.Replace(escapedValue, "*", "%2A", -1)
		escapedValue = strings.Replace(escapedValue, "%7E", "~", -1)

		canonicalizedQueryString += escapedKey + "=" + escapedValue
	}

	// 3. String to sign
	stringToSign := "POST" + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonicalizedQueryString)
	stringToSign = strings.Replace(stringToSign, "+", "%20", -1)
	stringToSign = strings.Replace(stringToSign, "*", "%2A", -1)
	stringToSign = strings.Replace(stringToSign, "%7E", "~", -1)

	// 4. Calculate signature
	mac := hmac.New(sha1.New, []byte(secretKey+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params["Signature"] = signature

	// 5. Build request body
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	// 6. Send request
	req, err := http.NewRequest("POST", "https://cdn.aliyuncs.com/", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Message string `json:"Message"`
		}
		json.Unmarshal(bodyBytes, &errResp)
		return fmt.Errorf("aliyun api error: %s (status: %d)", errResp.Message, resp.StatusCode)
	}

	return nil
}
