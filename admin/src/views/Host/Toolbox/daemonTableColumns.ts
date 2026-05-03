import { h } from "vue"
import { NButton, NInput, NTag, type DataTableColumns } from "naive-ui"
import { copyText } from "@/utils/util"

const renderTagWithCopy = (value?: string) => {
  const content = value || "-"
  return h("div", { class: "flex items-center gap-2" }, [
    h(
      NTag,
      {
        size: "small",
        type: value ? "info" : "default",
        bordered: false,
        class: "max-w-[380px] truncate"
      },
      { default: () => content }
    ),
    h(
      NButton,
      {
        text: true,
        size: "tiny",
        type: "primary",
        disabled: !value,
        onClick: (e: MouseEvent) => {
          e?.stopPropagation?.()
          if (!value) return
          copyText(value)
        }
      },
      "复制"
    )
  ])
}

interface DaemonColumnOptions {
  openPost: (row?: any) => void
  openLogs: (row: any) => void
  handleProcessStart: (name: string) => void
  handleProcessStop: (name: string) => void
  handleProcessReload: (name: string) => void
  handleProcessDelete: (name: string) => void
}

export const createDaemonColumns = (options: DaemonColumnOptions): DataTableColumns<any> => [
  { title: "名称", key: "name" },
  { title: "pid", key: "pid" },
  { title: "启动用户", key: "config.user" },
  {
    title: "运行目录",
    key: "config.directory",
    render: (row: any) => renderTagWithCopy(row?.config?.directory)
  },
  {
    title: "启动命令",
    key: "config.command",
    render: (row: any) => renderTagWithCopy(row?.config?.command)
  },
  {
    title: "进程数量",
    key: "config.numprocs",
    render: (row: any) => row?.config?.numprocs || 1
  },
  {
    title: "状态",
    key: "statename",
    render: (row: any) => h(NTag, { text: true }, row.statename)
  },
  {
    title: "操作",
    key: "actions",
    align: "center" as const,
    fixed: "right",
    render: (row: any) =>
      h("div", { class: "flex items-center justify-center gap-2" }, [
        h(
          NButton,
          {
            text: true,
            type: "primary",
            onClick: () => options.openPost(row)
          },
          "编辑"
        ),
        h(
          NButton,
          {
            text: true,
            type: "primary",
            onClick: () => options.openLogs(row)
          },
          "日志"
        ),
        h(
          NButton,
          {
            text: true,
            type: "primary",
            disabled: row.statename === "Running",
            onClick: () => options.handleProcessStart(row.name)
          },
          "启动"
        ),
        h(
          NButton,
          {
            text: true,
            type: "primary",
            disabled: row.statename === "Stopped" || row.statename === "Exited",
            onClick: () => options.handleProcessStop(row.name)
          },
          "停止"
        ),
        h(
          NButton,
          {
            text: true,
            type: "primary",
            disabled: row.statename === "Stopped" || row.statename === "Exited",
            onClick: () => options.handleProcessReload(row.name)
          },
          "重启"
        ),
        h(
          NButton,
          {
            text: true,
            type: "primary",
            onClick: () => options.handleProcessDelete(row.name)
          },
          "删除"
        )
      ])
  }
]

export const createStopConfirmContent = (stopConfirmInput: { value: string }) =>
  h("div", [
    h("div", { class: "mb-4 text-gray-500" }, "此操作将停止所有进程，输入“全部停止”以确认"),
    h(NInput, {
      value: stopConfirmInput.value,
      placeholder: "请输入：全部停止",
      "onUpdate:value": (v: string) => (stopConfirmInput.value = v)
    })
  ])

export const createDeleteConfirmContent = (deleteConfirmInput: { value: string }) =>
  h("div", [
    h(
      "div",
      {
        class: "mb-4"
      },
      "此操作将立即删除该进程，输入“立即删除”以确认"
    ),
    h(NInput, {
      value: deleteConfirmInput.value,
      placeholder: "请输入：立即删除",
      "onUpdate:value": (v: string) => (deleteConfirmInput.value = v)
    })
  ])
