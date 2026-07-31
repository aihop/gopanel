# GoPanel 桌面端

桌面端使用 Wails 2 封装现有 Vue 管理台，并继续复用 Fiber HTTP API 和 WebSocket 实时通道。

## 构建

macOS、Linux 或 Windows 应在对应系统上原生构建：

```bash
./build-desktop.sh
```

指定目标平台时，将 Wails 参数直接传给脚本：

```bash
./build-desktop.sh -platform darwin/arm64
```

macOS 产物位于 `desktop/build/bin/GoPanel.app`，Bundle ID 为 `io.aihop.gopanel`。

## 运行模型

- 桌面窗口通过 Wails 加载现有前端产物。
- Fiber 只监听 `127.0.0.1` 的随机端口，不暴露到局域网。
- 普通请求由 Wails AssetServer 转发到 Fiber，WebSocket 直接连接随机回环端口。
- 配置、数据库和日志写入 `~/.gopanel`，与现有 GoPanel 默认数据目录保持一致。
- 首次启动会创建 `admin@gopanel.local` 管理员，并通过原生对话框显示随机临时密码；登录后应立即修改密码。
- 关闭窗口时会停止 AI 执行任务并优雅关闭 HTTP 服务。

当前 macOS 构建使用 ad-hoc 自签名；正式分发仍需配置 Apple Developer ID 签名与公证。
