package aws

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// AWS Signature Version 4 (SigV4) Implementation

func hmacSHA256(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// doAWSSigV4Request signs and executes an AWS API request
func doAWSSigV4Request(accessKey, secretKey, region, service, method, host, path, query string, payload []byte, extraHeaders map[string]string) ([]byte, error) {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102") // YYYYMMDD

	// Create canonical URI
	canonicalURI := path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	// Create canonical query string
	var canonicalQueryString string
	if query != "" {
		qParams := strings.Split(query, "&")
		sort.Strings(qParams)
		canonicalQueryString = strings.Join(qParams, "&")
	}

	// Create canonical headers
	headers := map[string]string{
		"host":         host,
		"x-amz-date":   amzDate,
	}
	for k, v := range extraHeaders {
		headers[strings.ToLower(k)] = v
	}

	var headerKeys []string
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)

	var canonicalHeaders string
	var signedHeaders string
	for _, k := range headerKeys {
		canonicalHeaders += fmt.Sprintf("%s:%s\n", k, headers[k])
		if signedHeaders != "" {
			signedHeaders += ";"
		}
		signedHeaders += k
	}

	// Create payload hash
	payloadHash := sha256Hex(payload)

	// Create canonical request
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, payloadHash)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)))

	// Calculate signature
	signingKey := getSignatureKey(secretKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Create authorization header
	authorizationHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, accessKey, credentialScope, signedHeaders, signature)

	// Prepare HTTP request
	urlStr := fmt.Sprintf("https://%s%s", host, path)
	if query != "" {
		urlStr += "?" + query
	}

	req, err := http.NewRequest(method, urlStr, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Host", host)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	// Execute request
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
		return nil, fmt.Errorf("aws api http error: %s (status: %d)", string(bodyBytes), resp.StatusCode)
	}

	return bodyBytes, nil
}
