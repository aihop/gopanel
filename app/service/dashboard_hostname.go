package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	ErrHostnameInvalid         = errors.New("ErrHostnameInvalid")
	ErrHostnameToolUnavailable = errors.New("ErrHostnameToolUnavailable")
	ErrHostnameUpdateFailed    = errors.New("ErrHostnameUpdateFailed")
)

func (u *DashboardService) UpdateHostname(hostname string) (string, error) {
	hostname = strings.TrimSpace(hostname)
	if err := validateHostname(hostname); err != nil {
		return "", err
	}

	hostnamectl, err := exec.LookPath("hostnamectl")
	if err != nil {
		return "", ErrHostnameToolUnavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, hostnamectl, "set-hostname", hostname).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrHostnameUpdateFailed, strings.TrimSpace(string(output)))
	}

	actualHostname, err := os.Hostname()
	if err != nil || actualHostname == "" {
		return hostname, nil
	}
	return actualHostname, nil
}

func validateHostname(hostname string) error {
	if hostname == "" || len(hostname) > 253 {
		return ErrHostnameInvalid
	}
	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return ErrHostnameInvalid
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' {
				return ErrHostnameInvalid
			}
		}
	}
	return nil
}
