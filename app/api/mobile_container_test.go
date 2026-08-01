package api

import (
	"testing"

	"github.com/aihop/gopanel/app/dto"
)

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
