package service

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	firewallclient "github.com/aihop/gopanel/utils/firewall/client"
	"strings"
)

func (s *FirewallService) loadRuleItems(ruleType string) ([]FirewallRuleItem, error) {
	switch ruleType {
	case "ip":
		return s.loadAddressRules()
	case "forward":
		return s.loadForwardRules()
	default:
		return s.loadPortRules()
	}
}
func (s *FirewallService) loadPortRules() ([]FirewallRuleItem, error) {
	list, err := s.client.ListPort()
	if err != nil {
		return nil, err
	}
	descMap, err := loadFirewallDescriptions("port")
	if err != nil {
		return nil, err
	}
	items := make([]FirewallRuleItem, 0, len(list))
	exists := make(map[string]struct{})
	for _, item := range list {
		addrNormalized := normalizeAddress(item.Address)
		addrDisplay := addrNormalized
		if addrDisplay == "" {
			addrDisplay = "Anywhere"
		}
		rule := FirewallRuleItem{ID: buildPortRuleID(item), Type: "port", Family: normalizeFamily(item.Family), Address: addrDisplay, Port: strings.TrimSpace(item.Port), Protocol: normalizeProtocol(item.Protocol), Strategy: normalizeStrategy(item.Strategy), UsedStatus: normalizeUsedStatus(item.UsedStatus), Description: descMap[buildFirewallDescriptionKey("port", item.Port, item.Protocol, item.Address, item.Strategy)]}
		if rule.Strategy == "" {
			rule.Strategy = "accept"
		}
		exists[rule.ID] = struct{}{}
		items = append(items, rule)
	}
	var persisted []model.Firewall
	if err = global.DB.Where("type = ?", "port").Find(&persisted).Error; err != nil {
		return nil, err
	}
	for _, item := range persisted {
		addrNormalized := normalizeAddress(item.Address)
		addrDisplay := addrNormalized
		if addrDisplay == "" {
			addrDisplay = "Anywhere"
		}
		fireInfo := firewallclient.FireInfo{Port: item.Port, Protocol: item.Protocol, Address: addrNormalized, Strategy: item.Strategy}
		id := buildPortRuleID(fireInfo)
		if _, ok := exists[id]; ok {
			continue
		}
		items = append(items, FirewallRuleItem{ID: id, Type: "port", Family: "ipv4", Address: addrDisplay, Port: item.Port, Protocol: normalizeProtocol(item.Protocol), Strategy: normalizeStrategy(item.Strategy), UsedStatus: "unknown", Description: item.Description})
	}
	return items, nil
}
func (s *FirewallService) loadAddressRules() ([]FirewallRuleItem, error) {
	list, err := s.client.ListAddress()
	if err != nil {
		return nil, err
	}
	descMap, err := loadFirewallDescriptions("ip")
	if err != nil {
		return nil, err
	}
	items := make([]FirewallRuleItem, 0, len(list))
	for _, item := range list {
		items = append(items, FirewallRuleItem{ID: buildAddressRuleID(item), Type: "ip", Family: normalizeFamily(item.Family), Address: strings.TrimSpace(item.Address), Protocol: normalizeProtocol(item.Protocol), Port: strings.TrimSpace(item.Port), Strategy: normalizeStrategy(item.Strategy), Description: descMap[buildFirewallDescriptionKey("ip", "", "", item.Address, item.Strategy)], UsedStatus: normalizeUsedStatus(item.UsedStatus)})
	}
	return items, nil
}
func (s *FirewallService) loadForwardRules() ([]FirewallRuleItem, error) {
	list, err := s.client.ListForward()
	if err != nil {
		return nil, err
	}
	items := make([]FirewallRuleItem, 0, len(list))
	for _, item := range list {
		items = append(items, FirewallRuleItem{ID: buildForwardRuleID(item), Type: "forward", Num: strings.TrimSpace(item.Num), Port: strings.TrimSpace(item.Port), Protocol: normalizeProtocol(item.Protocol), TargetIP: normalizeTargetIP(item.TargetIP), TargetPort: strings.TrimSpace(item.TargetPort)})
	}
	return items, nil
}
