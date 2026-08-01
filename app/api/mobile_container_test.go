package api

import (
	"errors"
	"testing"

	"github.com/aihop/gopanel/app/dto"
)

type mobileContainerListReaderStub struct {
	total    int64
	items    interface{}
	pageErr  error
	stats    []dto.ContainerListStats
	statsErr error
}

func (stub mobileContainerListReaderStub) Page(*dto.PageContainer) (int64, interface{}, error) {
	return stub.total, stub.items, stub.pageErr
}

func (stub mobileContainerListReaderStub) ContainerListStats() ([]dto.ContainerListStats, error) {
	return stub.stats, stub.statsErr
}

func TestMobileContainerOperationAllowlist(t *testing.T) {
	for _, operation := range []string{"start", "stop", "restart"} {
		if !isMobileContainerOperationAllowed(operation) {
			t.Fatalf("expected %q to be allowed", operation)
		}
	}
	for _, operation := range []string{"remove", "kill", "pause", "up", ""} {
		if isMobileContainerOperationAllowed(operation) {
			t.Fatalf("expected %q to be rejected", operation)
		}
	}
}

func TestMobilePublishedTCPPorts(t *testing.T) {
	ports := mobilePublishedTCPPorts([]dto.PortHelper{
		{HostPort: "18080", ContainerPort: "80", Protocol: "tcp"},
		{HostPort: "1443", ContainerPort: "443", Protocol: "TCP"},
		{HostPort: "15353", ContainerPort: "53", Protocol: "udp"},
		{HostPort: "18080", ContainerPort: "8080", Protocol: "tcp"},
		{HostPort: "", ContainerPort: "3000", Protocol: "tcp"},
	})
	if len(ports) != 2 {
		t.Fatalf("expected two published TCP ports, got %#v", ports)
	}
	if ports[0].HostPort != 1443 || ports[0].ContainerPort != "443" {
		t.Fatalf("expected sorted port 1443 -> 443, got %#v", ports[0])
	}
	if ports[1].HostPort != 18080 || ports[1].ContainerPort != "80" {
		t.Fatalf("expected deduplicated port 18080 -> 80, got %#v", ports[1])
	}
}

func TestLoadMobileContainerListDegradesWhenStatsFail(t *testing.T) {
	data, err := loadMobileContainerList(mobileContainerListReaderStub{
		total: 1,
		items: []dto.ContainerInfo{{
			ContainerID: "container-1",
			Name:        "web",
			State:       "running",
			Ports:       []string{"18080->80/tcp"},
		}},
		statsErr: errors.New("stats unavailable"),
	})
	if err != nil {
		t.Fatalf("expected container list to survive stats failure: %v", err)
	}
	items, ok := data["items"].([]mobileContainerSummary)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one container summary, got %#v", data["items"])
	}
	if items[0].ContainerID != "container-1" || items[0].CPUPercent != 0 || items[0].MemoryUsage != 0 {
		t.Fatalf("expected zero-value stats on the container summary, got %#v", items[0])
	}
	if data["running"] != 1 || data["stopped"] != 0 {
		t.Fatalf("expected container counts to remain available, got %#v", data)
	}
}
