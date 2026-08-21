# admin 国际化扫描脚本

`scripts/i18n-check.mjs` 扫 `admin/src/views` 与 `admin/src/components` 下的 `.vue` / `.ts`，找出仍存在的硬编码中文字符串，用于 CI 兜底。

## 使用

```bash
# 查看当前所有硬编码（无基线，失败即报告全部）
pnpm i18n:report

# 与基线对比：基线内的允许保留，新增即失败
pnpm i18n:check

# 把当前扫描结果固化为基线（A3 落地时执行一次，后续迁移后刷新）
pnpm i18n:emit-baseline
```

## 基线文件

- 路径：`admin/scripts/i18n-baseline.json`
- 内容：当前所有硬编码中文行的 `[{file, line, text, chars}]` 数组
- 提交策略：每次 C1~C10 模块迁移后，必须用 `pnpm i18n:emit-baseline` 重新生成，并把变更同步进 commit

## 设计要点

1. **跳过 `i18n/locales/`**：翻译源文件本身就是中文，不应误报
2. **跳过测试文件**：`*.spec.*` / `*_test.*` / `*.test.*` 是 fixture，允许硬编码
3. **跳过 `//` 行尾注释**：通过 `stripCommentTail` 避免 `import xxx // 注释` 这种被误报，但会保留在字符串字面量里的注释
4. **CJK 检测范围**：`U+4E00-U9FFF`（基本汉字）+ `U+3400-U4DBF`（扩展 A 起点）
5. **CI fail 策略**：基线模式下，只有新增硬编码才 fail；存量随模块迁移逐步消化

## 迁移工作流

```
# 1. 在 i18n locales/zh.ts、en.ts 添加 key
# 2. 在 .vue / .ts 用 t('your.key') 替换硬编码
# 3. 重新生成基线：
pnpm i18n:emit-baseline
# 4. 提交（包含 src 改动 + i18n locales + baseline 三类文件）
git commit -m "i18n(admin): migrate <module> hardcoded strings"
```