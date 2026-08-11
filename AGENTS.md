# GoPanel v2 AGENTS

## 一句话定位

`gopanel` 是 GoPanel 的主控项目，负责 Web 管理台、AI 工作区、任务/日志、网站/容器/流水线管理，以及为手机端提供可复用的控制平面能力。

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

## AI 行为约束（防幻觉）

以下规则是硬性约束，违反它们意味着产生无效代码。

### 文件粒度

- 任何单个文件（.go / .vue / .ts）不超过 **500 行**。超过必须拆分。
- 后端 400 行强提醒，600+ 必须拆分。前端同样。
- 拆分方案见 `docs/ai/refactoring.md`。

### 读前写后

- 改任何文件前，先用 `read_file` 确认当前内容，**不要凭记忆写 patch**。
- 改完后，用 `read_file` 确认 patch 正确落地。
- 新增 import 时确认该包/文件确实存在于项目中。

### 结果验证

- 每个工具调用返回后，检查输出再走下一步。
  - 文件读：确认行号跟你预期 patch 的位置一致
  - shell 命令：检查 stdout/stderr，不只看退出码
  - 搜索：确认匹配结果确实是你想找的
- 不验证 = 没做完。不要在没有验证的情况下声称改动生效。

### 改动范围

- 一次改动只改任务相关的文件。不改动不应出现在提交中。
- 不要因为"顺便"修改不相关的文件，即使你觉得是小问题。
- 不要假设一个文件路径或 API 存在——先查再改。

### 模式复用

- 新增功能前，先读同类模块的实现方式，保持一致风格。
- 不要在同一个项目里引入两套不同的写法。
- 项目已有工具函数不要重新实现；改代码前先确认用的是什么工具链（axios、useTable、renderIcon 等）。

### 国际化

- 前端模板中所有面向用户显示的文本必须使用 `$t()` 或 `t()` 翻译 key，**不能硬编码中文或英文**。
- 新增翻译 key 时留意 `admin/src/locale/` 下的中英文文件，两边都要加。
- 后端提示信息可以通过 `i18n` 中间件或直接在响应中返回中文，前端统一用 key 做多语言。

### 错误处理

- 前端 API 调用必须覆盖 loading / error / empty 三种状态，不能只写成功路径。
- `try/catch` 后要在 UI 给用户反馈（`message.error`），不能静默吞掉异常。
- 后端错误统一通过 `e.Succ()` / `e.Fail()` 包装返回，不直接 `return c.SendString()` 或 `c.JSON(err)`。
- 不要用 `panic` 替代错误返回。

### 不越界

- 不要改任务不相关的模块。比如修数据库时不要顺手改容器代码，即使你觉得逻辑相似。
- 不要加当前不需要的抽象层、接口、配置项。解决现存问题，不为可能的需求提前设计。
- 不要引入新的第三方依赖，除非现有工具链确实无法满足需求。

### 样式一致

- 前端样式优先使用 TailwindCSS，不要混入手写 CSS 和 Tailwind 两种方式。
- hand-written CSS 只在 `&lt;style scoped&gt;` 中处理 Tailwind 难以覆盖的极少数场景（如 `:deep()` 穿透）。
- Naive UI 组件的尺寸控制用 `style` prop，不要用 Tailwind 的 `w-` / `h-` 类（原因见本文 Naive UI 约定段）。

### 数据流确认

- 改任何组件前先搞清楚数据从哪里来：props / API 响应 / composable / store / localStorage。
- 不要假设一个响应式变量在父组件中存在——先查父组件的 template 和 script 确认。
- 新增接口字段时确保后端 handler、service、前端 API 定义、组件 props 四层同步更新。

### 防止死循环

- **单次修复不超 3 轮**。同一个问题尝试 3 次修复仍未通过验证时，停止尝试并询问用户。
- **一次只改一个方向**。不要同时改多个不相关的点，改完一个验证再改下一个。
- **工具重试换参数**。工具调用失败时，不要用完全相同的参数重试。先分析错误原因，调整后再试。
- **改完就往前走**。验证通过后标记完成，不要回头再去重读或重构刚改好的文件。

## 已解决问题（详细记录见 `docs/ai/knowledge-base.md`）

- 容器列表缓存竞争条件 — `f578b1c`
- Contextx 不支持平面字段 — `8400702`
- OperationsView 缺失导入导致空白 — `40ef5ed`
- NModal 宽度用 Tailwind class 无效 — `4144358`
- DataView v-show 残留 DOM — `ece6439`
- SQL 导入被注释检查错误跳过 — `b8dd282`

## 发布版本（自动执行，不再询问）

**固定触发提示词：`发布新版本`。**

当用户说“发布新版本”“发布一个版本”或“发布到 GitHub/GitCode”时，直接完成整个官方发布流程，不询问版本号，不做二次确认：

1. 确定版本号：
   - 用户明确指定版本号时直接使用。
   - 用户未指定时，通过 GitHub Release/远程 Tag 获取最新正式语义化版本，默认递增补丁号，例如 `1.3.2 → 1.3.3`。
   - 忽略草稿、预发布版本和非语义化 Tag；不得仅依赖本地 Tag，因为本地可能尚未同步。
2. 检查发布基线：
   - 确认当前分支、最新提交、远程同步状态和工作区状态。
   - 若存在待发布的任务相关代码，先验证、提交并推送；不得把无关改动混入发布提交。
   - 确认目标版本在 GitHub/GitCode 尚未正式发布；如已存在则按脚本的幂等更新逻辑继续，不重复创建冲突记录。
3. 检查环境（不等待用户确认）：
   - `jq` 已安装。
   - `GITCODE_TOKEN` 环境变量或 `.env` 中已配置。
   - `GOPANEL_ADMIN_KEY` 在 `.env` 中已配置，用于同步 `gopanel.cn` changelog。
   - 优先使用有效的 `GH_TOKEN`/`GITHUB_TOKEN`；若 `gh auth status` 失效，则从系统 Git credential helper 读取 `github.com` 凭据并仅在当前命令中注入 `GH_TOKEN`，不得输出或写入密钥。
4. 执行 `bash build.sh <VERSION>` 生成全部默认平台包，并等待完整结束：
   - macOS ARM64 / AMD64
   - Linux ARM64 / AMD64
   - Windows AMD64
   - 若构建被本次代码的明确错误阻断，做最小修复、验证、提交、推送后自动重新构建；依赖审计警告不等于构建失败，不在发布过程中擅自升级依赖。
5. 非交互执行 `PUBLISH_POST=1 bash publish.sh <VERSION>`，自动回答脚本确认提示，同时发布到 GitHub、GitCode，并同步 changelog 到 `https://gopanel.cn/api/admin/posts`。
6. 发布后必须核验：
   - GitHub Release 为非草稿、非预发布，Tag 和全部平台附件正确。
   - GitCode Release Tag 和全部平台附件正确。
   - 官网 changelog 返回 2xx 且记录 key 为 `v<VERSION>`。
   - 仓库远程 URL 已恢复为无凭据地址，发布提交已推送。
7. 最终只向用户汇报版本号、GitHub/GitCode Release 链接、官网 changelog 同步结果和必要警告。

**默认仓库**：`aihop/gopanel`。
**默认目标**：GitHub + GitCode 同时发布。
**changelog 同步**：必须设置 `PUBLISH_POST=1`，否则不会自动调用 `register_changelog`。

## 文档索引

- `docs/ai/README.md` — 模块入口 + 快速参考
- `docs/ai/knowledge-base.md` — 已解决问题的技术债记录
- `docs/ai/refactoring.md` — 前后端文件拆分方案

## Git 提交约定（硬性规则）

**提交和推送是强制性的，不需要询问用户是否要提交。** 代码改动验证通过后，直接执行完整提交流程。

- 提交粒度：一个功能 / 一个 bugfix 一次提交
- 提交信息：首行用 `type: 标题` 概括（feat/fix/refactor/style/docs/chore），后续用 `-` 列表写关键改点
- 不做 `git add .` 全量提交，只加本次改动相关的文件
- 不改动文件不应出现在提交中

**推送流程（必须执行，不可跳过）：**

1. `git add <改动文件列表>` — 只添加本次变更的文件
2. `git commit -m "..."` — 单行标题 + 列表体
3. `git remote set-url origin https://<用户名>:<密码>@codeup.aliyun.com/64dc6e9a9210862005710a57/gopanel.git` — 设置带认证的远程 URL
4. `git push 2>&1` — 推送到远程
5. `git remote set-url origin https://codeup.aliyun.com/64dc6e9a9210862005710a57/gopanel.git` — 恢复无凭据的远程 URL，避免泄露

**认证凭据**（会话有效期内使用）：
- 用户名：`hughold`
- 密码/token：`GGfuren520`

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
