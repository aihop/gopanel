package client

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

func validateFirewallOperation(operation string) error {
	if operation != "add" && operation != "remove" {
		return fmt.Errorf("unsupported firewall operation %q", operation)
	}
	return nil
}

func validateFirewallProtocol(protocol string, optional bool) error {
	if optional && protocol == "" {
		return nil
	}
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("unsupported firewall protocol %q", protocol)
	}
	return nil
}

func validateFirewallPort(value string, optional bool) error {
	value = strings.TrimSpace(value)
	if optional && value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '-' || r == ':' })
	if len(parts) < 1 || len(parts) > 2 {
		return fmt.Errorf("invalid firewall port %q", value)
	}
	for _, part := range parts {
		port, err := strconv.Atoi(part)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid firewall port %q", value)
		}
	}
	return nil
}

func validateFirewallAddress(value string, optional bool) error {
	value = strings.TrimSpace(value)
	if optional && value == "" {
		return nil
	}
	for _, part := range strings.Split(value, "-") {
		if _, err := netip.ParseAddr(part); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(part); err != nil {
			return fmt.Errorf("invalid firewall address %q", value)
		}
	}
	return nil
}

func validateFireInfo(rule FireInfo, operation string) error {
	if err := validateFirewallOperation(operation); err != nil {
		return err
	}
	if err := validateFirewallAddress(rule.Address, true); err != nil {
		return err
	}
	if err := validateFirewallPort(rule.Port, true); err != nil {
		return err
	}
	if err := validateFirewallProtocol(rule.Protocol, true); err != nil {
		return err
	}
	if rule.Strategy != "accept" && rule.Strategy != "drop" {
		return fmt.Errorf("unsupported firewall strategy %q", rule.Strategy)
	}
	return nil
}

func validateForward(info Forward, operation string) error {
	if err := validateFirewallOperation(operation); err != nil {
		return err
	}
	if err := validateFirewallProtocol(info.Protocol, false); err != nil {
		return err
	}
	if err := validateFirewallPort(info.Port, false); err != nil {
		return err
	}
	if err := validateFirewallPort(info.TargetPort, false); err != nil {
		return err
	}
	if info.TargetIP != "" && info.TargetIP != "localhost" {
		if err := validateFirewallAddress(info.TargetIP, false); err != nil {
			return err
		}
	}
	return nil
}
