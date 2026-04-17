package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

func GetHttpRes(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Second * 300,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			IdleConnTimeout:       15 * time.Second,
		},
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, errors.New("ErrCreateHttpClient" + err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("ErrHttpReqTimeOut" + err.Error())
		} else {
			if strings.Contains(err.Error(), "no such host") {
				return nil, errors.New("ErrNoSuchHost" + err.Error())
			}
			return nil, errors.New("ErrHttpReqFailed" + err.Error())
		}
	}
	if resp.StatusCode == 404 {
		return nil, errors.New("ErrHttpReqNotFound")
	}

	return resp, nil
}
