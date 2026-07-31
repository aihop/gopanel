# GoPanel Client

GoPanel 的 Flutter 客户端，使用 Riverpod、GoRouter 和 Dio，对接 GoPanel 服务端现有控制面。

## Code 能力

AI Tab 是移动端 Code 指挥台，复用服务端 `/api/code` 链路，支持：

- 加载已有 Code 项目、执行器和历史会话
- 创建或续接开发会话
- 发送开发指令并轮询结构化状态
- 查看时间线、文件变化和预览
- 审批或拒绝高风险操作

客户端使用当前服务器的 `X-Auth` 登录态，不维护第二套 Code 协议。

## 验证

```bash
flutter analyze
flutter test
```
