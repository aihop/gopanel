# Tasks
- [x] Task 1: 收口数据库表导入接口的响应类型
  - [x] SubTask 1.1: 检查 `admin/src/api/modules/database.ts` 中导入接口的返回类型定义
  - [x] SubTask 1.2: 为导入结果补齐可被组件直接消费的类型约束

- [x] Task 2: 调整数据库表导入页面的结果读取逻辑
  - [x] SubTask 2.1: 更新 `admin/src/views/Database/src/components/DatabaseTablesView.vue` 中导入成功分支的类型安全读取
  - [x] SubTask 2.2: 更新导入失败分支的错误消息读取，保持现有交互不变

- [x] Task 3: 验证构建错误已消除
  - [x] SubTask 3.1: 运行与该页面相关的 TypeScript 或构建检查
  - [x] SubTask 3.2: 确认不再出现 `code`、`data`、`msg` 读取 `unknown` 的报错

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
