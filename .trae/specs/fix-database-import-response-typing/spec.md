# 数据库表导入响应类型修复 Spec

## Why
当前数据库管理页在构建时，`DatabaseTablesView.vue` 读取导入接口返回值时被推断为 `unknown`，导致 TypeScript 构建失败。
这个问题会阻塞前端发布，同时会让导入成功数量和失败提示的读取逻辑失去类型约束。

## What Changes
- 为数据库表导入接口补齐明确的响应类型约束
- 调整 `DatabaseTablesView` 中导入结果处理逻辑，按已知字段安全读取返回值
- 保持现有导入交互、成功提示和失败提示文案不变

## Impact
- Affected specs: 数据库管理、表导入、前端构建稳定性
- Affected code: `admin/src/views/Database/src/components/DatabaseTablesView.vue`、`admin/src/api/modules/database.ts`

## ADDED Requirements
### Requirement: 表导入接口结果必须可类型化读取
系统 SHALL 让数据库表导入页面能够以明确类型读取接口返回的 `code`、`msg` 和导入结果数据，而不是依赖 `unknown`。

#### Scenario: 导入成功
- **WHEN** 用户在数据库管理页执行表数据导入，且后端返回成功结果
- **THEN** 前端可以安全读取响应 `code`
- **AND** 前端可以安全读取已导入条数并展示成功提示

#### Scenario: 导入失败
- **WHEN** 用户在数据库管理页执行表数据导入，且后端返回失败结果
- **THEN** 前端可以安全读取响应 `msg`
- **AND** 页面展示原有失败提示而不触发 TypeScript 类型错误

## MODIFIED Requirements
### Requirement: 数据库管理页必须通过前端构建
数据库管理页中的表导入逻辑 MUST 在 TypeScript 构建阶段通过检查，不得因为接口返回值被推断为 `unknown` 而导致构建失败。

#### Scenario: 执行前端构建
- **WHEN** 对管理台执行 TypeScript 或构建检查
- **THEN** `DatabaseTablesView.vue` 不应再出现 `Property 'code' does not exist on type 'unknown'` 一类错误

## REMOVED Requirements
