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

// ListCdnDomains calls Aliyun CDN API to get the list of CDN domains
func ListCdnDomains(accessKey, secretKey string) ([]string, error) {
	params := map[string]string{
		"Action":           "DescribeUserDomains",
		"Version":          "2018-05-10",
		"Format":           "JSON",
		"AccessKeyId":      accessKey,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.New().String(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"PageSize":         "50",
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalizedQueryString string
	for _, k := range keys {
		if canonicalizedQueryString != "" {
			canonicalizedQueryString += "&"
		}
		canonicalizedQueryString += url.QueryEscape(k) + "=" + url.QueryEscape(params[k])
	}

	stringToSign := "POST" + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonicalizedQueryString)

	mac := hmac.New(sha1.New, []byte(secretKey+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params["Signature"] = signature

	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	req, err := http.NewRequest("POST", "https://cdn.aliyuncs.com/", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Message string `json:"Message"`
		}
		json.Unmarshal(bodyBytes, &errResp)
		return nil, fmt.Errorf("aliyun api error: %s (status: %d)", errResp.Message, resp.StatusCode)
	}

	var result struct {
		Domains struct {
			PageData []struct {
				DomainName string `json:"DomainName"`
			} `json:"PageData"`
		} `json:"Domains"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	var domains []string
	for _, d := range result.Domains.PageData {
		domains = append(domains, d.DomainName)
	}

	return domains, nil
}
