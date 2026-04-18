package tencentcloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}

// doTencentCloudRequestv3 implements TencentCloud API 3.0 Signature
func doTencentCloudRequestv3(secretId, secretKey, service, version, action, host, region string, payload []byte) ([]byte, error) {
	algorithm := "TC3-HMAC-SHA256"
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	
	// ************* 步骤 1：拼接规范请求串 *************
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", "application/json; charset=utf-8", host)
	signedHeaders := "content-type;host"
	hashedRequestPayload := sha256hex(string(payload))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// ************* 步骤 2：拼接待签名字符串 *************
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// ************* 步骤 3：计算签名 *************
	secretDate := hmacsha256(date, "TC3"+secretKey)
	secretService := hmacsha256(service, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(stringToSign, secretSigning)))

	// ************* 步骤 4：拼接 Authorization *************
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		secretId,
		credentialScope,
		signedHeaders,
		signature)

	// ************* 发送请求 *************
	req, err := http.NewRequest("POST", "https://"+host, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Version", version)
	if region != "" {
		req.Header.Set("X-TC-Region", region)
	}

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
		return nil, fmt.Errorf("tencentcloud api http error: %s (status: %d)", string(bodyBytes), resp.StatusCode)
	}

	// 检查腾讯云业务错误
	var result struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err == nil && result.Response.Error.Code != "" {
		return nil, fmt.Errorf("tencentcloud api error: %s (%s)", result.Response.Error.Message, result.Response.Error.Code)
	}

	return bodyBytes, nil
}
