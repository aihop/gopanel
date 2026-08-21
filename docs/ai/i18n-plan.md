# GoPanel v2 国际化迁移方案

> 范围：admin（管理后台 Vue/TS）+ app（Go 后端 API 错误/提示）+ client（Flutter 桌面/手机端）
> 默认语言：中文 zh；备选语言：英文 en
> 提交节奏：按模块分批提交（host / database / ai / container / website / …）

## 1. 现状盘点（已摸清）

### 1.1 后端 Go

- 基础设施已就位：
  - `pkg/i18n/`（基于 `nicksnyder/go-i18n/v2`，Fiber 中间件 + Localizer）
  - `i18n/i18n.go`（`GetMsg / GetErrMsg / GetMsgByKey / GetMsgWithMap`）
  - `buserr/`（`BusinessError` + `New / WithDetail / WithMap / WithName / WithErr`，走 i18n key）
  - `resource/locale/zh.yaml`、`en.yaml`（各 684 行，约 250+ 个 key）
- 已有调用模式：
  - `buserr.New("ErrRecordNotFound")`
  - `buserr.WithDetail("ErrInternalServer", err)`
  - `buserr.WithName("ErrCmdTimeout", name)`
  - 返回链路：`api/*.go` → `service/*.go` → `e.Succ / e.Fail`，错误走 i18n
- **遗留问题**：
  - 仍有大量 `fmt.Errorf`、`errors.New("中文...")`、`fmt.Sprintf("参数错误: %s", ...)` 直接返回，绕过 i18n
  - 部分 SSE/WS 推送消息、日志描述、运维侧文案直接写中文
  - `resource/locale` 缺一些新增业务 key（如 AI 工作区、Code Delivery、Flow 等）
  - 多数错误信息没有走 `Accept-Language`（当前 `GetMsg` 永远 fallback zh），需要走 ctx.lang

### 1.2 前端 admin（Vue 3 + Naive UI）

- 基础设施已就位：
  - `admin/src/i18n/`（基于 `vue-i18n` v9，`getI18NConf` + 合并 `flow.ts`）
  - `admin/src/i18n/locales/zh.ts`（3239 行）/ `en.ts`（3413 行）
  - 另有 `admin/src/i18n/locales/mobile.ts`、`codeProject.ts`、`containerRuntime.ts` 等 split 文件
  - `admin/src/store/i18n.ts`（Pinia 持久化当前 locale）
  - 既有页面已经把视图文案放进 `messages.ts`（`views/Code/codeWorkspaceMessages.ts` 等）
- **遗留问题**：
  - 仍有 190 个 `.vue` 文件包含中文硬编码（多为 columns、empty hint、按钮文案、菜单、placeholder）
  - 137 个 `.ts` 文件含中文硬编码（含一些尚未拆出的 messages 文件）
  - 大量 `meta`/`enum`/`columns` 的 `label/title/placeholder` 是字符串常量，未走 `$t()`
  - 表格里 `tag` 类型映射、状态枚举、菜单、确认弹窗内容仍是硬编码中文

### 1.3 Flutter 客户端

- **零 i18n 基础设施**：`client/lib/l10n/` 不存在，没有 `intl_utils` / `flutter_localizations`，没有 ARB
- 71 个 dart 文件含中文硬编码（含 `Text(...)`, `hintText`, `labelText`, `SnackBar`, `Dialog`, `AppBar.title`, `Tooltip` 等）
- 影响范围：所有 features/ 子模块（auth, ai_workspace, container, database, resources, server, settings, ssl, task_center, website, apps）

### 1.4 体量预估

| 模块 | 待改文件数 | 估算新增 key | 主要工作 |
|------|-----------|------------|----------|
| 后端 app/api + app/service | ~80 | ~150 键 | 提取错误 key、补 en.yaml |
| 前端 admin views/components | ~200 | ~1200 键 | $t() 替换、split messages |
| Flutter client | ~71 | ~600 键 | 接入 ARB + intl，引入 AppLocalizations |
| 合计 | ~350 | ~2000 键 | — |

> **重要提示**：这个量级单轮 PR 风险很高，所以下面采用"按模块分批提交 + 每模块一个 commit"的策略。

## 2. 目标

1. **统一语种切换**：
   - 前端 admin 通过 vue-i18n + Pinia 持久化 locale
   - 后端 API 通过 `Accept-Language`（或 query `lang`）返回对应语言错误
   - Flutter 通过 `Locale` + `AppLocalizations` 切换
2. **禁止硬编码**：模板中所有面向用户的文本走 `$t() / t()` 或后端 i18n key
3. **翻译键规范**：
   - 后端 i18n：`ErrXxx` 错误、`TYPE_xxx` 枚举、`Success/Failed` 通用；模板变量保持 `{{ .detail }}` 等 go-i18n 模板
   - 前端 vue-i18n：树形 namespace（`views.host.disk.title`），动态模板用 `{name}` 占位
   - Flutter ARB：`_PascalCase` key，参数 `{var}` 形式
4. **构建与回归**：
   - 后端：`go build ./...` + `go vet ./...`
   - 前端：`pnpm i18n:check`（新增脚本，扫剩余硬编码）+ `pnpm build`
   - Flutter：`flutter analyze` + `flutter test`

## 3. 迁移原则（硬性）

按 AGENTS.md + 已有工程实践：

- **增量增强**，不要平地起新体系。复用已有的 i18n 基建（`buserr`、`vue-i18n`、`pkg/i18n`）。
- **读前写后**：改文件前 `read_file`；改完后 `read_file` 校验。
- **结果验证**：每改一个模块跑一遍相关构建 / lint；不验证 = 没做完。
- **改动范围**：一次只改一个模块；不混改无关文件。
- **样式不变**：本次不调整 UI，只换文案来源。
- **不越界**：不改业务逻辑，不为可能需求提前设计。
- **错误处理三态**：前端 loading/error/empty 不能只写成功路径。
- **同一项目不引入两套 i18n 写法**（例如 Flutter 全部用 ARB + intl，不再用 Map 字符串字典）。

## 4. 阶段划分

### 阶段 A：补齐基础设施（不破坏现有）

A1. **改造 `i18n.GetMsg`，让后端支持 lang 上下文**
- 新增 `i18n.GetMsgByLang(lang, key, maps...)`
- 改造 `buserr.BusinessError.Error()`：当 `e.skip==true` 直接输出原文；否则尝试从 `request context` 读取 `Accept-Language`
- 引入 Fiber locals `langKey`（在 `pkg/i18n.New` 中间件中读取并写入 locals）
- 增加 `buserr.NewWithLang(lang, key)` 等轻量包装（可选）

A2. **补全 `resource/locale/zh.yaml` / `en.yaml`**
- 扫 `app/api` 与 `app/service` 中已经写中文的错误/提示
- 按模块生成新 key，写入 yaml
- 增加回归测试 `i18n/i18n_test.go`：所有 `.yaml` 中 zh/en key 数量一致

A3. **新增前端 i18n 检查脚本**
- `admin/scripts/i18n-check.mjs`：扫 `admin/src/{views,components}` 下 `.vue/.ts` 的中文残留，列出文件 + 行号
- `package.json` 加 `"i18n:check": "node scripts/i18n-check.mjs"`
- 接入 CI（可选）

### 阶段 B：后端按模块迁移（每模块一个 commit）

| 序号 | 模块 | 主要文件 | 关键点 |
|------|------|----------|--------|
| B1 | 通用 common / e | `e/common.go` | 把所有中文统一改成 key |
| B2 | auth + user | `api/auth*.go`, `api/user.go`, `service/user*.go` | 登录、密码、验证码、用户管理 |
| B3 | host | `api/host*.go`, `service/host*.go` | SSH、monitor、firewall、disk、process、Toolbox |
| B4 | container | `api/container*.go`, `service/container*.go` | 容器、镜像、网络、卷、模板 |
| B5 | website | `api/website*.go`, `service/website*.go` | 网站、SSL、Deployment |
| B6 | database | `api/database*.go`, `service/database*.go`, `api/db_manager.go` | DB 管理器、SQL 操作 |
| B7 | file | `api/file*.go`, `service/file*.go` | 文件管理、回收站、上传 |
| B8 | apps + app | `api/app*.go`, `api/apps*.go`, `service/app*.go` | 应用商店、本地应用 |
| B9 | pipeline + flow | `api/pipeline*.go`, `api/flow*.go`, `service/flow*.go` | 流水线 + 交付 |
| B10 | code + ai_agent | `api/code*.go`, `api/ai_agent*.go` | Code 工作区 + AI 工作区（最大模块） |
| B11 | cronjob + backup + monitor + log | `api/cronjob*.go`, `api/backup*.go`, `api/monitor*.go`, `api/logs.go` | 后台任务 |
| B12 | setting + ssl + notify + node | `api/setting*.go`, `api/ssl*.go`, `api/notify*.go`, `api/node*.go` | 设置 / SSL / 通知 / 节点 |
| B13 | mobile + daemon + dashboard | `api/mobile*.go`, `api/daemon*.go`, `api/dashboard*.go` | 手机端接口 |
| B14 | gp-agent / gpc | `gp-agent/app/api/*.go`, `gpc/cmd/*.go` | 宿主机 agent + helper（CLI 提示可走 i18n） |

每模块提交步骤：
1. 提取模块内硬编码中文 → 写入 yaml
2. 在调用处把 `errors.New / fmt.Errorf / fmt.Sprintf("中文...")` 替换为 `buserr.New("ErrXxx")` / `buserr.WithDetail("ErrXxx", ...)`
3. `go build ./...` 通过
4. `go vet ./...` 通过
5. `git commit -m "i18n(backend): migrate <module> hardcoded strings"`

### 阶段 C：前端 admin 按模块迁移（每模块一个 commit）

每模块提交步骤：
1. 在 `admin/src/i18n/locales/<module>.ts`（新建或扩展）放中文 + 英文 key
2. 在 `admin/src/i18n/index.ts` 注册（已有 `flow.ts` 合并机制可以参考）
3. 改造 `.vue/.ts`：把硬编码中文替换为 `t('xxx')` 或 `$t('xxx')`
4. `pnpm i18n:check` 通过（不再有新增硬编码）
5. `pnpm build` 通过
6. `git commit -m "i18n(admin): migrate <module> hardcoded strings"`

模块顺序（按中文密度从大到小）：

| 序号 | 模块 | 关键文件 |
|------|------|----------|
| C1 | Host | `views/Host/**`（disk/security/process/monitor/firewall/Toolbox） |
| C2 | Database | `views/Database/**`（manager/tables/operations/structure/createTable） |
| C3 | Container | `views/Container/**`（repo/network/volume/image/template） |
| C4 | Website | `views/Website/**`（security/diagnostic/deploy） |
| C5 | Pipeline + Flow | `views/Pipeline/**`, `views/Flow/**` |
| C6 | Code + AI | `views/Code/**`, `views/AIAgent/**` |
| C7 | Apps | `views/Apps/**` |
| C8 | Setting | `views/Setting/**` |
| C9 | Cronjob + Backup + Node + Dashboard + 其他 | 各小模块 |
| C10 | 公共组件 | `components/**`、`router/index.ts`、guards |

### 阶段 D：Flutter 客户端接入

D1. **搭建 Flutter i18n 基础设施**
- 在 `pubspec.yaml` 增加 `flutter_localizations`、`intl`、`intl_utils`（codegen）
- 新建 `client/lib/l10n/`：
  - `app_en.arb`
  - `app_zh.arb`
- 配置 `l10n.yaml`：
  ```yaml
  arb-dir: lib/l10n
  template-arb-file: app_zh.arb
  output-localization-file: app_localizations.dart
  ```
- 在 `MaterialApp` 注入 `localizationsDelegates` + `supportedLocales`

D2. **接入 locale 状态**
- 新建 `client/lib/core/i18n/locale_controller.dart`（Riverpod）
- 默认跟随系统语言；提供手动切换入口（设置页）

D3. **按模块迁移（每模块一个 commit）**
- 顺序：settings → server → auth → resources → ssl → container → database → apps → website → task_center → ai_workspace
- 每模块：把 `Text('中文')` / `'中文'` 改为 `AppLocalizations.of(context)!.xxx` / `l10n.xxx`
- ARB key 命名：`<module><Screen><Element>`，例：`authLoginEmailHint`

D4. **flutter analyze + flutter test 全过**

### 阶段 E：收尾

E1. **CI 兜底**：在仓库 README/CI 增加 i18n 检查脚本（admin + flutter），未来 PR 一旦引入硬编码就 fail
E2. **文档**：补 `docs/ai/i18n.md`，写明 i18n key 命名规范、新增流程
E3. **回归验证**：
- 后端：`go test ./...` + smoke test
- 前端 admin：`pnpm build` + `pnpm i18n:check`
- Flutter：`flutter analyze` + `flutter test`
E4. **最后做一个总 commit**：合并任何 CI 脚本与文档

## 5. 风险与应对

| 风险 | 应对 |
|------|------|
| 单 PR 过大无法 review | 按阶段 + 模块拆 commit，每个 commit 单独可构建 |
| 改 i18n ctx 读取破坏现有 panic 路径 | 先 A1 在保留 `skip` 行为前提下加 lang，灰度验证 |
| Flutter 接入 intl 后 build_runner 报错 | 单独 commit，先只增加依赖，不改文件，下一 commit 再迁移 |
| 表格列/枚举映射在 column 函数内难抽 $t() | 抽到 `useXxxColumns()` composable 里集中处理 |
| 后端 fallback 永远是中文 | A1 阶段加 `buserr.Error()` 时优先 ctx.lang，缺 key 时再 fallback |
| Flutter 端 i18n 增加 release 体积 | 接受，intl 是 Flutter 官方推荐方案 |
| 测试用例里的中文 fixture | 明确**不**算硬编码，扫描脚本排除 `*_test.*` / `*.spec.*` |

## 6. 验证清单（每次模块提交前必跑）

- 后端：
  - `cd /Users/hugh/code/aihop/gopanel_v2 && go build ./...`
  - `go vet ./...`
  - `go test ./app/<module>/...`
- 前端 admin：
  - `cd admin && pnpm install`
  - `pnpm i18n:check`（新增脚本）
  - `pnpm build`
- Flutter：
  - `cd client && flutter pub get`
  - `flutter gen-l10n`
  - `flutter analyze`
  - `flutter test`

## 7. 提交节奏

每次提交包含且仅包含一个模块的改动，commit message 形式：

```
i18n(<scope>): migrate <module> hardcoded strings to i18n keys

- <改动点 1>
- <改动点 2>
- <新增/调整的 yaml/arb 键>
```

scope 取值：`backend | admin | flutter`

## 8. 等待确认

请确认以下事项，确认后我会从阶段 A 开始执行：

1. ✅ 是否同意 Flutter 用 `intl + ARB` 方案（不引入新的状态管理）
2. ✅ 后端改 i18n ctx 是否要兼容旧的"中文 fallback"语义（推荐：保留，缺 key 时回退中文）
3. ✅ 前端 i18n 检查脚本是否要直接 fail build（推荐：是，作为 CI 兜底）
4. ✅ 是否同意按"模块分批 commit"而不是一次性提交
5. ⏸ Flutter 端 pubspec 增加依赖是否 OK（推荐：是，这是 Flutter 官方方案）

> 一旦你确认 1~4（5 可同确认），我会按 A → B → C → D → E 顺序推进，并在每模块完成后向你汇报。