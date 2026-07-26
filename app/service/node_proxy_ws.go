package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
	"github.com/aihop/gopanel/utils/nodesign"
	"github.com/fasthttp/websocket"
)

// nodeWSHandshakeTimeout ws 握手超时。只覆盖握手，连上之后不限时长
const nodeWSHandshakeTimeout = 15 * time.Second

// DialNodeWS 以主控身份拨通目标节点的 WebSocket 接口。
//
// 浏览器无法给 ws 握手加自定义请求头，但主控是服务端发起这次拨号，
// 可以正常带上签名头——这也是为什么必须由主控中转，而不是让浏览器直连节点。
func DialNodeWS(nodeID uint, targetPath string, rawQuery string) (*websocket.Conn, error) {
	node, err := repo.NewNode().GetByID(nodeID)
	if err != nil {
		return nil, errors.New("节点不存在")
	}
	if err := validateProxyPath(targetPath); err != nil {
		return nil, err
	}

	token, err := encrypt.StringDecrypt(node.ControlToken)
	if err != nil {
		return nil, fmt.Errorf("节点控制令牌解密失败: %v", err)
	}
	if strings.TrimSpace(token) == "" {
		return nil, ErrNodeControlDisabled
	}

	nodePath := "/api/" + strings.TrimPrefix(targetPath, "/")
	wsURL, err := buildWSURL(node.Addr, nodePath, rawQuery)
	if err != nil {
		return nil, err
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := common.RandStr(24)
	header := http.Header{}
	header.Set("X-Node-Ts", ts)
	header.Set("X-Node-Nonce", nonce)
	// ws 升级请求没有请求体，body 哈希就是空串的哈希
	header.Set("X-Node-Sign",
		nodesign.SignBody(token, ts, nonce, http.MethodGet, nodePath, rawQuery, nodesign.BodyHash(nil)))
	if entrance := strings.TrimSpace(node.Entrance); entrance != "" {
		header.Set("EntranceCode", base64.StdEncoding.EncodeToString([]byte(entrance)))
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: nodeWSHandshakeTimeout,
	}
	if node.SkipVerify {
		dialer.TLSClientConfig = insecureTLSConfig()
	}

	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			// 节点拒绝握手时 HTTP 状态码是唯一线索，带出去便于排查
			return nil, fmt.Errorf("节点拒绝 WebSocket 连接（HTTP %d）：%v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("连接节点 WebSocket 失败: %v", err)
	}
	return conn, nil
}

// PipeWS 在浏览器连接与节点连接之间双向搬运消息。
// 任一侧断开就关闭另一侧，避免留下半开连接。
func PipeWS(browser wsConn, node *websocket.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, data, err := node.ReadMessage()
			if err != nil {
				return
			}
			if err := browser.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}()

	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, data, err := browser.ReadMessage()
			if err != nil {
				return
			}
			if err := node.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}()

	<-done
	_ = node.Close()
	_ = browser.Close()
	global.LOG.Debug("[Node] WebSocket 代理连接已关闭")
}

// wsConn 只依赖读写关闭三个方法，避免 service 层反向依赖 pkg/websocket 的具体类型
type wsConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// buildWSURL 把节点的 http(s) 地址换成 ws(s)，拼上路径与查询串
func buildWSURL(addr string, path string, rawQuery string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(addr), "/"))
	if err != nil {
		return "", errors.New("节点地址格式不正确")
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		return "", errors.New("节点地址必须以 http:// 或 https:// 开头")
	}
	parsed.Path = path
	parsed.RawQuery = rawQuery
	return parsed.String(), nil
}
