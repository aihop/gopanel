# 网站智能诊断 Hook 接入

## 职责边界

- 网站项目负责发现业务错误、浏览器错误并生成统一事件。
- GoPanel 负责消费、脱敏、聚合、问题管理和 Code 任务编排。
- Caddy 与主动探测由 GoPanel 自己采集，项目不需要重复上报。
- Code 只接收聚合后的脱敏诊断包，不直接读取生产目录或执行事件内容。
- 所有采集入口在统一入库层执行网站的来源开关与内容开关；关闭某类监测后，文件、远程 Hook、Caddy 和主动探测都不能绕过设置入库。

## 统一事件

```json
{
  "schema": "gopanel.website-diagnostic.v1",
  "eventId": "release-requestId-errorCode",
  "websiteId": 17,
  "source": "backend",
  "kind": "business_error",
  "severity": "error",
  "title": "Order items missing",
  "message": "response.items is missing",
  "stack": "OrderDetail.vue:128",
  "requestId": "req-123",
  "sessionId": "sha256-session-id",
  "method": "GET",
  "route": "/api/orders/:id",
  "httpStatus": 200,
  "businessCode": "ORDER_ITEMS_MISSING",
  "durationMs": 326,
  "release": "8d91c4a",
  "occurredAt": "2026-08-13T12:00:00Z",
  "metadata": { "component": "OrderDetail" }
}
```

不要提交 Cookie、Authorization、Token、密码、请求/响应正文、表单值、localStorage、sessionStorage、真实 IP 或完整查询参数。`eventId` 必须在同一网站内稳定且唯一。

## 本机或容器项目

在网站诊断设置中查看 `tracking` 目录。项目先写入 `inbox/<eventId>.tmp`，完成写入和 `fsync` 后原子改名为 `inbox/<eventId>.ready`。GoPanel 只消费 `.ready` 文件。

容器项目只挂载 `tracking/inbox`，建议应用用户仅拥有该目录的写权限，不要挂载整个 GoPanel 数据目录。

Node.js 示例：

```js
import { open, rename } from "node:fs/promises"
import path from "node:path"

export async function reportGoPanelDiagnostic(inbox, event) {
  const safeId = event.eventId.replace(/[^a-zA-Z0-9_-]/g, "-")
  const temporary = path.join(inbox, `${safeId}.tmp`)
  const ready = path.join(inbox, `${safeId}.ready`)
  const file = await open(temporary, "wx", 0o640)
  try {
    await file.writeFile(JSON.stringify({ schema: "gopanel.website-diagnostic.v1", ...event }))
    await file.sync()
  } finally {
    await file.close()
  }
  await rename(temporary, ready)
}
```

Go 和 PHP 使用相同协议：写临时文件、刷新、关闭、同目录原子改名。不要直接写 `.ready`。

## 跨机器接收

在设置页生成站点 Hook 密钥。密钥只显示一次，保存在网站项目的 Secret 管理中。接口：

```text
POST /api/website-diagnostics/<websiteId>/events
```

请求体不超过 32 KB。签名串：

```text
timestamp\nnonce\nPOST\nrequestPath\nsha256(body)
```

用站点密钥执行 HMAC-SHA256，并以十六进制发送：

```text
X-GoPanel-Timestamp: Unix 秒
X-GoPanel-Nonce: 至少 16 字符且每次唯一
X-GoPanel-Signature: 十六进制 HMAC-SHA256
```

时间偏差超过 5 分钟、nonce 重放、来源域名不匹配、签名错误或每分钟超过 60 次均会拒绝。

## 浏览器 Hook

浏览器不能获得站点 Hook 密钥。Vue `errorHandler`、React `ErrorBoundary`、`window.onerror`、`unhandledrejection` 和资源加载错误应发送给网站自己的 `/__gopanel/diagnostics` 后端接口，由网站后端限流、脱敏并写入本机 inbox 或调用远程 HMAC 接口。

网站接收接口至少应：

- 校验当前站点 Origin。
- 限制正文为 32 KB。
- 按站点、会话和 IP 限流。
- 丢弃 Cookie、Token、表单、存储内容、查询参数和请求正文。
- 限制 message、stack、breadcrumbs 数量和长度。
- 只返回接收成功，不暴露目录、密钥或内部错误。

Vue 示例只负责采集，不直接调用 GoPanel：

```ts
app.config.errorHandler = (error, instance, info) => {
  void fetch("/__gopanel/diagnostics", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      eventId: crypto.randomUUID(),
      source: "browser",
      kind: "vue_error",
      severity: "error",
      title: info,
      message: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack?.slice(0, 16000) : "",
      route: location.pathname,
      occurredAt: new Date().toISOString()
    })
  })
}
```

React `ErrorBoundary` 和普通 JavaScript 使用同样的站内接收接口；资源错误使用捕获阶段的 `window.addEventListener("error", handler, true)`，Promise 错误使用 `unhandledrejection`。

## 验证

1. 先发送一条测试事件。
2. 在网站的“智能诊断 → 问题”确认聚合结果。
3. 检查消息、堆栈和 metadata 已脱敏。
4. 重复发送相同 `eventId`，确认不会增加次数。
5. 使用新 `eventId` 和相同错误指纹，确认聚合到同一 Issue。
