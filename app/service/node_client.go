package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/nodesign"
)

// nodeRequestTimeout 单节点请求超时。3~10 台规模下串行采集，一台卡住最多拖慢一轮 5 秒
const nodeRequestTimeout = 5 * time.Second

// nodeSummaryPath 被控侧只读摘要接口路径
const nodeSummaryPath = "/api/node/summary"

// ErrNodeUnauthorized 节点拒绝了令牌。单独成哨兵错误，让采集任务能把这种情况和“网络不可达”区分开——
// 前者要提示用户去重新配置令牌，后者才是真的离线。
var ErrNodeUnauthorized = errors.New("节点令牌无效或未开启只读接入")

// FetchNodeSummary 主控侧向单个节点拉取摘要。
// 返回的 error 已经是可直接展示给用户的原因（超时 / 令牌无效 / 安全入口拦截）。
func FetchNodeSummary(node model.Node) (model.NodeSummary, error) {
	token, err := encrypt.StringDecrypt(node.AccessToken)
	if err != nil {
		return model.NodeSummary{}, fmt.Errorf("节点令牌解密失败: %v", err)
	}
	if strings.TrimSpace(token) == "" {
		return model.NodeSummary{}, fmt.Errorf("节点未配置只读令牌")
	}

	target := strings.TrimRight(strings.TrimSpace(node.Addr), "/") + nodeSummaryPath
	ctx, cancel := context.WithTimeout(context.Background(), nodeRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return model.NodeSummary{}, fmt.Errorf("节点地址不合法: %v", err)
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := common.RandStr(16)
	req.Header.Set("X-Node-Ts", ts)
	req.Header.Set("X-Node-Nonce", nonce)
	req.Header.Set("X-Node-Sign", nodesign.Sign(token, ts, nonce, http.MethodGet, nodeSummaryPath))
	req.Header.Set("Accept", "application/json")
	// 节点开启了安全入口时必须带上，否则会被节点的 Entrance 中间件拦掉
	if entrance := strings.TrimSpace(node.Entrance); entrance != "" {
		req.Header.Set("EntranceCode", base64.StdEncoding.EncodeToString([]byte(entrance)))
	}

	resp, err := newNodeHTTPClient(node.SkipVerify).Do(req)
	if err != nil {
		return model.NodeSummary{}, fmt.Errorf("节点不可达: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.NodeSummary{}, fmt.Errorf("读取节点响应失败: %v", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return model.NodeSummary{}, ErrNodeUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return model.NodeSummary{}, fmt.Errorf("被节点安全入口拦截，请在节点配置中填写安全入口")
	}
	if resp.StatusCode != http.StatusOK {
		return model.NodeSummary{}, fmt.Errorf("节点返回状态码 %d", resp.StatusCode)
	}

	var result struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data model.NodeSummary `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// 旧版面板没有 /api/node/summary，请求会落到 SPA 兜底路由 r.Get("/*")，
		// 于是拿到 200 + index.html。这时候提示"地址不对"是误导的——地址没错，是节点版本太旧。
		return model.NodeSummary{}, describeNonJSONResponse(node)
	}
	if result.Code != constant.StatusCodeSuccess {
		msg := result.Msg
		if msg == "" {
			msg = "节点返回失败"
		}
		return model.NodeSummary{}, fmt.Errorf("%s", msg)
	}
	return result.Data, nil
}

// ProbeNode 校验节点配置是否可用，成功时回传一次摘要供前端立即展示
func ProbeNode(node model.Node) (dto.NodeTokenRes, error) {
	summary, err := FetchNodeSummary(node)
	if err != nil {
		return dto.NodeTokenRes{}, err
	}
	return dto.NodeTokenRes{
		Addr:     node.Addr,
		Hostname: summary.Hostname,
		Version:  summary.Version,
	}, nil
}

func newNodeHTTPClient(skipVerify bool) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	if skipVerify {
		// 节点常用自签证书，由用户在节点配置里显式开启，不做默认跳过
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   nodeRequestTimeout,
		Transport: transport,
		// 面板接口不应该发生跳转，跟随跳转反而可能把签名头带去别的主机
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
