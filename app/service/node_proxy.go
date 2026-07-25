package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/nodesign"
)

// nodeProxyTimeout 代理请求超时。比摘要采集宽松得多——
// 拉镜像、建站这类操作本身就慢，5 秒会把正常操作打断。
const nodeProxyTimeout = 120 * time.Second

// proxyDeniedPrefixes 不允许代理的目标路径前缀。
//   - auth：节点的登录接口，代理过去毫无意义，还会把凭据搬来搬去
//   - node：一是防止 A 代理到 B 再代理到 C 的链式转发，二是节点的令牌管理
//     必须在本机操作，不能被远程改掉自己的接入凭据
var proxyDeniedPrefixes = []string{"auth", "node"}

// ErrNodeControlDisabled 节点没有配置控制令牌，只能观测
var ErrNodeControlDisabled = errors.New("该节点未配置控制令牌，只能查看状态，不能执行操作")

// NodeProxyRequest 主控收到的一次代理请求
type NodeProxyRequest struct {
	NodeID uint
	Method string
	// TargetPath 去掉代理前缀后的节点侧 API 路径，如 container/list
	TargetPath  string
	RawQuery    string
	Body        []byte
	ContentType string
	// AcceptLanguage 透传语言，让节点返回的提示语与用户界面一致
	AcceptLanguage string
}

// NodeProxyResponse 节点的响应，原样回传给浏览器
type NodeProxyResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// ForwardToNode 把主控收到的请求转发给目标节点的同名接口。
//
// 设计要点：
//   - 节点侧路径与主控完全一致（/api/container/list），所以前端各页面无需改动，
//     只要 axios 层把 URL 前缀换成 /node-proxy/{id} 即可
//   - 签名覆盖请求体，节点侧还会查 nonce 防重放
//   - 节点返回的 body 原样回传，不做解析：主控不需要理解每个接口的语义
func ForwardToNode(req NodeProxyRequest) (NodeProxyResponse, error) {
	node, err := repo.NewNode().GetByID(req.NodeID)
	if err != nil {
		return NodeProxyResponse{}, errors.New("节点不存在")
	}
	if err := validateProxyPath(req.TargetPath); err != nil {
		return NodeProxyResponse{}, err
	}

	token, err := encrypt.StringDecrypt(node.ControlToken)
	if err != nil {
		return NodeProxyResponse{}, fmt.Errorf("节点控制令牌解密失败: %v", err)
	}
	if strings.TrimSpace(token) == "" {
		return NodeProxyResponse{}, ErrNodeControlDisabled
	}

	// 节点侧看到的路径，必须和签名里的一致，否则签名校验必然失败
	nodePath := "/api/" + strings.TrimPrefix(req.TargetPath, "/")
	target := strings.TrimRight(node.Addr, "/") + nodePath
	if req.RawQuery != "" {
		target += "?" + req.RawQuery
	}

	ctx, cancel := context.WithTimeout(context.Background(), nodeProxyTimeout)
	defer cancel()

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(string(req.Body)))
	if err != nil {
		return NodeProxyResponse{}, fmt.Errorf("构造节点请求失败: %v", err)
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := common.RandStr(24)
	httpReq.Header.Set("X-Node-Ts", ts)
	httpReq.Header.Set("X-Node-Nonce", nonce)
	httpReq.Header.Set("X-Node-Sign", nodesign.SignBody(token, ts, nonce, method, nodePath, nodesign.BodyHash(req.Body)))
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	if req.AcceptLanguage != "" {
		httpReq.Header.Set("Accept-Language", req.AcceptLanguage)
	}
	httpReq.Header.Set("Accept", "application/json")
	if entrance := strings.TrimSpace(node.Entrance); entrance != "" {
		httpReq.Header.Set("EntranceCode", base64.StdEncoding.EncodeToString([]byte(entrance)))
	}

	resp, err := newNodeProxyClient(node.SkipVerify).Do(httpReq)
	if err != nil {
		return NodeProxyResponse{}, fmt.Errorf("节点不可达: %v", err)
	}
	defer resp.Body.Close()

	// 上限保护：代理不该被用来搬运大文件，那条链路需要流式转发，属于后续阶段
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return NodeProxyResponse{}, fmt.Errorf("读取节点响应失败: %v", err)
	}

	// 节点的鉴权失败码不能原样透传：主控前端把 50 当成"自己"的登录失效
	// （http-enum.ts 的 TOKEN_EXPIRED = 50），会直接把用户踢到登录页。
	// 节点拒绝的是节点凭据，跟主控登录态无关，必须翻译成普通失败。
	if msg, isAuthReject := nodeAuthRejection(body); isAuthReject {
		return NodeProxyResponse{}, fmt.Errorf("节点拒绝了控制请求：%s（请检查该节点的控制令牌）", msg)
	}

	return NodeProxyResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
	}, nil
}

// NodeSupportsControl 该节点是否配置了控制令牌
func NodeSupportsControl(node model.Node) bool {
	return strings.TrimSpace(node.ControlToken) != ""
}

// nodeAuthRejection 判断节点响应是不是鉴权拒绝（code=50），并取出原因
func nodeAuthRejection(body []byte) (string, bool) {
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false
	}
	if result.Code != constant.StatusCodeAuthInvalid {
		return "", false
	}
	msg := strings.TrimSpace(result.Msg)
	if msg == "" {
		msg = "未提供原因"
	}
	return msg, true
}

func validateProxyPath(path string) error {
	clean := strings.Trim(strings.TrimSpace(path), "/")
	if clean == "" {
		return errors.New("代理路径不能为空")
	}
	// 目录穿越防护：拼进 URL 前先挡掉，避免 ../ 逃出 /api 前缀
	if strings.Contains(clean, "..") {
		return errors.New("代理路径不合法")
	}
	head := clean
	if idx := strings.Index(clean, "/"); idx > 0 {
		head = clean[:idx]
	}
	for _, denied := range proxyDeniedPrefixes {
		if strings.EqualFold(head, denied) {
			return fmt.Errorf("接口 %s 不允许通过节点代理访问", head)
		}
	}
	return nil
}

func newNodeProxyClient(skipVerify bool) *http.Client {
	client := newNodeHTTPClient(skipVerify)
	// 复用摘要客户端的 TLS/跳转策略，只把超时放宽到代理场景
	client.Timeout = nodeProxyTimeout
	return client
}
