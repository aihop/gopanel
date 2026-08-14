package docker

import (
	"errors"
	"strings"
	"testing"
)

type fakeNetworkRuntimeClient struct {
	exists      bool
	createErr   error
	createCalls int
	closed      bool
}

func (f *fakeNetworkRuntimeClient) Close() {
	f.closed = true
}

func (f *fakeNetworkRuntimeClient) NetworkExist(string) bool {
	return f.exists
}

func (f *fakeNetworkRuntimeClient) CreateNetwork(string) error {
	f.createCalls++
	return f.createErr
}

func TestEnsurePodmanNetworkWithFallbackUsesExistingAPINetwork(t *testing.T) {
	apiClient := &fakeNetworkRuntimeClient{exists: true}
	err := ensurePodmanNetworkWithFallback("gopanel-network", func(string) error {
		return errors.New(`exec: "podman": executable file not found in $PATH`)
	}, func() (networkRuntimeClient, error) {
		return apiClient, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if apiClient.createCalls != 0 || !apiClient.closed {
		t.Fatalf("createCalls=%d closed=%v", apiClient.createCalls, apiClient.closed)
	}
}

func TestEnsurePodmanNetworkWithFallbackCreatesViaAPI(t *testing.T) {
	apiClient := &fakeNetworkRuntimeClient{}
	err := ensurePodmanNetworkWithFallback("gopanel-network", func(string) error {
		return errors.New("podman CLI unavailable")
	}, func() (networkRuntimeClient, error) {
		return apiClient, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if apiClient.createCalls != 1 || !apiClient.closed {
		t.Fatalf("createCalls=%d closed=%v", apiClient.createCalls, apiClient.closed)
	}
}

func TestEnsurePodmanNetworkWithFallbackReportsBothFailures(t *testing.T) {
	err := ensurePodmanNetworkWithFallback("gopanel-network", func(string) error {
		return errors.New("podman CLI unavailable")
	}, func() (networkRuntimeClient, error) {
		return nil, errors.New("podman socket unavailable")
	})
	if err == nil || !strings.Contains(err.Error(), "podman CLI unavailable") || !strings.Contains(err.Error(), "podman socket unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
