# GoPanel v2 AGENTS

## 一句话定位

`gopanel_v2` 是 GoPanel 的主控项目，负责 Web 管理台、AI 工作区、任务/日志、网站/容器/流水线管理，以及为手机端提供可复用的控制平面能力。

当前阶段，这个项目不只是“服务器面板后端”，还要逐步成为：

- 手机 App 的开发指挥后端
- 电脑开发节点与 AI 会话的控制平面
- 开发过程、任务状态、预览结果、审批动作的统一来源

## 当前核心目标

围绕“手机里发送开发指令、查看开发过程、预览开发产物”推进能力建设。

默认优先级：

1. 让手机端可以向某个开发会话发送开发指令
2. 让手机端可以实时看到过程、日志、状态和关键摘要
3. 让手机端可以直接打开或查看本轮开发产物预览
4. 让高风险动作具备审批、审计和权限控制

不要优先做：

- 远程桌面
- 手机端完整 IDE
- 为移动端重造一套与现有 Web/AI 不兼容的协议

## 项目结构入口

### 后端入口

- 路由总入口：`app/router/app.go`
- AI 路由入口：`app/router/ai_agent.go`
- AI 会话核心：`app/api/ai_agent.go`
- 容器终端：`app/api/terminal.go`
- 认证中间件：`app/middleware/jwt.go`
- 用户模型：`app/model/user.go`

### Web 管理台入口

- 管理台路由：`admin/src/router/index.ts`
- AI 终端组件：`admin/src/views/AIAgent/components/AgentTerminal.vue`
- 通用终端组件：`admin/src/components/Terminal.vue`
- Pipeline 页面：`admin/src/views/Pipeline`
- Website 页面：`admin/src/views/Website`

### 本机辅助层

- 高权限 helper：`gpc/`
- 宿主机 agent：`gp-agent/`
- 本地与特权通信封装：`utils/gpc/`、`utils/gpagent/`

## 当前已具备的能力

- 已有 AI 工作区与持久化工作目录
- 已有 AI 终端 WebSocket：`GET /api/ai/terminal`
- 已有 AI 任务、分组、消息存储基础
- 已有容器终端、容器日志、进程/文件实时通道
- 已有 SSE 日志链路，可用于安装、流水线、系统更新等长任务
- 已有 Website、Pipeline、Release、AppDeploy 等可承接预览与发布的对象
- 已有登录、JWT、Session、子管理员目录沙箱基础

这些能力优先复用，不要重新发明一套。

## 当前主要缺口

当前系统已经接近“开发控制台”，但还没有完整落地“手机开发指挥台”所需的几个关键抽象：

- 缺少统一的 `DevSession` 概念
- 缺少统一的 `Instruction -> Task -> Timeline -> Preview` 数据链路
- 缺少专门面向手机端的结构化状态摘要接口
- 缺少预览对象的统一登记与回传机制
- 缺少针对“手机发指令”的审批、审计和权限拆分

当前 AI 能力仍偏 WebSocket 终端和 AI 任务页，不等于完整的手机开发控制面。

## 当前阶段推荐抽象

后续新增能力时，优先统一以下领域术语：

- `Node`：开发节点，通常是一台电脑或某个远程执行环境
- `DevSession`：某个项目/工作目录绑定的一条长期开发会话
- `Instruction`：来自手机或 Web 的一条开发指令
- `TaskRun`：Instruction 的一次实际执行
- `TimelineEvent`：执行过程中的结构化事件
- `Preview`：本轮结果关联的预览地址、截图、状态
- `Approval`：高风险动作的人工确认节点
- `AuditLog`：谁在何时发起了什么操作

如果现有 `AITask` 能承接其中一部分，优先渐进增强，不要直接平地起全新体系。

## 与手机端对接的首选策略

优先沿现有 AI 能力和任务能力扩展，而不是新开一条割裂协议。

推荐链路：

1. 手机端选择一个 `DevSession`
2. 手机端发送一条 `Instruction`
3. 后端把它投递到 AI 会话或受控终端
4. 执行过程产出 `TimelineEvent`
5. 关键日志和摘要回推给手机端
6. 如有页面/服务启动成功，登记 `Preview`
7. 如遇高风险动作，进入 `Approval`

## 当前开发清单

### P0：先把骨架补齐

- 补齐 `DevSession` 领域对象
- 提供会话列表、会话详情、当前状态接口
- 支持按用户、项目、工作目录管理会话
- 明确会话和 `AITask`、`AIGroup` 的关系

### P1：让手机能发开发指令

- 提供“发送开发指令”接口
- 指令支持自然语言正文 + 限制项
- 限制项至少包括：
  - 是否允许改代码
  - 是否自动启动预览
  - 是否有危险操作先确认
  - 是否只分析不执行
- 指令必须落库并绑定到会话和用户

### P1：让手机能看过程

- 提供任务时间线接口
- 提供任务状态摘要接口
- 提供错误摘要、当前阶段、最近输出接口
- 优先复用现有 WS/SSE，必要时在服务端做二次结构化聚合

### P1：让手机能看预览

- 为开发结果增加 `Preview` 对象
- 统一记录：
  - 预览类型
  - URL
  - 标题
  - 来源任务
  - 是否可访问
  - 最近探测时间
- 对前端开发优先支持临时预览地址
- 对网站对象优先复用现有 Website/Proxy 语义

### P2：审批与审计

- 把删除、覆盖、推送、安装系统依赖、执行危险命令纳入审批
- 手机端需要可查看待审批事项并明确放行/拒绝
- 所有手机指令和 AI 执行结果都要留审计记录

### P2：权限拆分

- 区分“查看会话”“发送指令”“查看日志”“执行 shell”“审批危险操作”
- 不要让所有移动端用户都默认拥有完整终端权限

## Web / 后端分工

### Web 管理台负责

- 完整管理操作
- 深度会话查看
- 复杂排障与资源配置
- 开发与预览链路的桌面端承载

### 手机端负责

- 快速发起开发指令
- 查看过程状态
- 查看错误摘要
- 打开预览结果
- 做关键确认与轻量追问

因此，服务端输出的数据要分两层：

- 原始层：完整日志、终端流、消息历史
- 摘要层：适合手机展示的结构化状态

## 开发原则

- 优先增量增强现有 AI/Task/Website/Pipeline 链路
- 不要为了手机端单独重做一套完全独立的后端架构
- 优先让协议语义稳定，再考虑前端表现
- 重要对象必须可审计、可恢复、可定位
- 预览能力优先复用现有网站、容器、流水线与发布能力
- 所有涉及远程执行的功能都必须考虑权限和审批

## 前端重构约定

- 管理台前端存在一批历史大文件，超过 400 行时默认优先拆分，不要继续把新逻辑堆回原文件
- 遇到手写 CSS 时，优先尽可能改写为 TailwindCSS，并确保页面视觉效果一比一还原
- 拆分顺序优先：
  1. 纯视图区块组件
  2. 表格列工厂、选项常量、格式化函数
  3. 页面级 composable
- 重构时优先保持现有接口、事件名、弹窗打开方式和提交流程不变，先做结构收口，再做行为优化
- 如果页面已有正在推进的业务改动，优先避开冲突文件，先处理未脏的大文件，减少误伤
- 当前管理台已验证可行的拆分模式：
  - `views/Pipeline/src/CreatePipelineModal.vue` 适合拆为主容器 + 表单区块组件 + `pipelineForm.ts`
  - `views/Host/files.vue` 适合拆为工具栏、筛选区、表格列工厂、路径导航 composable、文件动作 composable
  - `views/Host/firewall.vue` 适合拆为概览区、规则工作台、列工厂、规则服务函数
  - `views/Host/process.vue` 适合拆为搜索工具栏、详情抽屉、进程/网络列配置，主页面保留 WebSocket 与分页状态调度
  - `views/Host/Toolbox/Daemon.vue` 适合拆为顶部概览区、表格列工厂、Agent 状态 composable、守护进程动作 composable，主页面保留 useTable、tab 状态和弹窗引用
  - `views/Container/image/index.vue` 适合先抽顶部工具栏，再继续拆表格逻辑
  - `views/Container/container/index.vue` 适合拆为列表工具栏、表格列工厂、列表数据 composable，主页面保留弹窗引用与批量操作编排
  - `views/Container/container/operate/index.vue` 适合拆为基础信息、端口网络、挂载、进阶参数等表单区块，并把端口列配置与表单转换/校验 helper 抽离
  - `views/Container/compose/index.vue` 适合拆为列表工具栏、创建抽屉、删除确认弹窗、表格列工厂和创建流程 composable，主页面保留列表查询、分页和编辑入口
  - `views/Container/setting/index.vue` 适合拆为运行时状态头部、基础配置区块、全部配置编辑区、修复弹窗，并把运行时状态/保存/修复逻辑拆到页面级 composable
  - `views/Apps/components/AppsAll.vue` 适合拆为卡片列表、安装弹窗、详情抽屉、日志弹窗和安装流程 composable
  - `views/Apps/components/AppsInstalled.vue` 适合拆为已安装应用卡片列表、详情抽屉、卸载确认弹窗、运行/安装日志弹窗，以及独立的日志修复流 composable
  - `views/Dashboard/components/StatusCard.vue` 适合拆为基础信息概览条、CPU/内存/负载资源卡片、磁盘与 GPU/XPU 面板，并把数值格式化函数抽到共享 helper
  - `views/Website/components/AccessLogDrawer.vue` 适合拆为抽屉壳、日志面板、详情弹窗、IP 统计弹窗，并把日志解析与绑定元信息读取抽到 helper/composable
  - `views/Website/SSL.vue` 适合拆为页面头部、证书相关弹层、表格列工厂，以及按“推送规则/日志流/页面状态”分层的 composable
  - `views/Database/src/DatabaseManager.vue` 适合拆为左侧导航、顶部上下文条、页面级 `useDatabaseManager` composable，主页面保留标签页编排与子视图挂载
  - `views/Database/src/components/DataView.vue` 适合把表列表、记录分页、筛选、删除等状态抽到 `useDataView` composable，主文件保留浏览态工具栏与数据表格渲染
  - `views/Database/src/components/StructureView.vue` 适合把字段和索引操作、结构摘要抽到 `useStructureView` composable，主文件保留结构表格、索引区块和弹层编排
  - `views/Database/src/components/OperationsView.vue`、`views/Database/src/components/SearchView.vue` 适合补表摘要、危险操作分区、搜索状态提示等轻量工作台信息，优先增强交互反馈，再决定是否继续拆 composable
  - `views/Database/src/components/DatabaseWorkspaceHeader.vue` 适合承接各数据库子页共享的服务器/数据库/表上下文头部，统一摘要徽标和右侧动作区域
- 这类重构完成后，至少要检查改动文件诊断，确保结构调整没有引入模板或类型错误

## 后端大文件重构约定

- `app` 目录下的 Go 文件，`400` 行默认视为强提醒阈值；`600+` 基本都应优先拆分
- `400` 不是死线；如果文件仍是单一主流程，可以暂时保留，但应先把 helper、校验、适配层、日志/预览等侧向逻辑抽走
- 后端拆分优先使用“同 package 新文件 + 迁移整组函数”的方式，避免改动对外符号、路由注册和 service/repo 调用链
- 拆分顺序优先：
  1. handler 与 helper 分离
  2. 主流程与日志/预览/状态探测分离
  3. CRUD 与 repair/compat/utils 分离
  4. runtime/provider 特化逻辑分离
- 当前已验证可行的后端拆分模式：
  - `app/repo/pipeline.go` 适合拆为 `pipeline_repo`、`pipeline_record_repo`、`release_repo`、`release_repo_repair`
  - `app/api/ai_agent_rest.go` 适合拆为 group、task、session、preview 四组
  - `app/api/container.go` 适合拆为 container 主体、network、volume、compose
  - `app/api/app_install.go` 适合拆为 install 主入口、installed、本地安装、日志流
  - `app/api/file.go` 适合拆为 manage、content、transfer 三层
  - `app/service/pipeline.go` 适合把 step 函数抽到独立文件，主文件保留编排入口
  - `app/service/pipeline_utils.go` 适合拆为 detect、execute、runner、image_detect、artifact 五层
  - `app/service/db_manager.go` 适合拆为表结构查询、记录操作、核心 service
  - `app/service/pipeline_application.go` 适合拆为发布记录流、元信息 helper、主 service
  - `app/service/app_deploy.go` 适合拆为 website 发布动作、runtime 探测/切换、部署元信息 helper
  - `app/service/apps_utils.go` 适合拆为 runtime 状态、compose env 校验、安装文件流程、compose compat、命令展开 helper
  - `app/service/container.go` 适合先拆为 container 壳文件、列表链路、运行时管理，再把列表管理二次细分
  - `app/service/container_utils.go` 适合拆为 stats、config、inspect 三组 helper
  - `app/service/container_logs.go` 适合拆为主日志流、GPC 直连、journald/podman 日志兜底
  - `app/service/container_docker.go` 适合拆为 runtime 状态读取、daemon 配置写入、运行时控制
  - `app/service/container_network.go` 适合拆为主接口、podman 适配、static IP 兜底逻辑
  - `app/service/website_engine.go` 适合拆为部署主流程、镜像/入口探测、runner 配置 helper
  - `app/service/firewall.go` 适合拆为 service 主入口、rule loader、rule filter/normalize
  - `app/service/caddy_utils.go` 适合拆为基础读写、server block 修改、查询/解析 helper
  - `app/service/file.go` 适合拆为 tree、content、gpc 适配、主 service
  - `app/service/ssl.go` 适合拆为主 service、ACME 签发流程、证书文件/域名 helper
  - `app/api/ai_agent.go` 适合拆为主 WebSocket handler、preview/workdir helper、workspace/runtime helper、PTY 输出转发 helper
- 后端拆分后至少做一项真实验证，优先顺序：
  1. `gofmt`
  2. 目标 package `go test`
  3. 关键改动文件诊断为空或仅剩低价值 hint

## API 设计原则

- 手机端优先消费稳定 REST 接口 + WS/SSE 实时流
- 保持与现有 `/api` 路由体系一致
- 不要凭空设计和当前鉴权不兼容的协议
- 查询类接口优先返回结构化摘要
- 详情类接口再返回原始日志、消息、事件
- 新接口命名优先围绕 `session`、`instruction`、`timeline`、`preview`、`approval`

## Runner 目录约定

- Pipeline Runner 目录语义要明确区分 `sourceMountDir` 与 `workingDir`，不要再让 `workingDir` 同时承担“代码源挂载点”和“最终运行目录”两种含义
- 当用户显式填写 `workingDir` 时，默认直接把 `release`/代码源挂到该目录；不要再额外强制经由固定中间目录
- 当用户未填写 `workingDir` 时，才保留当前默认的只读中间目录挂载 + 同步到最终运行目录的兼容链路
- `persistentPaths` 一律视为高优先级持久化目录；同步/清理脚本不得删除已声明的持久化路径
- 调整 Runner 行为时，日志中必须明确打印代码源挂载点、最终运行目录以及是否启用中间同步，方便排障

## 预览能力原则

预览不是附属信息，而是手机开发链路的核心交付之一。

新增预览能力时优先满足：

- 能返回可直接打开的 URL
- 能区分预览不可用、启动中、可访问、已失效
- 能回传最近一次探测结果
- 能在任务详情里看到该预览来源于哪个任务
- 能在后续支持截图缓存

## 验证要求

涉及后端改动时，至少做一项验证：

- 相关 Go 文件可通过诊断
- 新增/修改路由与现有鉴权兼容
- WS/SSE 链路能正常建立
- 管理台或移动端能正确消费新增字段

涉及前后端联动时，至少确认：

- 指令发出后任务能被创建
- 状态能持续推进
- 预览可被登记和读取
- 审批能正确阻断危险动作

## Git 提交约定

每次完成一个功能任务或修复一个 bug 后，必须立即提交并推送。

- 提交粒度：一个功能 / 一个 bugfix 一次提交
- 提交信息：首行用 `type: 标题` 概括（feat/fix/refactor/style/docs/chore），后续用 `-` 列表写关键改点
- 不做 `git add .` 全量提交，只加本次改动相关的文件
- 不改动文件不应出现在提交中
- 每次 commit 后立即执行 `git pull --rebase && git push`，确保远程同步（先拉后推，避免冲突）

## 不要做的事

- 不要把手机需求理解成远程桌面需求
- 不要把 AI 工作区继续维持为“只有终端，没有结构化状态”
- 不要把所有结果都只塞进原始日志，让手机自己解析
- 不要为追求一步到位而重写整个 AI 模块
- 不要忽略审计和权限边界

## 给后续 AI 的一句话

在 `gopanel_v2` 中，当前最重要的方向是把现有 AI 工作区、任务链路、网站/预览能力和实时通道收口为“手机开发指挥台”的控制平面：手机发指令，后端编排会话与任务，AI/终端执行，系统回传过程和预览，并对高风险动作做审批与审计。
