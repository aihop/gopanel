# 已解决问题 & 技术债

记录修过的 bug、踩过的坑、确认过的决策。供后续 AI 会话快速恢复记忆。

---

## 已解决的问题

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

---

## 技术债

当前未修复但已确认的问题：

- 无

---

## 架构决策

### 导入分流：文件走 multipart，粘贴走 JSON

- **commit**: `1ec00bc`
- **决策**: 文件选择后不走文本域展示，直接通过 `POST /database/manager/upload` multipart 上传。粘贴内容保留文本域 + JSON 方式。
- **原因**: 大文件展示在文本域会卡死页面，JSON 传输对大文件也不高效。
- **新增端点**: `POST /database/manager/upload`（接受 multipart/form-data）。

### 列编辑 SQL 生成：MySQL 用 CHANGE COLUMN，PG 用多条 ALTER COLUMN

- 编辑字段时 MySQL 生成完整 `ALTER TABLE ... CHANGE COLUMN ... colDef`。
- PostgreSQL 生成 `RENAME COLUMN` + `ALTER COLUMN TYPE` + `SET/DROP NOT NULL` + `SET DEFAULT` + `COMMENT ON COLUMN` 多条语句。
- 在 `useStructureView.ts` 的 `buildColumnDef()` 和 `submitColumn()` 中实现。
