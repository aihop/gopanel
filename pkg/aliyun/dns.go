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

func doAlidnsRequest(accessKey, secretKey string, action string, extraParams map[string]string) error {
	params := map[string]string{
		"Action":           action,
		"Version":          "2015-01-09",
		"Format":           "JSON",
		"AccessKeyId":      accessKey,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.New().String(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}

	for k, v := range extraParams {
		params[k] = v
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
		// Aliyun API 签名规范：对 + 和 * 和 ~ 进行特殊处理
		escapedKey := strings.Replace(url.QueryEscape(k), "+", "%20", -1)
		escapedKey = strings.Replace(escapedKey, "*", "%2A", -1)
		escapedKey = strings.Replace(escapedKey, "%7E", "~", -1)

		escapedValue := strings.Replace(url.QueryEscape(params[k]), "+", "%20", -1)
		escapedValue = strings.Replace(escapedValue, "*", "%2A", -1)
		escapedValue = strings.Replace(escapedValue, "%7E", "~", -1)

		canonicalizedQueryString += escapedKey + "=" + escapedValue
	}

	stringToSign := "POST" + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonicalizedQueryString)
	stringToSign = strings.Replace(stringToSign, "+", "%20", -1)
	stringToSign = strings.Replace(stringToSign, "*", "%2A", -1)
	stringToSign = strings.Replace(stringToSign, "%7E", "~", -1)

	mac := hmac.New(sha1.New, []byte(secretKey+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params["Signature"] = signature

	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	req, err := http.NewRequest("POST", "https://alidns.aliyuncs.com/", strings.NewReader(data.Encode()))
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
		return fmt.Errorf("aliyun dns api error: %s (status: %d)", errResp.Message, resp.StatusCode)
	}

	return nil
}

// AddTxtRecord calls Aliyun DNS API to add a TXT record
func AddTxtRecord(accessKey, secretKey, domain, rr, value string) error {
	params := map[string]string{
		"DomainName": domain,
		"RR":         rr,
		"Type":       "TXT",
		"Value":      value,
	}
	return doAlidnsRequest(accessKey, secretKey, "AddDomainRecord", params)
}

// DeleteSubDomainRecords calls Aliyun DNS API to delete records of a subdomain
func DeleteSubDomainRecords(accessKey, secretKey, domain, rr string) error {
	params := map[string]string{
		"DomainName": domain,
		"RR":         rr,
		"Type":       "TXT",
	}
	return doAlidnsRequest(accessKey, secretKey, "DeleteSubDomainRecords", params)
}
