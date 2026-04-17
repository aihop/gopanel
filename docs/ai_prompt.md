# GoPanel 项目开发提示词 (For AI Assistants)

## 项目简介
GoPanel 是一个基于 Golang 和 Vue 的容器化服务器管理面板（类似于 1Panel），支持通过 Web 界面来管理服务器环境、网站、数据库和容器化应用商店。

## 技术栈
**后端**:
- 语言：Golang 1.24.0
- Web 框架：Go fiber v3.0.0-beta.4
- 数据库：SQLite (使用 gorm / glebarez/sqlite)
- 容器管理：Docker 28.2.2 / Docker Compose V2

**前端**:
- 框架：Vue 3 + Vite
- 语言：TypeScript
- UI 组件库：Naive UI

## 目录结构核心说明
```text
.
├── admin/          # (重要) 管理面板前端项目源码目录 (Vite + Vue3)
├── app/            # 后端核心业务逻辑目录 (MVC分层架构)
│   ├── api/        # 控制器层 (Controller)，处理 HTTP 请求
│   ├── dto/        # 数据传输对象 (Data Transfer Object)，包含 request 和 response 结构体定义
│   ├── model/      # 数据库模型定义 (GORM Entities)
│   ├── repo/       # 数据访问层 (Repository)，封装数据库操作
│   ├── service/    # 业务逻辑层 (Service)
│   └── router/     # 路由配置注册
├── apps/           # 容器化应用商店的 Docker Compose 模板库 (如 mysql, redis, clickhouse 等)
│                   # 包含各个应用的 data.yml (表单配置) 和 docker-compose.yml 模板
├── cmd/            # 命令行入口，如 cmd/server.go 定义了服务启动参数和工具命令
├── constant/       # 全局常量和变量定义，包含路径、状态码、内置服务名等
├── init/           # 服务启动时的初始化脚本 (包括数据库、Caddy、缓存、系统目录等)
├── public/         # 静态资源文件目录 (前端 admin 编译后的产物放置于此)
├── utils/          # 各种工具类封装 (Docker 操作、文件操作、命令执行等)
└── main.go         # 后端服务主入口文件
```

## 关键运行与开发机制
1. **应用商店机制 (`apps/` 目录)**:
   - `apps/` 目录下存放的是系统支持一键安装的第三方应用模板（例如 `mysql`）。
   - 安装逻辑：面板会读取 `data.yml` 中的参数配置，结合用户的输入生成真实的 `.env` 文件，并调用系统内置的 Docker 工具，通过 `docker compose up -d` 部署容器。
   - 容器网络：大多数应用默认使用 `gopanel-network` 外部网络。

2. **前端开发构建 (`admin/` 目录)**:
   - 前端代码主要在 `admin/` 目录下进行开发 (`npm run dev`)。
   - 构建发布时 (`npm run build`)，生成的静态文件会被分发到根目录下的 `public/` 文件夹中，由后端的 Fiber 框架提供静态服务代理。

3. **后端开发提示**:
   - 本地开发直接运行 `go run main.go`。
   - 默认数据存储位置在 `/opt/gopanel`（通过 `init.yaml.dev` 或环境变量控制），因此建议使用 root 权限或者确保对该目录有读写权限。
   - 路由入口统一由 `app/router/app.go` 管理，按照功能模块划分为多个子路由（如 `AppsRouter` 等）。

## AI 辅助工作指令
当接收到关于该项目的任务时，请遵循以下原则：
1. **参考开源标杆**：该项目的后端设计和业务逻辑大量参考并借鉴了 `1panel` 项目的架构。在实现如应用商店、容器管理、备份恢复等核心业务逻辑时，请尽量**参照 1panel 的实现方式和数据流转逻辑**，确保架构的一致性。
2. **前后端开发定位**：
   - 如果涉及**前端管理面板**修改，请直接前往 `admin/` 目录查找组件和视图。
   - 如果涉及**接口及业务逻辑**，请前往 `app/` 目录，按照 `router -> api -> service -> repo -> model` 的链路进行追踪。
   - 如果涉及**Docker或第三方应用配置**，请检查 `apps/` 目录下的模板和 `app/service/app_install.go` 的相关逻辑。
3. **技术规范**：遵循 Go fiber v3 的语法特性（不要使用 v2 语法），并使用 GORM 处理数据库操作。
4. **架构设计规约（云服务凭证解耦）**：
   - `DnsAccount` (dns_account_id)：专门负责**“域名归属”与“DNS 解析层”**的授权（如 Cloudflare，腾讯云 DNSPod 等）。用于 SSL 证书的 DNS-01 验证、域名解析记录的管理。
   - `CloudAccount` (cloud_account_id)：专门负责**“业务资源层”**的授权（如阿里云 CDN，对象存储 OSS，云主机等）。用于 CDN 域名拉取、证书部署推送 (Auto-Push)、自动备份上传等业务。
   - **严禁混用**：在任何新业务开发中，若涉及域名解析相关操作必须关联 `DnsAccount`；若涉及实体云资源调度（如存储、CDN）必须关联 `CloudAccount`，以支持跨云部署的复杂架构。

### 🚨 AI 行为约束（防止死循环与过度修改）
- **避免无效重试**：如果在使用 linter 或工具执行修改时，同一文件或同一类型错误修改超过 3 次依然报错，**必须立即停止对该文件的操作**，将错误信息和目前状态整理后返回给用户，等待人工干预。
- **克制“自我进化”与过度重构**：你的任务是**实现用户需求**，而不是进行未经授权的代码重构或依赖升级。除非用户明确要求（例如“优化这段代码”、“更新此依赖”），否则**绝对不要**主动删除看似未使用的包（如之前误删 `utils/captcha` 的情况）、重写项目基础架构或修改与当前任务无关的全局文件。
- **修改后必须验证**：每次修改关键逻辑后，尽量通过读取关联文件或运行编译命令来验证修改的上下文是否自洽，而不是盲目覆盖。
