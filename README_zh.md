<p align="center">
  <a href="https://gopanel.cn/">
    <img src="./admin/src/assets/images/logo-text.svg" alt="GoPanel" width="360" />
  </a>
</p>

<p align="center"><strong>面向一人公司与小型团队的自托管 AI 开发交付指挥台。</strong></p>

<p align="center">
  用手机委派开发，运行 rootless 容器，自动签发与续签证书，管理数据和备份，并在同一个控制平面中维护产品上线所需的一切。
</p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0-2563EB.svg" alt="License: GPL v3" /></a>
  <a href="https://github.com/aihop/gopanel/releases"><img src="https://img.shields.io/github/v/release/aihop/gopanel?color=4F46E5" alt="GitHub release" /></a>
  <a href="https://github.com/aihop/gopanel"><img src="https://img.shields.io/github/stars/aihop/gopanel?style=flat&color=7C3AED" alt="GitHub stars" /></a>
  <a href="https://gopanel.cn/"><img src="https://img.shields.io/badge/%E5%AE%98%E7%BD%91-gopanel.cn-0891B2.svg" alt="GoPanel 官网" /></a>
</p>

<p align="center">
  <a href="./README.md"><img src="https://img.shields.io/badge/English-E2E8F0.svg" alt="English" /></a>
  <a href="./README_zh.md"><img src="https://img.shields.io/badge/%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-2563EB.svg" alt="简体中文" /></a>
</p>

<p align="center">
  <a href="https://gopanel.cn/">官方网站</a> ·
  <a href="https://github.com/aihop/gopanel/releases">版本发布</a> ·
  <a href="https://github.com/aihop/gopanel/issues">问题反馈</a>
</p>

---

经营一家软件公司，往往意味着一个人同时承担产品、开发、评审、发布和运维。AI Coding Agent 已经可以编写代码，但需求之后的 Git 管理、质量检查、预览验收、生产发布、运行监测与风险控制，仍然需要人手动连接。

GoPanel 把这些责任收口到一个自托管控制平面中。

它不是“服务器面板里加一个聊天框”，而是围绕真实交付组织 AI 开发会话、隔离工作区、Git 交付、质量检查、流水线、版本制品、网站、容器、数据库、运行诊断、审批和移动操作。它只有一个明确目标：

> 让一个人不必整天守在终端前，也能安全地开发、发布和运营多个产品。

## 从需求到生产

GoPanel 正在把已经存在的 Code、Pipeline、Release、Website、Container、Flow 和 Mobile 能力收口为一条连续链路：

```text
从 Web 或手机描述需求
  → AI 在项目隔离工作区中执行
  → 查看过程、文件、Diff 与质量检查
  → 打开预览并验收结果
  → 审批高风险或生产动作
  → 构建并发布可追溯版本
  → 持续监测、诊断故障并安全恢复
```

控制平面保存适合决策的结构化状态，完整终端流和原始日志则作为详情随时可查。用户能够知道正在发生什么、为什么暂停、现在需要谁处理，以及下一步可以安全执行什么。

## 为什么选择 GoPanel

### 真正产出可交付代码的 AI 开发工作区

- 支持使用 **Codex、Claude Code、OpenCode、Aider** 或内置终端执行开发任务。
- 在项目中保存长期会话、原生 CLI 交互、执行历史和可续接上下文。
- 通过受管 Git Worktree 隔离修改，并支持多仓库项目交付。
- 在工作台中查看文件、Git 状态、Diff、历史、分支、Token 用量与审计事件。
- 为项目定义质量检查，并在代码交付前执行验证。
- 通过结构化交付任务完成保存、合并、推送、回退和冲突恢复，而不是把一切藏进不可见的 Shell 脚本。
- 沉淀可复用的项目记忆与执行器配置，同时保持 AI 提供方可替换。

### 手机开发指挥台，而不是缩小版 IDE

GoPanel 手机端围绕“决策”设计，而不是机械复制桌面后台：

- 配对、管理和撤销可信设备；
- 查看项目、会话、任务、资源和安全风险；
- 发送新指令并跟踪结构化执行状态；
- 审查 Git 修改和交付进度；
- 打开预览并验收结果；
- 批准或拒绝高风险动作；
- 停止、重试或恢复需要处理的任务；
- 只在确有必要时使用轻量终端兜底。

手机端与 Web 控制台复用相同的项目、会话、审批和交付契约。业务真相保存在服务端，不让客户端依靠日志或局部状态猜测流程进度。

### 全新服务器默认优先 Rootless 与最小权限

安全是 GoPanel 默认部署模型的一部分，不是后续付费附加项。

- 在全新 Linux 服务器上，安装程序推荐让 GoPanel 以**普通服务用户**运行，并使用 **rootless Podman**。
- 安装流程会准备运行用户家目录、`subuid` / `subgid` 映射、用户级 `podman.socket`、`linger` 和 rootless Socket 路径。
- 运行时诊断会检查 Docker 或 Podman 是否安装、API Socket 是否可达，以及当前是否为 rootless 模式。
- 内置修复动作覆盖 Podman Socket、用户会话、短名镜像源、Compose 和从属 UID/GID 等常见问题。
- 对既有环境继续完整支持 Docker；对全新服务器，则优先推荐最小权限的 rootless Podman 路径。
- 面板日常运行保持非特权；只有确实需要宿主机高权限时，才通过独立的 `gpc` helper 执行边界明确、可审计的受控动作。

当 AI Agent、构建脚本、业务容器和生产基础设施运行在同一台机器上时，这种权限分离尤其重要：每一层只获得完成任务所必需的权限。

### 有治理的自动化，而不是盲目自动化

- 每个开发会话可以选择**手动审批、安全自动或完全自动**策略。
- AI 的危险操作会进入明确的审批记录，而不是静默执行。
- 操作日志、会话审计事件、执行历史和 Token 用量可供追溯。
- 通过角色和目录边界，让非超级管理员只能在受管项目路径中工作。
- Git 凭据和 AI 提供方凭据作为受控资源管理，不把密钥直接写进任务。
- 诊断数据会脱敏常见密钥、认证头、Cookie、Token、私钥和敏感查询参数。
- AI 给出的高风险安全建议必须经过审批，不能直接修改生产环境。

### 开发交付与线上运维共用一个控制平面

GoPanel 同时提供把代码变成在线产品所需要的基础设施：

- **流水线与版本制品**：拉取源码、执行脚本、构建文件或容器镜像、保存日志与记录，并发布有版本身份的制品。
- **交付 Flow**：把确定的代码基线、流水线与面向环境的运行状态连接起来，持续收口端到端交付流程。
- **多节点**：集中观测已注册服务器，并通过主控制平面转发受控的 HTTP 与 WebSocket 操作。
- **内置访问入口与自更新**：无需额外 Web Server 即可运行面板，管理安全访问入口，并通过版本发布通道更新已安装程序。

## 日常生产实用能力

AI 交付是 GoPanel 的差异化能力，但它同时也是一套真正能够长期维护线上业务的实用服务器面板。

### 网站、域名与自动 HTTPS

- 在同一个网站工作台中托管静态站、反向代理和容器化 Web 应用。
- 管理域名、上游节点、访问日志、部署历史、版本切换和应用快照。
- 管理 ACME 账号并直接申请证书，无需再拼装一套独立证书工具链。
- 首次运行自动创建每日证书续签任务，对已开启自动续签且临近到期的证书进行自动续签。
- 对不适合自动签发的场景，也可以上传和管理已有证书。
- 配置证书推送规则，让新签发或续签后的证书自动部署到指定 CDN 服务商与域名。
- 通过完整日志跟踪申请、续签与 CDN 部署过程，HTTPS 不再是一个无法解释的黑盒。

### 真正可用的数据库管理工作台

- 接入本机、远程或从容器中自动发现的 **MySQL、MariaDB 和 PostgreSQL** 服务。
- 创建和删除数据库，管理数据库用户、密码、来源主机与权限。
- 分页浏览表和数据、搜索记录，并直接新增、编辑或删除数据行。
- 按数据库引擎能力查看和修改表结构、字段、索引、视图、存储过程、触发器、序列与数据库元数据。
- 使用内置 SQL 控制台执行查询与管理语句，常用数据库操作无需离开 GoPanel。
- 从粘贴内容或上传文件导入 SQL / CSV，并支持大数据文件分片导入。
- 按字段、过滤条件和结构选项导出 CSV / SQL，支持复制表以及各数据库引擎对应的维护操作。
- 直接完成数据库备份、恢复和数据迁移，不必再切换到另一套数据库管理产品。

### 备份恢复与计划维护

- 为数据库、网站和已安装应用创建与恢复备份，查看执行日志并下载备份记录。
- 可以从已有备份记录恢复，也可以上传备份文件进行恢复。
- 支持本地存储，以及已配置的 **S3、OSS、SFTP、OneDrive、MinIO、COS、KODO 和 WebDAV** 备份目标。
- 在同一个任务中心定时执行数据库备份、证书续签、日志清理和 Shell 任务。
- 每个计划任务都可以启停、立即执行，并查看完整执行记录。

### 容器、应用、文件与宿主机操作

- 管理 rootless Podman 或 Docker 容器、Compose 项目、镜像、镜像仓库、网络、存储卷、日志、端口和资源状态。
- 安装常用数据库、中间件、开发工具和自托管应用，查看连接信息并管理已安装版本与参数。
- 检测并修复 Compose、Podman Socket/用户会话、从属 UID/GID、镜像短名和端口冲突等常见运行问题。
- 通过内置文件管理器上传、下载、编辑、移动、压缩、解压、搜索文件并调整权限。
- 查看进程与端口、管理长期运行服务、检查主机资源，并在确实需要时使用终端直接处理。

### 主机可观测、防火墙、审计与告警

- 在同一个仪表盘查看 CPU、内存、磁盘、网络流量、I/O、进程和关键主机状态。
- 管理防火墙端口、IP 和端口转发规则，同时保持对实际生效策略的清晰掌握。
- 扫描磁盘大文件并对明确目标执行受保护的清理，不依赖不可解释的一键清理脚本。
- 集中查看登录、SSH、操作、终端和系统日志，让重要变更始终可追溯。
- 接收磁盘、容器、节点、证书、Code 与安全事件通知，并通过去抖、静默时段和恢复通知降低告警噪音。

### 让线上问题重新回到 Code

网站智能诊断不是一个更好看的日志查看器，而是一条可闭环的处理链：

- 接收应用事件并采集 Web Server 故障；
- 把关联事件聚合成结构化问题；
- 按阈值执行主动健康探针；
- 在诊断上下文形成前完成敏感数据脱敏；
- 把问题证据、路由、版本上下文和验证要求移交给新的 Code 修复会话；
- 修复完成后重新验证问题。

安全监测也会把网站、认证、SSH 和系统信号整理成结构化风险，用户可以从 Web 或手机端查看和决策。

## 为一人公司而生

当一个人同时维护多个 SaaS、客户项目或自托管服务时，GoPanel 尤其有价值：

- **委派但不失控**：让 Agent 执行开发，同时保留审批边界和完整交付记录。
- **离开电脑也能推进工作**：在手机上发任务、看进度、开预览、做审批和恢复决策。
- **减少工具切换**：把 Coding Agent、Git、CI/CD、容器、网站、证书、数据库与监测连接到一套工作台。
- **数据和控制权留在自己手里**：控制平面、项目上下文、运维数据和凭据均运行在自己管理的基础设施上。
- **降低长期维护成本**：使用 Go 单二进制控制平面、内嵌 Web 资源，以及 Docker/Podman 优先的应用运行模型。
- **自动处理重复运维**：让证书续签、CDN 部署、数据库备份和维护任务自动运行，不再成为个人待办清单的一部分。
- **更快搭好产品环境**：直接安装常用数据库、中间件和自托管应用，不再为每个项目重复手工搭建基础环境。

GoPanel 从一人公司切入，因为这里的需求最直接、决策链最短、工作流也最清晰。随着团队成长，同一套基础还可以通过共享项目、角色、审批边界和多节点管理自然扩展到小型团队。

## 产品原则

- **先结构化状态，再展开原始日志**：摘要和下一步动作是一等信息，终端流作为完整证据保留。
- **自动化必须有控制**：日常步骤自动推进，高风险和不可逆操作暂停等待决策。
- **交付事实不可含糊**：需求、会话、Commit、Release、Deployment、Approval 和 Incident 应始终可追溯。
- **失败必须可以恢复**：重试、冲突、重启和异常都要有明确状态与安全恢复路径。
- **默认最小权限**：rootless 容器与受控提权共同降低用户、脚本和 Agent 的影响半径。
- **连接一条主链，而不是制造新孤岛**：持续增强 Code → Pipeline → Website → Mobile 现有链路。

## 快速开始

Linux 与 macOS：

```bash
bash <(curl -fsSL https://gopanel.run)
```

在全新 Linux 服务器上，请优先采用安装程序推荐的“普通用户运行 GoPanel + rootless Podman”方案。安装程序会自动准备用户命名空间映射和用户级容器运行服务。

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/aihop/gopanel/main/install.ps1 | iex
```

## 界面预览

![GoPanel Preview](./preview.png)

GoPanel 的界面不是一组互不相关的管理页面，而是一个工作控制平面：项目执行、待处理事项、基础设施状态、交付记录和运行证据都围绕用户真正需要做出的决策组织。

## 系统兼容性

- **Linux**：主要生产平台；支持普通用户部署、rootless Podman、Docker、`systemd`、`gpc` 和 `gp-agent`。
- **macOS**：适合本地开发与演示，支持 Podman machine。
- **Windows 10 1809+**：支持本地 GoPanel 开发和基于 ConPTY 的 AI 终端；目前不支持 `gpc` 与 `gp-agent` 宿主管理。

## 技术架构

- **控制平面**：Go + Fiber + SQLite/GORM
- **Web 控制台**：Vue 3 + TypeScript + Naive UI + TailwindCSS
- **手机客户端**：Flutter + Riverpod + GoRouter + Dio
- **容器运行时**：rootless Podman 或 Docker
- **高权限 Helper**：`gpc`，只提供边界明确的本地宿主机动作
- **宿主机 Agent**：`gp-agent`，用于宿主机能力与节点控制
- **发布形态**：Web 资源内嵌的 Go 单二进制

主要目录：

- `app/` — API、模型、服务、仓储、中间件与路由
- `admin/` — Web 管理台与 Code 工作区
- `client/` — Flutter 手机开发指挥台
- `gpc/` — 高权限 Helper
- `gp-agent/` — 宿主机 Agent
- `docs/ai/` — 产品契约、架构决策与流程文档

## 开发与构建

环境要求：

- Go 1.25.1+
- Node.js 20+

本地运行后端：

```bash
git clone https://github.com/aihop/gopanel.git
cd gopanel
go run main.go
```

构建 Web 控制台并写入后端静态资源：

```bash
cd admin
npm install
npm run build:public
```

## 许可证

GoPanel 采用 [GNU General Public License v3.0](./LICENSE) 许可协议。
