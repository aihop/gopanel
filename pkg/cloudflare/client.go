package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
		return nil, fmt.Errorf("cloudflare api error: %s (status: %d)", string(bodyBytes), resp.StatusCode)
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
		return "", fmt.Errorf("could not find zone id for domain %s", domain)
	}

	return result.Result[0].Id, nil
}
