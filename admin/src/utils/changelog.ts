import { h, type VNode } from "vue"
import { NPopover, NTag } from "naive-ui"

/**
 * 更新说明来自仓库提交信息，属于外部输入：
 * 这里一律用文本节点渲染，禁止 v-html / innerHTML。
 */
export const parseChangelogLines = (changelog?: string): string[] => {
  if (!changelog) return []
  return changelog
    .split("\n")
    .map(line => line.trim())
    .filter(Boolean)
}

/** 表格单元格：显示首条 + 条数，悬浮展开全部 */
export const renderChangelogCell = (changelog?: string): VNode => {
  const lines = parseChangelogLines(changelog)
  if (!lines.length) {
    return h("span", { class: "text-slate-400 text-xs" }, "-")
  }

  const preview = h("div", { class: "flex flex-col gap-0.5 min-w-0 cursor-pointer" }, [
    h("div", { class: "truncate text-xs text-slate-700" }, lines[0]),
    lines.length > 1
      ? h(NTag, { size: "tiny", type: "info", bordered: false }, { default: () => `共 ${lines.length} 条提交` })
      : null
  ])

  if (lines.length === 1) {
    return h(NPopover, { trigger: "hover", placement: "top-start" }, {
      trigger: () => preview,
      default: () => h("div", { class: "max-w-[420px] break-all text-xs text-slate-600" }, lines[0])
    })
  }

  return h(NPopover, { trigger: "hover", placement: "top-start" }, {
    trigger: () => preview,
    default: () =>
      h("div", { class: "max-h-[320px] max-w-[420px] space-y-1 overflow-auto text-xs text-slate-600" },
        lines.map(line => h("div", { class: "break-all" }, `· ${line}`)))
  })
}
