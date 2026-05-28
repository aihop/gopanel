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
