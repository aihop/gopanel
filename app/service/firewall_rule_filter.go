package service

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/firewall"
	firewallclient "github.com/aihop/gopanel/utils/firewall/client"
	"slices"
	"strconv"
	"strings"
)

func filterFirewallItems(items []FirewallRuleItem, req *dto.RuleSearch) []FirewallRuleItem {
	info := strings.TrimSpace(strings.ToLower(req.Info))
	strategy := normalizeFilterValue(req.Strategy)
	status := normalizeFilterValue(req.Status)
	filtered := make([]FirewallRuleItem, 0, len(items))
	for _, item := range items {
		if strategy != "" && normalizeStrategy(item.Strategy) != strategy {
			continue
		}
		if status != "" && normalizeUsedStatus(item.UsedStatus) != status {
			continue
		}
		if info != "" {
			fields := []string{item.Port, item.Protocol, item.Address, item.TargetIP, item.TargetPort, item.Description, item.Strategy}
			if !containsAny(fields, info) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	slices.SortFunc(filtered, func(a, b FirewallRuleItem) int {
		return strings.Compare(a.ID, b.ID)
	})
	return filtered
}
func containsAny(values []string, query string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
func normalizeRuleType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "ip":
		return "ip"
	case "forward":
		return "forward"
	default:
		return "port"
	}
}
func normalizeStrategy(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "allow":
		return "accept"
	case "deny", "reject":
		return "drop"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}
func normalizeUsedStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "inused", "used", "10", "已使用":
		return "used"
	case "unused", "20", "未使用":
		return "unused"
	case "":
		return "unknown"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}
func normalizeFilterValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "all" || value == "0" {
		return ""
	}
	if value == "10" {
		return "used"
	}
	if value == "20" {
		return "unused"
	}
	if value == "reject" || value == "deny" {
		return "drop"
	}
	if value == "allow" {
		return "accept"
	}
	return value
}
func normalizeProtocol(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "tcp"
	}
	return value
}
func normalizeAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "(v6)", "")
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if lower == "anywhere" {
		return ""
	}
	if lower == "0.0.0.0/0" || lower == "::/0" {
		return ""
	}
	return value
}
func normalizeOperation(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "delete", "del", "remove":
		return "remove"
	default:
		return "add"
	}
}
func normalizeTargetIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "127.0.0.1"
	}
	return value
}
func normalizeFamily(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "ipv4"
	}
	return value
}
func normalizePage(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return page, limit
}
func buildFirewallDescriptionKey(ruleType, port, protocol, address, strategy string) string {
	return strings.Join([]string{normalizeRuleType(ruleType), strings.TrimSpace(port), normalizeProtocol(protocol), normalizeAddress(address), normalizeStrategy(strategy)}, "|")
}
func buildPortRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{"port", strings.TrimSpace(rule.Port), normalizeProtocol(rule.Protocol), normalizeAddress(rule.Address), normalizeStrategy(rule.Strategy)}, "|")
}
func buildAddressRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{"ip", strings.TrimSpace(rule.Address), normalizeStrategy(rule.Strategy)}, "|")
}
func buildForwardRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{"forward", strings.TrimSpace(rule.Num), strings.TrimSpace(rule.Port), normalizeProtocol(rule.Protocol), normalizeTargetIP(rule.TargetIP), strings.TrimSpace(rule.TargetPort)}, "|")
}
func loadFirewallDescriptions(ruleType string) (map[string]string, error) {
	var items []model.Firewall
	if err := global.DB.Where("type = ?", normalizeRuleType(ruleType)).Find(&items).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[buildFirewallDescriptionKey(item.Type, item.Port, item.Protocol, item.Address, item.Strategy)] = item.Description
	}
	return result, nil
}
func (s *FirewallService) persistRuleDescription(ruleType string, rule firewallclient.FireInfo, operation string) error {
	query := global.DB.Where("type = ? AND port = ? AND protocol = ? AND address = ? AND strategy = ?", normalizeRuleType(ruleType), strings.TrimSpace(rule.Port), normalizeProtocol(rule.Protocol), normalizeAddress(rule.Address), normalizeStrategy(rule.Strategy))
	if operation == "remove" {
		return query.Delete(&model.Firewall{}).Error
	}
	var existing model.Firewall
	if err := query.First(&existing).Error; err == nil {
		return global.DB.Model(&existing).Updates(map[string]any{"description": strings.TrimSpace(rule.Description)}).Error
	}
	return global.DB.Create(&model.Firewall{Type: normalizeRuleType(ruleType), Port: strings.TrimSpace(rule.Port), Protocol: normalizeProtocol(rule.Protocol), Address: strings.TrimSpace(rule.Address), Strategy: normalizeStrategy(rule.Strategy), Description: strings.TrimSpace(rule.Description)}).Error
}
func ensureFirewallTables() error {
	return global.DB.AutoMigrate(&model.Firewall{}, &model.Forward{})
}
func OperateFirewallPort(oldPorts, newPorts []int) error {
	client, err := firewall.NewFirewallClient()
	if err != nil {
		return err
	}
	for _, port := range newPorts {
		if err := client.Port(firewallclient.FireInfo{Port: strconv.Itoa(port), Protocol: "tcp", Strategy: "accept"}, "add"); err != nil {
			return err
		}
	}
	for _, port := range oldPorts {
		if err := client.Port(firewallclient.FireInfo{Port: strconv.Itoa(port), Protocol: "tcp", Strategy: "accept"}, "remove"); err != nil {
			return err
		}
	}
	return client.Reload()
}
