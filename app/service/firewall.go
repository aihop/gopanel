package service

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/firewall"
	firewallutil "github.com/aihop/gopanel/utils/firewall"
	firewallclient "github.com/aihop/gopanel/utils/firewall/client"
)

type FirewallService struct {
	client firewallutil.FirewallClient
}

type FirewallSearchResult struct {
	Items []FirewallRuleItem `json:"items"`
	Total int                `json:"total"`
}

type FirewallRuleItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Num         string `json:"num"`
	Family      string `json:"family"`
	Address     string `json:"address"`
	Destination string `json:"destination"`
	Port        string `json:"port"`
	SrcPort     string `json:"srcPort"`
	DestPort    string `json:"destPort"`
	TargetIP    string `json:"targetIP"`
	TargetPort  string `json:"targetPort"`
	Protocol    string `json:"protocol"`
	Strategy    string `json:"strategy"`
	UsedStatus  string `json:"usedStatus"`
	Description string `json:"description"`
}

func NewFirewall() (*FirewallService, error) {
	client, err := firewallutil.NewFirewallClient()
	if err != nil {
		return nil, err
	}
	if err := ensureFirewallTables(); err != nil {
		return nil, err
	}
	return &FirewallService{client: client}, nil
}

func (s *FirewallService) Base() (*dto.FirewallBaseInfo, error) {
	status, err := s.client.Status()
	if err != nil {
		return nil, err
	}
	version, err := s.client.Version()
	if err != nil {
		return nil, err
	}
	return &dto.FirewallBaseInfo{
		Name:       s.client.Name(),
		Status:     status,
		Version:    version,
		PingStatus: "Unknown",
	}, nil
}

type portOfApp struct {
	AppName   string
	HttpPort  string
	HttpsPort string
}

func (u *FirewallService) loadPortByApp() []portOfApp {
	var datas []portOfApp
	apps, err := appInstallRepo.ListBy()
	if err != nil {
		return datas
	}
	for i := 0; i < len(apps); i++ {
		datas = append(datas, portOfApp{
			AppName:   apps[i].App.Key,
			HttpPort:  strconv.Itoa(apps[i].HttpPort),
			HttpsPort: strconv.Itoa(apps[i].HttpsPort),
		})
	}
	systemPort, err := settingRepo.Get(settingRepo.WithByKey("ServerPort"))
	if err != nil {
		return datas
	}
	datas = append(datas, portOfApp{AppName: "gopanel", HttpPort: systemPort.Value})
	return datas
}

func (s *FirewallService) Search(req *dto.RuleSearch) (*FirewallSearchResult, error) {
	ruleType := normalizeRuleType(req.Type)
	items, err := s.loadRuleItems(ruleType)
	if err != nil {
		return nil, err
	}
	items = filterFirewallItems(items, req)
	total := len(items)
	page, pageSize := normalizePage(req.Page, req.PageSize)
	start := (page - 1) * pageSize
	if start >= total {
		return &FirewallSearchResult{Items: []FirewallRuleItem{}, Total: total}, nil
	}

	if req.Type == "port" {
		apps := s.loadPortByApp()
		for i := 0; i < len(items); i++ {
			items[i].UsedStatus = checkPortUsed(items[i].Port, items[i].Protocol, apps)
		}
	}
	end := min(start+pageSize, total)
	return &FirewallSearchResult{
		Items: items[start:end],
		Total: total,
	}, nil
}

func (s *FirewallService) Operate(req *dto.FirewallOperation) error {
	switch req.Operation {
	case "start":
		return s.client.Start()
	case "stop":
		return s.client.Stop()
	case "restart":
		return s.client.Restart()
	case "disablePing", "enablePing":
		return fmt.Errorf("当前版本暂未支持 Ping 开关")
	default:
		return fmt.Errorf("不支持的操作: %s", req.Operation)
	}
}

func (s *FirewallService) OperatePortRule(req *dto.PortRuleOperate) error {
	info := firewallclient.FireInfo{
		Address:     normalizeAddress(req.Address),
		Port:        strings.TrimSpace(req.Port),
		Protocol:    normalizeProtocol(req.Protocol),
		Strategy:    normalizeStrategy(req.Strategy),
		Description: strings.TrimSpace(req.Description),
	}
	operation := normalizeOperation(req.Operation)
	if info.Address != "" {
		if err := s.client.RichRules(info, operation); err != nil {
			return err
		}
	} else if info.Strategy == "drop" && s.client.Name() == "firewalld" {
		if err := s.client.RichRules(info, operation); err != nil {
			return err
		}
	} else {
		if err := s.client.Port(info, operation); err != nil {
			return err
		}
	}
	return s.persistRuleDescription("port", info, operation)
}

func (s *FirewallService) OperateIPRule(req *dto.AddrRuleOperate) error {
	info := firewallclient.FireInfo{
		Address:     normalizeAddress(req.Address),
		Strategy:    normalizeStrategy(req.Strategy),
		Description: strings.TrimSpace(req.Description),
	}
	operation := normalizeOperation(req.Operation)
	if err := s.client.RichRules(info, operation); err != nil {
		return err
	}
	return s.persistRuleDescription("ip", info, operation)
}

func (s *FirewallService) OperateForwardRule(req *dto.ForwardRuleOperate) error {
	for _, item := range req.Rules {
		if err := s.client.PortForward(firewallclient.Forward{
			Num:        strings.TrimSpace(item.Num),
			Protocol:   normalizeProtocol(item.Protocol),
			Port:       strings.TrimSpace(item.Port),
			TargetIP:   normalizeTargetIP(item.TargetIP),
			TargetPort: strings.TrimSpace(item.TargetPort),
		}, normalizeOperation(item.Operation)); err != nil {
			return err
		}
	}
	return nil
}

func (s *FirewallService) UpdatePortRule(req *dto.PortRuleUpdate) error {
	oldRule := req.OldRule
	oldRule.Operation = "remove"
	if err := s.OperatePortRule(&oldRule); err != nil {
		return err
	}
	newRule := req.NewRule
	newRule.Operation = "add"
	return s.OperatePortRule(&newRule)
}

func (s *FirewallService) UpdateAddrRule(req *dto.AddrRuleUpdate) error {
	oldRule := req.OldRule
	oldRule.Operation = "remove"
	if err := s.OperateIPRule(&oldRule); err != nil {
		return err
	}
	newRule := req.NewRule
	newRule.Operation = "add"
	return s.OperateIPRule(&newRule)
}

func (s *FirewallService) UpdateDescription(req *dto.UpdateFirewallDescription) error {
	return global.DB.Model(&model.Firewall{}).
		Where("type = ? AND port = ? AND protocol = ? AND address = ? AND strategy = ?", normalizeRuleType(req.Type), strings.TrimSpace(req.Port), normalizeProtocol(req.Protocol), normalizeAddress(req.Address), normalizeStrategy(req.Strategy)).
		Updates(map[string]any{"description": strings.TrimSpace(req.Description)}).Error
}

func (s *FirewallService) BatchOperate(req *dto.BatchRuleOperate) error {
	for _, item := range req.Rules {
		item.Operation = "remove"
		if err := s.OperatePortRule(&item); err != nil {
			return err
		}
	}
	return nil
}

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
		rule := FirewallRuleItem{
			ID:          buildPortRuleID(item),
			Type:        "port",
			Family:      normalizeFamily(item.Family),
			Address:     addrDisplay,
			Port:        strings.TrimSpace(item.Port),
			Protocol:    normalizeProtocol(item.Protocol),
			Strategy:    normalizeStrategy(item.Strategy),
			UsedStatus:  normalizeUsedStatus(item.UsedStatus),
			Description: descMap[buildFirewallDescriptionKey("port", item.Port, item.Protocol, item.Address, item.Strategy)],
		}
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
		fireInfo := firewallclient.FireInfo{
			Port:     item.Port,
			Protocol: item.Protocol,
			Address:  addrNormalized,
			Strategy: item.Strategy,
		}
		id := buildPortRuleID(fireInfo)
		if _, ok := exists[id]; ok {
			continue
		}
		items = append(items, FirewallRuleItem{
			ID:          id,
			Type:        "port",
			Family:      "ipv4",
			Address:     addrDisplay,
			Port:        item.Port,
			Protocol:    normalizeProtocol(item.Protocol),
			Strategy:    normalizeStrategy(item.Strategy),
			UsedStatus:  "unknown",
			Description: item.Description,
		})
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
		items = append(items, FirewallRuleItem{
			ID:          buildAddressRuleID(item),
			Type:        "ip",
			Family:      normalizeFamily(item.Family),
			Address:     strings.TrimSpace(item.Address),
			Protocol:    normalizeProtocol(item.Protocol),
			Port:        strings.TrimSpace(item.Port),
			Strategy:    normalizeStrategy(item.Strategy),
			Description: descMap[buildFirewallDescriptionKey("ip", "", "", item.Address, item.Strategy)],
			UsedStatus:  normalizeUsedStatus(item.UsedStatus),
		})
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
		items = append(items, FirewallRuleItem{
			ID:         buildForwardRuleID(item),
			Type:       "forward",
			Num:        strings.TrimSpace(item.Num),
			Port:       strings.TrimSpace(item.Port),
			Protocol:   normalizeProtocol(item.Protocol),
			TargetIP:   normalizeTargetIP(item.TargetIP),
			TargetPort: strings.TrimSpace(item.TargetPort),
		})
	}
	return items, nil
}

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
			fields := []string{
				item.Port,
				item.Protocol,
				item.Address,
				item.TargetIP,
				item.TargetPort,
				item.Description,
				item.Strategy,
			}
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

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}

func buildFirewallDescriptionKey(ruleType, port, protocol, address, strategy string) string {
	return strings.Join([]string{
		normalizeRuleType(ruleType),
		strings.TrimSpace(port),
		normalizeProtocol(protocol),
		normalizeAddress(address),
		normalizeStrategy(strategy),
	}, "|")
}

func buildPortRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{
		"port",
		strings.TrimSpace(rule.Port),
		normalizeProtocol(rule.Protocol),
		normalizeAddress(rule.Address),
		normalizeStrategy(rule.Strategy),
	}, "|")
}

func buildAddressRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{
		"ip",
		strings.TrimSpace(rule.Address),
		normalizeStrategy(rule.Strategy),
	}, "|")
}

func buildForwardRuleID(rule firewallclient.FireInfo) string {
	return strings.Join([]string{
		"forward",
		strings.TrimSpace(rule.Num),
		strings.TrimSpace(rule.Port),
		normalizeProtocol(rule.Protocol),
		normalizeTargetIP(rule.TargetIP),
		strings.TrimSpace(rule.TargetPort),
	}, "|")
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
	query := global.DB.Where("type = ? AND port = ? AND protocol = ? AND address = ? AND strategy = ?",
		normalizeRuleType(ruleType),
		strings.TrimSpace(rule.Port),
		normalizeProtocol(rule.Protocol),
		normalizeAddress(rule.Address),
		normalizeStrategy(rule.Strategy),
	)

	if operation == "remove" {
		return query.Delete(&model.Firewall{}).Error
	}

	var existing model.Firewall
	if err := query.First(&existing).Error; err == nil {
		return global.DB.Model(&existing).Updates(map[string]any{
			"description": strings.TrimSpace(rule.Description),
		}).Error
	}

	return global.DB.Create(&model.Firewall{
		Type:        normalizeRuleType(ruleType),
		Port:        strings.TrimSpace(rule.Port),
		Protocol:    normalizeProtocol(rule.Protocol),
		Address:     strings.TrimSpace(rule.Address),
		Strategy:    normalizeStrategy(rule.Strategy),
		Description: strings.TrimSpace(rule.Description),
	}).Error
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
