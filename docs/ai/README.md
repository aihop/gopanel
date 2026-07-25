# GoPanel v2 — AI 项目入口

`gopanel_v2` 是 GoPanel 的主控项目。Golang + Fiber v3 + Vue 3 (Naive UI)，单二进制部署。

当前阶段方向：把 AI 工作区、任务链路、网站/预览能力和实时通道收口为"手机开发指挥台"的控制平面。

---

## 给 AI 的最高优先级指令

优先沿现有 AI/Task/Website/Pipeline 链路增量增强，不要重做。
手机发指令 → 后端编排会话与任务 → AI/终端执行 → 回传过程和预览 → 高风险动作做审批与审计。

---

## 模块快速入口

| 模块 | 路由注册 | 核心 API 文件 | 核心模型 |
|------|---------|-------------|---------|
| AI 助手 | `app/router/ai_agent.go` | `app/api/ai_agent*.go` | `app/model/ai_chat_history.go` |
| 容器 | `app/router/container.go` | `app/api/container*.go` | `dto/container.go` |
| 数据库 | `app/router/database.go` | `app/api/database*.go` `app/api/db_manager.go` | `app/model/database*.go` |
| 网站 | `app/router/website.go` | `app/api/website*.go` | `app/model/website*.go` |
| 流水线 | `app/router/pipeline.go` | `app/api/pipeline*.go` | — |
| SSL | `app/router/ssl.go` | `app/api/ssl*.go` | — |
| 多节点观测 | `app/router/node.go` | `app/api/node.go` | `app/model/node.go` |

### 前端对应

| 页面 | 路由 | 核心视图 |
|------|------|---------|
| AI 工作区 | `/ai` | `admin/src/views/AIAgent/` |
| 容器 | `/container` | `admin/src/views/Container/` |
| 数据库 | `/database` | `admin/src/views/Database/` |
| 多节点 | `/node` | `admin/src/views/Node/`、常驻细条 `admin/src/layouts/common/NodeRail/` |

### 辅助层

- 高权限 helper：`gpc/`（Go）
- 宿主机 agent：`gp-agent/`（Go）
- 通信封装：`utils/gpc/`、`utils/gpagent/`
- 主控↔节点签名：`utils/nodesign/`（单独成包，避免 service 与 middleware 循环依赖）

---

## 关键文档索引

- [知识库（已解决问题 & 技术债）](./knowledge-base.md)
- [文件拆分方案（重构参考）](./refactoring.md)
- 项目根 `AGENTS.md`：历史完整版，含全部细节约定

---

## 快速开始

```bash
# 后端编译验证
go vet ./app/...

# 检查改动文件
git status
git log --oneline -5
```
