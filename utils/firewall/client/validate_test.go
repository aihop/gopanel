package client

import "testing"

func TestFirewallValidationRejectsShellInput(t *testing.T) {
	badRules := []FireInfo{
		{Port: "80;id", Protocol: "tcp", Strategy: "accept"},
		{Port: "80", Protocol: "tcp$(id)", Strategy: "accept"},
		{Address: "127.0.0.1\nadd rule", Strategy: "drop"},
	}
	for _, rule := range badRules {
		if err := validateFireInfo(rule, "add"); err == nil {
			t.Fatalf("expected rule to be rejected: %#v", rule)
		}
	}
}

func TestFirewallValidationAllowsSemanticValues(t *testing.T) {
	rule := FireInfo{Address: "192.0.2.0/24", Port: "8000-8010", Protocol: "tcp", Strategy: "accept"}
	if err := validateFireInfo(rule, "add"); err != nil {
		t.Fatalf("expected valid rule: %v", err)
	}
}
