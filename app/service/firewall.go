package service

import (
	"fmt"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	firewallutil "github.com/aihop/gopanel/utils/firewall"
	firewallclient "github.com/aihop/gopanel/utils/firewall/client"
	"strconv"
	"strings"
)

type FirewallService struct{ client firewallutil.FirewallClient }
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
	return &dto.FirewallBaseInfo{Name: s.client.Name(), Status: status, Version: version, PingStatus: "Unknown"}, nil
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
		datas = append(datas, portOfApp{AppName: apps[i].App.Key, HttpPort: strconv.Itoa(apps[i].HttpPort), HttpsPort: strconv.Itoa(apps[i].HttpsPort)})
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
	page, limit := normalizePage(req.Page, req.Limit)
	start := (page - 1) * limit
	if start >= total {
		return &FirewallSearchResult{Items: []FirewallRuleItem{}, Total: total}, nil
	}
	if req.Type == "port" {
		apps := s.loadPortByApp()
		for i := 0; i < len(items); i++ {
			items[i].UsedStatus = checkPortUsed(items[i].Port, items[i].Protocol, apps)
		}
	}
	end := min(start+limit, total)
	return &FirewallSearchResult{Items: items[start:end], Total: total}, nil
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
	info := firewallclient.FireInfo{Address: normalizeAddress(req.Address), Port: strings.TrimSpace(req.Port), Protocol: normalizeProtocol(req.Protocol), Strategy: normalizeStrategy(req.Strategy), Description: strings.TrimSpace(req.Description)}
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
	info := firewallclient.FireInfo{Address: normalizeAddress(req.Address), Strategy: normalizeStrategy(req.Strategy), Description: strings.TrimSpace(req.Description)}
	operation := normalizeOperation(req.Operation)
	if err := s.client.RichRules(info, operation); err != nil {
		return err
	}
	return s.persistRuleDescription("ip", info, operation)
}
func (s *FirewallService) OperateForwardRule(req *dto.ForwardRuleOperate) error {
	for _, item := range req.Rules {
		if err := s.client.PortForward(firewallclient.Forward{Num: strings.TrimSpace(item.Num), Protocol: normalizeProtocol(item.Protocol), Port: strings.TrimSpace(item.Port), TargetIP: normalizeTargetIP(item.TargetIP), TargetPort: strings.TrimSpace(item.TargetPort)}, normalizeOperation(item.Operation)); err != nil {
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
	return global.DB.Model(&model.Firewall{}).Where("type = ? AND port = ? AND protocol = ? AND address = ? AND strategy = ?", normalizeRuleType(req.Type), strings.TrimSpace(req.Port), normalizeProtocol(req.Protocol), normalizeAddress(req.Address), normalizeStrategy(req.Strategy)).Updates(map[string]any{"description": strings.TrimSpace(req.Description)}).Error
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
