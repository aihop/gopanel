package huaweicloud

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// doHuaweiCloudRequest implements HuaweiCloud API v3 Signature
func doHuaweiCloudRequest(ak, sk, method, endpoint, path, query string, payload []byte) ([]byte, error) {
	t := time.Now().UTC()
	timestamp := t.Format("20060102T150405Z")

	// 1. CanonicalRequest
	canonicalURI := path
	if !strings.HasSuffix(canonicalURI, "/") {
		canonicalURI += "/"
	}

	// query 排序
	var canonicalQueryString string
	if query != "" {
		qParams := strings.Split(query, "&")
		sort.Strings(qParams)
		canonicalQueryString = strings.Join(qParams, "&")
	}

	canonicalHeaders := fmt.Sprintf("host:%s\nx-sdk-date:%s\n", endpoint, timestamp)
	signedHeaders := "host;x-sdk-date"

	hashedPayload := sha256hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, hashedPayload)

	// 2. StringToSign
	hashedCanonicalRequest := sha256hex([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("SDK-HMAC-SHA256\n%s\n%s", timestamp, hashedCanonicalRequest)

	// 3. Signature
	mac := hmac.New(sha256.New, []byte(sk))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	// 4. Authorization
	authorization := fmt.Sprintf("SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s", ak, signedHeaders, signature)

	urlStr := fmt.Sprintf("https://%s%s", endpoint, path)
	if query != "" {
		urlStr += "?" + query
	}

	req, err := http.NewRequest(method, urlStr, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;charset=utf8")
	req.Header.Set("X-Sdk-Date", timestamp)
	req.Header.Set("Authorization", authorization)

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
		return nil, fmt.Errorf("huaweicloud api http error: %s (status: %d)", string(bodyBytes), resp.StatusCode)
	}

	// 检查业务错误
	var result struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err == nil && result.ErrorCode != "" {
		return nil, fmt.Errorf("huaweicloud api error: %s (%s)", result.ErrorMsg, result.ErrorCode)
	}

	return bodyBytes, nil
}

func sha256hex(b []byte) string {
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
