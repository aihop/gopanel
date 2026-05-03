import { h } from "vue"
import { NButton, NTag } from "naive-ui"

type RuleType = "port" | "ip" | "forward"

const renderStrategy = (row: any) => {
  const isAccept = row.strategy === "accept"
  return h(
    NTag,
    { type: isAccept ? "success" : "error", size: "small", bordered: false, round: true },
    { default: () => (isAccept ? "允许" : "拒绝") }
  )
}

const renderStatus = (row: any) => {
  const map: Record<string, { text: string; type: "success" | "warning" | "default" }> = {
    inUsed: { text: "已使用", type: "warning" },
    unused: { text: "未使用", type: "success" },
    unknown: { text: "未知", type: "default" }
  }
  const statusItem = map[row.usedStatus] || map.unknown
  return h(
    NTag,
    { type: statusItem.type, size: "small", bordered: false, round: true },
    { default: () => statusItem.text }
  )
}

export const createFirewallColumns = (
  ruleType: RuleType,
  onEdit: (row: any) => void,
  onDelete: (row: any) => void
) => {
  const actionColumn = {
    title: "操作",
    key: "actions",
    width: 160,
    render: (row: any) =>
      h("div", { class: "flex gap-2" }, [
        h(
          NButton,
          {
            size: "small",
            type: "primary",
            text: true,
            onClick: () => onEdit(row)
          },
          { default: () => "编辑" }
        ),
        h(
          NButton,
          {
            size: "small",
            type: "error",
            text: true,
            onClick: () => onDelete(row)
          },
          { default: () => "删除" }
        )
      ])
  }

  if (ruleType === "ip") {
    return [
      { type: "selection", width: 50 },
      { title: "IP / 网段", key: "address", minWidth: 220 },
      { title: "策略", key: "strategy", width: 120, render: renderStrategy },
      { title: "协议族", key: "family", width: 120 },
      { title: "描述", key: "description", minWidth: 220 },
      actionColumn
    ]
  }

  if (ruleType === "forward") {
    return [
      { type: "selection", width: 50 },
      { title: "协议", key: "protocol", width: 120 },
      { title: "入口端口", key: "port", width: 140 },
      { title: "目标 IP", key: "targetIP", minWidth: 180 },
      { title: "目标端口", key: "targetPort", width: 140 },
      actionColumn
    ]
  }

  return [
    { type: "selection", width: 50 },
    { title: "协议", key: "protocol", width: 120 },
    { title: "端口", key: "port", width: 140 },
    { title: "策略", key: "strategy", width: 120, render: renderStrategy },
    { title: "状态", key: "usedStatus", width: 120, render: renderStatus },
    { title: "协议族", key: "family", width: 120 },
    { title: "描述", key: "description", minWidth: 220 },
    actionColumn
  ]
}
