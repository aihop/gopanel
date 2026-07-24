package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type cloudflareAPIResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Messages []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"messages"`
}

func summarizeCloudflareMessages(items []struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		msg := strings.TrimSpace(item.Message)
		if msg == "" {
			continue
		}
		if item.Code > 0 {
			parts = append(parts, fmt.Sprintf("%d:%s", item.Code, msg))
		} else {
			parts = append(parts, msg)
		}
	}
	return strings.Join(parts, "; ")
}

func formatCloudflareAPIError(path string, statusCode int, bodyBytes []byte) error {
	var apiResp cloudflareAPIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err == nil {
		errText := summarizeCloudflareMessages(apiResp.Errors)
		msgText := summarizeCloudflareMessages(apiResp.Messages)
		if errText == "" && msgText == "" {
			return fmt.Errorf("cloudflare api request failed: path=%s status=%d body=%s", path, statusCode, string(bodyBytes))
		}
		detailParts := make([]string, 0, 2)
		if errText != "" {
			detailParts = append(detailParts, "errors="+errText)
		}
		if msgText != "" {
			detailParts = append(detailParts, "messages="+msgText)
		}
		return fmt.Errorf("cloudflare api request failed: path=%s status=%d %s", path, statusCode, strings.Join(detailParts, " "))
	}
	return fmt.Errorf("cloudflare api request failed: path=%s status=%d body=%s", path, statusCode, string(bodyBytes))
}

func doCloudflareRequest(token, method, path string, payload []byte) ([]byte, error) {
	url := "https://api.cloudflare.com/client/v4" + path
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

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
		return nil, formatCloudflareAPIError(path, resp.StatusCode, bodyBytes)
	}

	var apiResp cloudflareAPIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err == nil && !apiResp.Success {
		return nil, formatCloudflareAPIError(path, resp.StatusCode, bodyBytes)
	}

	return bodyBytes, nil
}

func getZoneIdByName(token, domain string) (string, error) {
	// Cloudflare 需要通过 domain name 拿到 zone id
	path := fmt.Sprintf("/zones?name=%s", domain)
	bodyBytes, err := doCloudflareRequest(token, "GET", path, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			Id string `json:"id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if !result.Success || len(result.Result) == 0 {
		return "", fmt.Errorf("cloudflare zone lookup failed for domain %s: zone not found or token has no access", domain)
	}

	return result.Result[0].Id, nil
}
