# 文件拆分方案（重构参考）

当单个文件超过阈值时的推荐拆分方式。
后端 400 行强提醒，600+ 应拆分。前端同样。

## 硬门禁

- `bash scripts/check-file-size.sh` 扫描受版本控制的 `.go`、`.vue`、`.ts` 文件，默认上限 500 行。
- `.file-size-baseline` 冻结存量超限文件的精确行数；存量文件只能缩短，缩短时必须同步降低或移除基线。
- 新文件不得加入基线，已有基线不得提高。CI 会与目标分支比较并阻止放宽规则。
- `bash scripts/install-git-hooks.sh` 启用 pre-commit 暂存区检查；`build.sh` 和远程 CI 使用同一脚本。
- 语言包、自动生成声明和 Wails 生成代码显式豁免，其他例外必须修改检查脚本并接受代码审查。

---

## 前端拆分

拆分顺序：纯视图区块组件 → 表格列工厂/选项常量/格式化函数 → 页面级 composable

| 文件 | 推荐拆分 |
|------|---------|
| `views/Pipeline/src/CreatePipelineModal.vue` | 主容器 + 表单区块组件 + `pipelineForm.ts` |
| `views/Host/files.vue` | 工具栏、筛选区、表格列工厂、路径导航 composable、文件动作 composable |
| `views/Host/firewall.vue` | 概览区、规则工作台、列工厂、规则服务函数 |
| `views/Host/process.vue` | 搜索工具栏、详情抽屉、进程/网络列配置，主页面保留 WebSocket 与分页调度 |
| `views/Host/Toolbox/Daemon.vue` | 顶部概览区、表格列工厂、Agent 状态 composable、守护进程动作 composable |
| `views/Container/image/index.vue` | 先抽顶部工具栏，再拆表格逻辑 |
| `views/Container/container/index.vue` | 列表工具栏、表格列工厂、列表数据 composable，主页面保留弹窗与批量操作编排 |
| `views/Container/container/operate/index.vue` | 基础信息、端口网络、挂载、进阶参数等表单区块 + 端口列配置/表单 helper |
| `views/Container/compose/index.vue` | 列表工具栏、创建抽屉、删除确认弹窗、表格列工厂 + 创建流程 composable |
| `views/Container/setting/index.vue` | 运行时状态头部、基础配置区块、全部配置编辑区、修复弹窗 + composable |
| `views/Apps/components/AppsAll.vue` | 卡片列表、安装弹窗、详情抽屉、日志弹窗 + 安装流程 composable |
| `views/Apps/components/AppsInstalled.vue` | 已安装卡片列表、详情抽屉、卸载确认弹窗、日志弹窗 + 日志修复流 composable |
| `views/Dashboard/components/StatusCard.vue` | 基础信息概览条、CPU/内存/负载资源卡片、磁盘/GPU 面板 + 格式化 helper |
| `views/Website/components/AccessLogDrawer.vue` | 抽屉壳、日志面板、详情弹窗、IP 统计弹窗 + helper/composable |
| `views/Website/SSL.vue` | 页面头部、证书弹层、表格列工厂 + 分层 composable |
| `views/Database/src/DatabaseManager.vue` | 左侧导航、顶部上下文条、`useDatabaseManager` composable |
| `views/Database/src/components/DataView.vue` | 表列表/记录分页/筛选/删除 → `useDataView` composable |
| `views/Database/src/components/StructureView.vue` | 字段/索引操作、结构摘要 → `useStructureView` composable |
| `views/Database/src/components/OperationsView.vue`, `SearchView.vue` | 优先增强交互反馈，再决定是否拆 composable |
| `views/Database/src/components/DatabaseWorkspaceHeader.vue` | 各数据库子页共享的上下文头部 |

---

## 后端拆分

拆分顺序：handler 与 helper 分离 → 主流程与日志/预览/状态探测分离 → CRUD 与 repair/compat/utils 分离 → runtime/provider 特化分离。使用同 package 新文件 + 迁移整组函数的方式。

| 文件 | 推荐拆分 |
|------|---------|
| `repo/pipeline.go` | `pipeline_repo`、`pipeline_record_repo`、`release_repo`、`release_repo_repair` |
| `api/ai_agent_rest.go` | group、task、session、preview 四组 |
| `api/container.go` | container 主体、network、volume、compose |
| `api/app_install.go` | install 主入口、installed、本地安装、日志流 |
| `api/file.go` | manage、content、transfer 三层 |
| `service/pipeline.go` | step 函数抽到独立文件，主文件保留编排入口 |
| `service/pipeline_utils.go` | detect、execute、runner、image_detect、artifact 五层 |
| `service/db_manager.go` | 表结构查询、记录操作、核心 service |
| `service/pipeline_application.go` | 发布记录流、元信息 helper、主 service |
| `service/app_deploy.go` | website 发布动作、runtime 探测/切换、部署元信息 helper |
| `service/apps_utils.go` | runtime 状态、compose env 校验、安装文件流程、compose compat、命令展开 helper |
| `service/container.go` | container 壳文件、列表链路、运行时管理 → 再二次细分 |
| `service/container_utils.go` | stats、config、inspect 三组 helper |
| `service/container_logs.go` | 主日志流、GPC 直连、journald/podman 日志兜底 |
| `service/container_docker.go` | runtime 状态读取、daemon 配置写入、运行时控制 |
| `service/container_network.go` | 主接口、podman 适配、static IP 兜底 |
| `service/website_engine.go` | 部署主流程、镜像/入口探测、runner 配置 helper |
| `service/firewall.go` | 主入口、rule loader、rule filter/normalize |
| `service/caddy_utils.go` | 基础读写、server block 修改、查询/解析 helper |
| `service/file.go` | tree、content、gpc 适配、主 service |
| `service/ssl.go` | 主 service、ACME 签发流程、证书文件/域名 helper |
| `api/ai_agent.go` | 主 WebSocket handler、preview/workdir helper、workspace/runtime helper、PTY 输出转发 helper |

拆分后至少做一项验证：`gofmt` → `go test` → 文件诊断。

---

## 前端重构注意事项

- 保持现有接口、事件名、弹窗方式和提交流程不变。先做结构收口，再做行为优化。
- 避开正在推进的业务改动文件，先处理未脏的大文件，减少误伤。
- 遇到手写 CSS 优先改写为 TailwindCSS，视觉效果一比一还原。
- Naive UI 组件的宽度控制用 `style` 而非 `class`（参见 knowledge-base NModal 条目）。
