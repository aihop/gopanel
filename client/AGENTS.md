# GoPanel App AGENTS

## 一句话定位

`gopanel-app` 是 GoPanel 的手机端客户端。

当前阶段，它不只是“服务器管理 App”，还要逐步成为：

- 手机里的开发指挥台
- 远程 AI 开发会话的移动入口
- 开发过程、任务状态、预览结果、审批动作的轻量控制台

## 当前产品目标

围绕以下链路设计与开发：

1. 手机里发送开发指令
2. 手机里看到开发过程
3. 手机里打开开发产物预览
4. 手机里做关键确认与轻量追问

不要把当前目标误解为：

- 在手机里做完整 IDE
- 在手机里做远程桌面
- 在手机里重建桌面管理台全部能力

当前最佳产品形态是“开发指挥台”，不是“移动代码编辑器”。

## 当前项目现状

### 已有基础

- Flutter + Riverpod + GoRouter + Dio
- 多服务器连接与登录态管理
- 主导航结构已经成型：概览 / 资源 / 任务 / AI / 设置
- 任务中心页面已存在
- AI 工作区页面已存在
- 统一 API 客户端已存在

### 关键入口

- 应用入口：`lib/main.dart`
- 顶层路由：`lib/app/router/app_router.dart`
- 主导航：`lib/app/presentation/screens/main_scaffold_screen.dart`
- API 客户端：`lib/core/network/api_client.dart`
- 鉴权拦截器：`lib/core/network/interceptors/auth_interceptor.dart`
- 本地存储：`lib/core/storage/storage_service.dart`
- AI 工作区控制器：`lib/features/ai_workspace/presentation/controllers/ai_workspace_controller.dart`
- AI 工作区仓库：`lib/features/ai_workspace/data/ai_workspace_repository.dart`
- AI 工作区展示组件：`lib/features/ai_workspace/presentation/widgets/`
- 任务中心：`lib/features/task_center/`
- 任务中心 AI 摘要组件：`lib/features/task_center/presentation/widgets/`

### 当前关键缺口

- AI 工作区仍是 mock，`sendAiCommand()` 没有接真实后端
- 没有真正的 `DevSession` 领域模型
- 没有 WebSocket 客户端能力
- 没有预览页、WebView 或外部打开预览的完整链路
- 没有“审批中心”与“开发过程摘要”模型

## 与服务端的关系

这个 App 默认直接对接 `gopanel_v2` 的现成能力，不额外发明第二套业务协议。

默认原则：

- 服务端负责业务真相
- App 负责移动端展示、交互和轻控制
- App 优先复用服务端现有 AI、Task、Website、Pipeline、Auth 能力
- 需要新增接口时，优先补服务端而不是在客户端猜业务

## 当前阶段正确的模块方向

### AI Tab

从“聊天页”升级为“开发会话页”。

应该承载：

- 会话列表
- 当前会话状态
- 指令输入
- 追问与补充
- 当前任务摘要
- 当前预览入口

### 任务 Tab

从“通用日志查看”升级为“开发过程页”。

应该承载：

- 时间线
- 当前阶段
- 关键输出
- 错误摘要
- 文件变更摘要
- 审批状态

### 资源 / 网站相关

逐步补“预览入口”能力，而不是只停留在对象管理。

## 当前推荐领域模型

后续新增功能时，优先使用统一术语：

- `Node`：开发节点
- `DevSession`：开发会话
- `Instruction`：手机发出的一条开发指令
- `TaskRun`：指令的执行任务
- `TimelineEvent`：过程事件
- `Preview`：本轮可打开的预览
- `Approval`：待确认动作

不要继续只用“chatHistory + 一段文本回复”来承载开发会话。

## 当前开发清单

### P0：把 AI 工作区接成真实会话

- 为 AI 工作区补齐真实后端接口
- 引入 `sessionId`
- 支持会话列表、会话详情、最近消息、最近任务
- 支持绑定工作目录或项目
- 明确“发送指令”与“查看历史”的数据结构

### P1：手机发送开发指令

- 在 AI 页保留自然语言输入框
- 每条输入都创建一个 `Instruction`
- 支持附加限制项：
  - 只分析不执行
  - 允许改代码
  - 自动启动预览
  - 危险操作先确认
- 支持中途补充要求

### P1：手机查看开发过程

- 用任务时间线替代单纯聊天记录
- 显示：
  - 当前阶段
  - 最近关键输出
  - 错误摘要
  - 最近修改文件
  - 等待确认状态
- 原始日志作为二级内容展开，不要默认整屏刷终端

### P1：手机查看预览

- 新增预览页
- 支持打开 Preview URL
- 第一阶段至少支持系统浏览器打开
- 第二阶段再评估 WebView 内嵌
- 预览列表要能显示：
  - 标题
  - 状态
  - 来源任务
  - 最近更新时间

### P2：审批中心

- 新增待确认列表
- 用户可对危险动作进行继续/拒绝
- 审批结果要回写到会话和任务中

### P2：轻量终端兜底

- 只做辅助能力
- 仅用于少量紧急命令
- 不要把手机终端作为主交互

## UI 与交互原则

- 手机端优先“看状态、发任务、开预览、做确认”
- 不要把桌面后台机械压缩到手机里
- 列表负责概览，详情负责展开
- 默认先展示摘要，再展示原始日志
- 高风险动作必须更清晰地确认
- 高频入口前置，复杂能力后置
- 保持信息密度克制，避免页面像缩小版后台

## 技术原则

- 保持现有 Riverpod / GoRouter / Dio 技术路线
- 优先增量修改，不擅自大重构
- API 请求继续统一走 `ApiClient`
- 多服务器上下文继续走现有存储和激活连接逻辑
- 新增领域能力优先在 `features/` 内自闭环组织

## 推荐目录演进

基于当前项目结构，优先新增而不是推翻：

```text
lib/features/
  ai_workspace/      继续保留，但升级为真实开发会话入口
  task_center/       继续保留，升级为过程时间线中心
  dev_session/       新增，承载会话模型、repository、controller、screen
  preview/           新增，承载预览列表与预览详情
  approval/          新增，承载待审批列表与动作确认
```

如果功能较小，也可以先把 `preview`、`approval` 作为 `dev_session` 的子模块渐进落地。

## 实施顺序建议

### 第一阶段

- 打通真实 AI 会话接口
- 用真实会话替换 mock
- 让“发送开发指令”可用

### 第二阶段

- 补齐任务时间线与过程摘要
- AI 页和任务页共享任务状态

### 第三阶段

- 增加预览页
- 打通开发结果 URL 打开能力

### 第四阶段

- 增加审批中心
- 增加轻量终端或快捷动作

## 与后端联调时必须知道

- 服务端已有 AI WebSocket 与 AI task/group/message 基础
- 服务端已有容器终端、日志、SSE、Website、Pipeline 等能力
- 手机端真正缺的是“会话协议接入”和“移动端结构化展示”
- 遇到接口不清晰时，优先去 `gopanel_v2` 查 `router -> api -> service/repo`

## 开发边界

- 不要为了新需求替换现有状态管理方案
- 不要为了“更像聊天产品”弱化任务、预览和审批的结构化信息
- 不要把所有过程都做成原始终端流
- 不要让手机端承担复杂编辑器职责
- 不要继续长期依赖 mock AI 仓库

## 验证要求

任何重要改动后，至少做一项验证：

- `flutter analyze`
- 目标页面可正常构建
- 多服务器切换不串数据
- 登录态、Token、Cookie 仍然正常
- AI 会话、任务状态、预览入口链路可跑通

涉及新页面时，至少检查：

- 加载态
- 空态
- 失败态
- 长文本展示
- 返回路径与导航行为

## 给后续 AI 的一句话

在 `gopanel-app` 中，当前最重要的任务不是把手机做成 IDE，而是把现有的 AI、任务、多服务器和资源管理基础升级成“开发指挥台”：手机发开发指令，App 展示过程与摘要，必要时打开预览并做审批，所有能力都应围绕轻交互、高可读、低风险和可持续扩展展开。
