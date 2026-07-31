# 已解决问题 & 技术债

记录修过的 bug、踩过的坑、确认过的决策。供后续 AI 会话快速恢复记忆。

---

## 已解决的问题

### 2026-07-31: 数据库连接失败被显示为空列表

- **问题**: MySQL/PostgreSQL 服务器连接或查询失败时，数据库列表显示为空，用户无法区分无数据库和连接异常。
- **根因**: `DatabaseRepo.List()` 静默忽略单个服务器的连接与查询错误，API 仍返回成功空数组。
- **修复**: 列表响应新增服务器级 `warnings`，保留其他服务器的可用数据；前端展示部分失败告警与脱敏错误详情。
- **验证**: 覆盖部分成功、服务器筛选、分页总数和密码脱敏回归测试。

### 2026-05-28: 容器列表缓存竞争条件

- **commit**: `f578b1c`
- **问题**: 勾选容器启动后调用列表接口，容器仍显示关闭状态。刷新页面才正确。
- **根因**: 缓存版本竞争。容器操作完成时调用 `invalidateContainerListCaches()` 只清了缓存条目但没停正在进行的后台刷新。刷新完成后又把旧数据写回缓存。
- **修复**: 给 `containerListViewCache` 加 `version` 字段。`invalidateContainerListCaches()` 时推进版本号、关闭 waitCh、重置 refreshing。`refreshContainerListView()` 完成后对比版本号，不一致则放弃写入。

### 2026-05-28: Contextx 不支持平面字段

- **commit**: `8400702`
- **问题**: 切换服务器后数据库列表没过滤，返回了所有服务器的所有数据库。
- **根因**: 前端发 `{ server_id: val }` 平面字段，后端 `Contextx` 没有这个 JSON 字段，被 `json.Unmarshal` 静默丢弃。后端期待 `{ wheres: [{ field: "server_id", rule: "eq", val: "1" }] }`。
- **修复**: 前端修改 API 调用，使用 `wheres` 数组格式。

### 2026-05-28: OperationsView 缺失导入导致空白

- **commit**: `40ef5ed`
- **问题**: 点击「操作」标签页显示空白。
- **根因**: `OperationsView.vue` 模板使用了 `NIcon` 和 `renderIcon()` 但未 import。`renderIcon` 为 undefined 时调用导致 JS 异常，组件渲染中断。
- **修复**: 补上 `import { NIcon } from 'naive-ui'` 和 `import { renderIcon } from '@/utils'`。

### 2026-05-28: NModal 宽度用 Tailwind class 无效

- **commit**: `4144358`
- **问题**: `class="w-[560px]"` 对 NModal 不生效。
- **根因**: Naive UI 用 CSS-in-JS 内联控制关键尺寸。`class` 挂在外层 wrapper（全屏遮罩层），不是实际卡片容器。
- **修复**: 改用 `style="width: 560px"`。

### 2026-05-28: DataView v-show 残留 DOM

- **commit**: `ece6439`
- **问题**: 切换标签页时其他子视图的浏览记录还在。
- **根因**: DataView 用 `v-show`（CSS 隐藏），其他子视图用 `v-if`（销毁重建）。切换到结构/SQL 标签时 DataView 的 DOM 仍然保留。
- **修复**: DataView 改为 `v-if`。`dataViewRef` 访问加 `nextTick` 确保组件挂载后引用。

### 2026-05-28: SQL 导入被注释检查错误跳过

- **commit**: `b8dd282`
- **问题**: SQL 导入不生效。
- **根因**: `execSQLImport` 外层 `HasPrefix("--")` 把以注释开头的整段 SQL 跳过了。内层循环其实已正确处理注释行。
- **修复**: 删除外层 `HasPrefix("--")` 检查。CSV 导入同时补充了 `\r\n` 换行符兼容。

### 2026-07-25: SettingRepo.UpdateOrCreate 无法把值清空

- **commit**: `f94d34f`
- **问题**: 节点管理页点「关闭只读接入」提示成功，但主控依然能用旧令牌拉到本机摘要——权限没真的收回。
- **根因**: `repo/setting.go:127` 的 `UpdateOrCreate` 实现是 `Assign(model.Setting{Key: key, Value: value}).FirstOrCreate(...)`。GORM 的 `Assign` 传结构体时会**跳过零值字段**，所以 `Value: ""` 被静默忽略，行里的旧值原封不动。
- **修复**: 清空场景改用 `Update(key, "")`——它内部走 `Updates(map[string]interface{}{...})`，map 不受零值跳过影响。
- **注意**: 这个坑对 `UpdateOrCreate` 的**所有**调用方都成立。任何"把某个 setting 清空"的需求都不能用它，必须走 `Update`。已确认只改了节点这一处，其他调用点未逐一排查。

### 2026-07-25: 证书剩余天数用 -1 当哨兵会撞车

- **commit**: `f94d34f`
- **问题**: 节点摘要里「没有证书」和「证书过期 1 天」返回同一个值。
- **根因**: `CertMinDays` 用 `-1` 表示"无证书"，但已过期证书的剩余天数本身就是负数（实测某张证书返回 `-12`），`-1` 是合法值。
- **修复**: 新增独立的 `CertTotal` 字段判空，`CertMinDays` 只表达天数，负数明确表示已过期。

---

## 技术债

当前未修复但已确认的问题：

- **macOS 上磁盘水位取不到**：`service/dashboard.go:251` 的 `loadDiskInfo()` 在 macOS 下对所有挂载点（含 `/`）都报 `no such file or directory`，导致 `DiskData` 为空。影响：节点摘要的 `diskMaxPercent` 恒为 0，磁盘告警在 Mac 上不会触发。Linux 生产环境不受影响，未修。

---

## 架构决策

### 导入分流：文件走 multipart，粘贴走 JSON

- **commit**: `1ec00bc`
- **决策**: 文件选择后不走文本域展示，直接通过 `POST /database/manager/upload` multipart 上传。粘贴内容保留文本域 + JSON 方式。
- **原因**: 大文件展示在文本域会卡死页面，JSON 传输对大文件也不高效。
- **新增端点**: `POST /database/manager/upload`（接受 multipart/form-data）。

### 多节点：先做只读观测面，代理面单独分期

- **commit**: `f94d34f`、`56cc65e`
- **决策**: 多服务器管理拆成两期。**A 观测面**（已完成）：主控只读拉取各节点摘要并集中展示，不代理任何操作。**B 操作面**（未开工）：`/api/node/:id/*` 反向代理，让现有页面直接操作远程节点。
- **原因**: 「服务器多了不好管理」的痛点九成是"不知道该看哪台"，A 独立解决它，且对现有代码零侵入。B 的成本是 A 的两倍多，长尾在前端——实测有约 20 处绕过 axios 手工拼 URL 的地方（5 处 `new WebSocket`、约 10 处 `__VITE_API_URL__` 拼接、`PortJump.vue` 直连 `systemIP`），漏一处就是"这个页面静默地还在操作本机"。
- **节点鉴权**: HMAC-SHA256(token, ts+nonce+method+path)，令牌不随请求传输，允许节点跑明文 HTTP。nonce 只让签名不可预测，**没有服务端查重**，300 秒窗口内可重放——只读接口可接受，挂到写接口前必须补 nonce 查重。
- **安全入口**: 没有把 `/api/node/summary` 加进 `middleware.shouldBypassEntrance` 白名单。宁可让用户在节点配置里填一次安全入口，也不削弱既有的安全入口控制。
- **未做**: 顶栏「当前节点」chip 和高危弹窗带节点名。A 阶段没有"当前节点"概念（面板始终操作本机），这两件事要等 B 的切换能力落地才有意义。
- **预留**: `node.connect_mode` 字段当前只有 `direct` 一个取值，反连隧道（NAT 场景）走同一抽象接入。

### 列编辑 SQL 生成：MySQL 用 CHANGE COLUMN，PG 用多条 ALTER COLUMN

- 编辑字段时 MySQL 生成完整 `ALTER TABLE ... CHANGE COLUMN ... colDef`。
- PostgreSQL 生成 `RENAME COLUMN` + `ALTER COLUMN TYPE` + `SET/DROP NOT NULL` + `SET DEFAULT` + `COMMENT ON COLUMN` 多条语句。
- 在 `useStructureView.ts` 的 `buildColumnDef()` 和 `submitColumn()` 中实现。
