# GoPanel v2 AGENTS

## 一句话定位

`gopanel_v2` 是 GoPanel 的主控项目，负责 Web 管理台、AI 工作区、任务/日志、网站/容器/流水线管理，以及为手机端提供可复用的控制平面能力。

当前阶段方向：把 AI 工作区、任务链路、网站/预览能力和实时通道收口为"手机开发指挥台"的控制平面。

## 给 AI 的最高优先级指令

优先沿现有 AI/Task/Website/Pipeline 链路增量增强，不要重做一套。

手机发指令 → 后端编排会话与任务 → AI/终端执行 → 系统回传过程和预览 → 高风险动作做审批与审计。

服务端输出数据分两层：原始层（完整日志/终端流）和摘要层（适合手机展示的结构化状态）。

## 模块快速入口

- 路由总入口：`app/router/app.go`
- AI 路由：`app/router/ai_agent.go`
- 数据库路由：`app/router/database.go`
- 容器路由：`app/router/container.go`
- AI 会话/模型：`app/model/ai_chat_history.go`
- AI 会话核心：`app/api/ai_agent.go`
- 数据库管理器：`app/service/db_manager*.go` + `app/api/db_manager.go`
- 容器缓存：`app/service/container_list_cache.go`（已知竞争条件问题已修）
- 前端 AI 页面：`admin/src/views/AIAgent/`
- 前端数据库管理器：`admin/src/views/Database/`
- 高权限 helper：`gpc/`。宿主机 agent：`gp-agent/`。通信封装：`utils/gpc/`、`utils/gpagent/`

## 开发原则

- 优先增量增强现有 AI/Task/Website/Pipeline 链路，不要平地起全新体系
- 优先让协议语义稳定，再考虑前端表现
- 重要对象必须可审计、可恢复、可定位
- 所有涉及远程执行的功能都必须考虑权限和审批
- 查询类接口优先返回结构化摘要，详情类接口再返回原始日志

## 已解决问题（详细记录见 `docs/ai/knowledge-base.md`）

- 容器列表缓存竞争条件 — `f578b1c`
- Contextx 不支持平面字段 — `8400702`
- OperationsView 缺失导入导致空白 — `40ef5ed`
- NModal 宽度用 Tailwind class 无效 — `4144358`
- DataView v-show 残留 DOM — `ece6439`
- SQL 导入被注释检查错误跳过 — `b8dd282`

## 文档索引

- `docs/ai/README.md` — 模块入口 + 快速参考
- `docs/ai/knowledge-base.md` — 已解决问题的技术债记录
- `docs/ai/refactoring.md` — 前后端文件拆分方案

## Git 提交约定

每次完成一个功能任务或修复一个 bug 后，必须立即提交并推送。

- 提交粒度：一个功能 / 一个 bugfix 一次提交
- 提交信息：首行用 `type: 标题` 概括（feat/fix/refactor/style/docs/chore），后续用 `-` 列表写关键改点
- 不做 `git add .` 全量提交，只加本次改动相关的文件
- 不改动文件不应出现在提交中
- 每次 commit 后立即执行 `git pull --rebase && git push`，确保远程同步（先拉后推，避免冲突）

## Naive UI 组件样式约定

Naive UI 的 NModal 等弹窗组件用 `class` 属性挂 TailwindCSS 宽度类（如 `w-[560px]`）是无效的，原因：

- `class` 挂在了组件外层 wrapper 上（全屏遮罩层），不是实际卡片容器
- Naive UI 用 CSS-in-JS 内联控制关键尺寸，外部类优先级不够
- NModal 的 `preset="card"` 内部渲染的是嵌套的 `.n-card`，宽度控制在那一层

**正确的写法：**

```
<n-modal preset="card" style="width: 560px">
```

用 Vue 的 `style` prop 将宽度内联到组件根元素，Naive UI 的内部样式机制会将它传递给实际卡片容器。

## 不要做的事

- 不要把手机需求理解成远程桌面需求
- 不要把 AI 工作区继续维持为"只有终端，没有结构化状态"
- 不要把所有结果都只塞进原始日志，让手机自己解析
- 不要为追求一步到位而重写整个 AI 模块
- 不要忽略审计和权限边界
