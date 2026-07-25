package service

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/utils/encrypt"
)

// 节点状态取值
const (
	NodeStatusOnline       = "online"
	NodeStatusOffline      = "offline"
	NodeStatusUnauthorized = "unauthorized"
	NodeStatusUnknown      = "unknown"
)

// 告警阈值
const (
	diskWarnPercent   = 85.0
	diskDangerPercent = 90.0
	certDangerDays    = 7
)

// NodeTokenLength 节点只读令牌的固定长度。
// 签发和校验共用这个值，前端也据此判断用户粘贴的字符串是否完整——
// 长度不对几乎总是"复制漏了"或"填的不是节点签发的串"，在保存前就该拦住。
const NodeTokenLength = 40

// clearTokenMarker 更新时传这个值表示"清除该令牌"。
// 不能用空串表达清除——空串已经被用来表示"不修改"（前端拿不到明文，只能留空提交）。
const clearTokenMarker = "-"

// encryptOptionalToken 可选令牌的加密：空串原样返回，不做加密
func encryptOptionalToken(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == clearTokenMarker {
		return "", nil
	}
	return encrypt.StringEncrypt(value)
}

type NodeService struct{}

func NewNode() *NodeService {
	return &NodeService{}
}

func (s *NodeService) List() ([]dto.NodeRes, error) {
	nodes, err := repo.NewNode().List()
	if err != nil {
		return nil, err
	}
	list := make([]dto.NodeRes, 0, len(nodes))
	for _, node := range nodes {
		list = append(list, toNodeRes(node))
	}
	return list, nil
}

func (s *NodeService) Create(req dto.NodeCreateReq) error {
	addr, err := normalizeNodeAddr(req.Addr)
	if err != nil {
		return err
	}
	exist, err := repo.NewNode().CountByAddr(addr, 0)
	if err != nil {
		return err
	}
	if exist > 0 {
		return errors.New("该节点地址已存在")
	}
	cipherToken, err := encrypt.StringEncrypt(strings.TrimSpace(req.AccessToken))
	if err != nil {
		return err
	}
	cipherControl, err := encryptOptionalToken(req.ControlToken)
	if err != nil {
		return err
	}
	node := &model.Node{
		Name:         strings.TrimSpace(req.Name),
		Addr:         addr,
		Entrance:     strings.TrimSpace(req.Entrance),
		AccessToken:  cipherToken,
		ControlToken: cipherControl,
		ConnectMode:  "direct",
		SkipVerify:   req.SkipVerify,
		IsProd:       req.IsProd,
		Sort:         req.Sort,
		Status:       NodeStatusUnknown,
	}
	return repo.NewNode().Create(node)
}

func (s *NodeService) Update(req dto.NodeUpdateReq) error {
	node, err := repo.NewNode().GetByID(req.ID)
	if err != nil {
		return err
	}
	addr, err := normalizeNodeAddr(req.Addr)
	if err != nil {
		return err
	}
	exist, err := repo.NewNode().CountByAddr(addr, req.ID)
	if err != nil {
		return err
	}
	if exist > 0 {
		return errors.New("该节点地址已存在")
	}

	node.Name = strings.TrimSpace(req.Name)
	node.Addr = addr
	node.Entrance = strings.TrimSpace(req.Entrance)
	node.SkipVerify = req.SkipVerify
	node.IsProd = req.IsProd
	node.Sort = req.Sort
	// 令牌留空表示保留原值，避免前端因为拿不到明文而误清空
	if token := strings.TrimSpace(req.AccessToken); token != "" {
		cipherToken, err := encrypt.StringEncrypt(token)
		if err != nil {
			return err
		}
		node.AccessToken = cipherToken
	}
	// 控制令牌：留空=不改，"-"=清除（关闭该节点的操作能力）
	switch control := strings.TrimSpace(req.ControlToken); control {
	case "":
	case clearTokenMarker:
		node.ControlToken = ""
	default:
		cipherControl, err := encrypt.StringEncrypt(control)
		if err != nil {
			return err
		}
		node.ControlToken = cipherControl
	}
	return repo.NewNode().Save(&node)
}

func (s *NodeService) Delete(id uint) error {
	if _, err := repo.NewNode().GetByID(id); err != nil {
		return err
	}
	return repo.NewNode().DeleteByID(id)
}

// Probe 立即探测单个节点，用于列表页的手动采集和编辑态的测试连接。
// 采集失败必须把错误返回出去——之前这里丢掉了 CollectNode 的错误，
// 导致节点明明连不上，接口却回成功、前端提示"测试成功"。
func (s *NodeService) Probe(id uint) (dto.NodeRes, error) {
	if _, err := repo.NewNode().GetByID(id); err != nil {
		return dto.NodeRes{}, err
	}
	collectErr := CollectNode(id)
	refreshed, err := repo.NewNode().GetByID(id)
	if err != nil {
		return dto.NodeRes{}, err
	}
	// 即使失败也把最新的行返回出去，前端可以顺手更新状态列
	return toNodeRes(refreshed), collectErr
}

// ProbeDraft 校验尚未保存的节点配置，供新增弹窗里的“测试连接”使用
func (s *NodeService) ProbeDraft(req dto.NodeCreateReq) (dto.NodeTokenRes, error) {
	addr, err := normalizeNodeAddr(req.Addr)
	if err != nil {
		return dto.NodeTokenRes{}, err
	}
	cipherToken, err := encrypt.StringEncrypt(strings.TrimSpace(req.AccessToken))
	if err != nil {
		return dto.NodeTokenRes{}, err
	}
	return ProbeNode(model.Node{
		Addr:        addr,
		Entrance:    strings.TrimSpace(req.Entrance),
		AccessToken: cipherToken,
		SkipVerify:  req.SkipVerify,
	})
}

func toNodeRes(node model.Node) dto.NodeRes {
	tokenLen := 0
	if plain, err := encrypt.StringDecrypt(node.AccessToken); err == nil {
		tokenLen = len(plain)
	}
	controlLen := 0
	if plain, err := encrypt.StringDecrypt(node.ControlToken); err == nil {
		controlLen = len(plain)
	}
	return dto.NodeRes{
		ID:          node.ID,
		Name:        node.Name,
		Addr:        node.Addr,
		Entrance:    node.Entrance,
		ConnectMode: node.ConnectMode,
		SkipVerify:  node.SkipVerify,
		IsProd:      node.IsProd,
		Sort:        node.Sort,
		Status:      node.Status,
		StatusMsg:   node.StatusMsg,
		Version:     node.Version,
		LastSeenAt:  node.LastSeenAt,
		Summary:     node.Summary,
		Warnings:    buildNodeWarnings(node),
		HasToken:    strings.TrimSpace(node.AccessToken) != "",

		TokenLen:         tokenLen,
		TokenLenExpected: NodeTokenLength,
		HasControlToken:  strings.TrimSpace(node.ControlToken) != "",
		ControlTokenLen:  controlLen,
	}
}

// buildNodeWarnings 把摘要数值折算成告警项。
// 阈值集中在后端，避免前端各处重复写魔法数字；文案由前端按 type 做 i18n。
func buildNodeWarnings(node model.Node) []dto.NodeWarning {
	warnings := make([]dto.NodeWarning, 0, 4)

	switch node.Status {
	case NodeStatusOffline:
		offlineHours := 0.0
		if !node.LastSeenAt.IsZero() {
			offlineHours = time.Since(node.LastSeenAt).Hours()
		}
		warnings = append(warnings, dto.NodeWarning{Type: "offline", Level: "danger", Value: offlineHours})
		return warnings
	case NodeStatusUnauthorized:
		warnings = append(warnings, dto.NodeWarning{Type: "unauthorized", Level: "danger", Value: 0})
		return warnings
	case NodeStatusUnknown:
		return warnings
	}

	summary := node.Summary
	if summary.DiskMaxPercent >= diskDangerPercent {
		warnings = append(warnings, dto.NodeWarning{Type: "disk", Level: "danger", Value: summary.DiskMaxPercent})
	} else if summary.DiskMaxPercent >= diskWarnPercent {
		warnings = append(warnings, dto.NodeWarning{Type: "disk", Level: "warn", Value: summary.DiskMaxPercent})
	}

	if summary.CertTotal > 0 && summary.CertExpiringCount > 0 {
		level := "warn"
		// 负数代表已过期，同样按 danger 处理
		if summary.CertMinDays <= certDangerDays {
			level = "danger"
		}
		warnings = append(warnings, dto.NodeWarning{Type: "cert", Level: level, Value: float64(summary.CertMinDays)})
	}

	if summary.ContainerAbnormal > 0 {
		warnings = append(warnings, dto.NodeWarning{Type: "container", Level: "warn", Value: float64(summary.ContainerAbnormal)})
	}

	return warnings
}

// normalizeNodeAddr 统一节点地址格式，必须显式带协议，避免拼出 http://https://x 这类地址
func normalizeNodeAddr(raw string) (string, error) {
	addr := strings.TrimRight(strings.TrimSpace(raw), "/")
	if addr == "" {
		return "", errors.New("节点地址不能为空")
	}
	parsed, err := url.Parse(addr)
	if err != nil {
		return "", errors.New("节点地址格式不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("节点地址必须以 http:// 或 https:// 开头")
	}
	if parsed.Host == "" {
		return "", errors.New("节点地址缺少主机名")
	}
	return addr, nil
}
