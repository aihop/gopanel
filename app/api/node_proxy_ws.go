package api

import (
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/convertor"
)

// NodeProxyWs 把浏览器的 WebSocket 连接桥接到目标节点的同名接口。
//
// 为什么必须由主控中转：浏览器无法给 ws 握手加自定义请求头，也就没法带节点签名；
// 而且节点常在内网或用自签证书，浏览器直连会遇到混合内容与证书问题。
// 主控作为服务端拨号则不受这些限制。
//
// 查询参数除主控鉴权用的 auth/token 外全部透传，所以前端只需把 endpoint 换成
// node-proxy-ws/{id}/container/exec，cols/rows/containerID 这些照原样带就行。
//
// 路径形如 /api/node-proxy-ws/:id/container/exec?containerID=xxx&cols=80&rows=40
// → 节点的 /api/container/exec?containerID=xxx&cols=80&rows=40
func NodeProxyWs(wsConn *websocket.Conn) {
	nodeID, _ := convertor.ToInt(wsConn.Params("id"))
	// 通配符要取 "*1"：pkg/websocket 升级时按 c.Route().Params 里的名字复制参数，
	// 而 Fiber 把通配符存成 *1。HTTP 侧的 c.Params("*") 有特殊处理，ws 这层没有。
	targetPath := wsConn.Params("*1")
	if targetPath == "" {
		targetPath = wsConn.Params("*")
	}
	rawQuery, _ := wsConn.Locals(middleware.NodeProxyQueryLocalKey).(string)

	if nodeID <= 0 || targetPath == "" {
		writeWsError(wsConn, "代理参数不完整")
		return
	}

	nodeConn, err := service.DialNodeWS(uint(nodeID), targetPath, rawQuery)
	if err != nil {
		global.LOG.Warnf("[Node] WebSocket 代理拨号失败: %v", err)
		writeWsError(wsConn, err.Error())
		return
	}

	service.PipeWS(wsConn, nodeConn)
}

func writeWsError(wsConn *websocket.Conn, msg string) {
	_ = wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
	_ = wsConn.Close()
}
