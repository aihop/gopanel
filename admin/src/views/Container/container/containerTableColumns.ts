import { h } from "vue"
import { NButton, NDropdown, NSpace, NTag } from "naive-ui"
import { t } from "@/i18n"
import type { Container } from "@/api/interface/container"
import { buildRuntimeSummaryText, getRunUserLabel, getRuntimeKindLabel, getRuntimeModeLabel } from "@/utils/runtime"

export type ColumnSetting = {
  key: string
  title: string
  visible: boolean
  fixed?: boolean
  original?: any
}

interface ContainerListColumnOptions {
  stateOptions: Array<{ label: string; value: string }>
  onInspect: (row: Container.ContainerInfo) => void
  onTerminal: (row: Container.ContainerInfo) => void
  onLog: (row: Container.ContainerInfo) => void
  getRowActions: (row: Container.ContainerInfo) => Array<{ label: string; disabled?: boolean; click: () => void }>
}

export const getContainerSourceLabel = (row: Container.ContainerInfo) => {
  switch (row.sourceType) {
    case "app":
      return t("container.typeApp")
    case "pipeline":
      return t("container.typePipeline")
    case "compose":
      return t("container.typeCompose")
    case "website":
      return t("container.typeWebsite")
    default:
      return t("container.typeManual")
  }
}

export const getContainerSourceTagType = (row: Container.ContainerInfo) => {
  switch (row.sourceType) {
    case "app":
      return "success"
    case "pipeline":
      return "warning"
    case "compose":
      return "info"
    case "website":
      return "primary"
    default:
      return "default"
  }
}

export const createContainerColumns = (options: ContainerListColumnOptions) => [
  {
    key: "__selection",
    type: "selection",
    width: 50,
    options: ["all", "none"]
  },
  {
    title: t("commons.table.name"),
    key: "name",
    width: 200,
    ellipsis: false,
    render(row: Container.ContainerInfo) {
      return h(
        NButton,
        {
          text: true,
          type: "primary",
          style: {
            padding: 0,
            textAlign: "left",
            height: "auto",
            whiteSpace: "normal",
            wordBreak: "break-all"
          },
          onClick: () => options.onInspect(row)
        },
        { default: () => row.name }
      )
    }
  },
  {
    title: t("container.image"),
    key: "imageName",
    width: 100
  },
  {
    title: t("commons.table.status"),
    key: "state",
    width: 100,
    render(row: Container.ContainerInfo) {
      const opt = options.stateOptions.find((item: any) => item.value === row.state)
      const label = opt ? opt.label : (row.state ?? "--")
      return h(
        NTag,
        {
          size: "small",
          type: row.state === "running" ? "success" : row.state === "dead" ? "error" : "default",
          bordered: false
        },
        { default: () => label }
      )
    }
  },
  {
    title: t("container.ip"),
    key: "network",
    width: 140,
    render(row: Container.ContainerInfo) {
      const ips = Array.isArray(row.network) ? row.network.filter(Boolean) : []
      if (!ips.length) {
        return "--"
      }
      return h(
        "div",
        { class: "text-xs leading-5" },
        ips.map((ip: string) => h("div", { key: ip }, ip))
      )
    }
  },
  {
    title: t("container.runtimeType"),
    key: "source",
    width: 210,
    render(row: Container.ContainerInfo) {
      const tags = [
        h(
          NTag,
          {
            size: "small",
            bordered: false,
            type: row.runtimeKind === "docker" ? "success" : row.runtimeKind === "podman" ? "warning" : "default"
          },
          { default: () => getRuntimeKindLabel(row, { kindFallback: "-" }) }
        ),
        h(
          NTag,
          {
            size: "small",
            bordered: false,
            type: row.runtimeMode === "rootless" ? "warning" : "default"
          },
          {
            default: () =>
              getRuntimeModeLabel(row, {
                rootlessLabel: t("container.rootless"),
                rootfulLabel: t("container.rootful"),
                defaultModeLabel: t("container.defaultMode")
              })
          }
        ),
        h(
          NTag,
          {
            size: "small",
            bordered: false,
            type: getContainerSourceTagType(row) as any
          },
          { default: () => getContainerSourceLabel(row) }
        )
      ]
      return h("div", { class: "space-y-1" }, [
        h(NSpace, { size: "small", wrap: true }, { default: () => tags }),
        h(
          "div",
          { class: "text-xs leading-5 text-gray-500" },
          `${t("container.runUser")}: ${getRunUserLabel(row, { userFallback: t("container.userDefault") })}`
        ),
        row.appInstallName
          ? h("div", { class: "text-xs leading-5 text-gray-500" }, row.appInstallName)
          : row.websites?.length
            ? h("div", { class: "text-xs leading-5 text-gray-500" }, row.websites[0])
            : null
      ])
    }
  },
  {
    title: t("commons.table.port"),
    key: "ports",
    width: 130,
    render(row: Container.ContainerInfo) {
      return h(
        NSpace,
        { vertical: true, size: "small" },
        {
          default: () =>
            (row.ports || []).map((port: string) =>
              h(
                NTag,
                {
                  bordered: false,
                  size: "small",
                  type: "info"
                },
                { default: () => port }
              )
            )
        }
      )
    }
  },
  {
    title: t("container.upTime"),
    key: "runTime",
    width: 150
  },
  {
    title: t("commons.table.operate"),
    key: "operate",
    width: 120,
    fixed: "right",
    render(row: Container.ContainerInfo) {
      return h(NSpace, null, {
        default: () => [
          h(
            NButton,
            {
              text: true,
              type: "primary",
              disabled: row.state !== "running",
              onClick: () => options.onTerminal(row)
            },
            { default: () => t("container.containerTerminal") }
          ),
          h(
            NButton,
            {
              text: true,
              type: "primary",
              onClick: () => options.onLog(row)
            },
            { default: () => t("commons.button.log") }
          ),
          h(
            NDropdown,
            {
              trigger: "hover",
              menuProps: () => ({
                style: "display: grid; grid-template-columns: repeat(2, minmax(120px, 1fr));"
              }),
              options: options.getRowActions(row).map(item => ({
                label: item.label,
                key: item.label,
                disabled: item.disabled ?? false
              })),
              onSelect: (key: string) => {
                const action = options.getRowActions(row).find(item => item.label === key)
                if (action) action.click()
              }
            },
            {
              default: () =>
                h(
                  NButton,
                  {
                    text: true,
                    type: "primary"
                  },
                  { default: () => t("tabs.more") }
                )
            }
          )
        ]
      })
    }
  }
]

export const createColumnSettings = (columns: any[]): ColumnSetting[] =>
  columns.map((column: any) => ({
    key: String(column.key || column.type),
    title:
      typeof column.title === "string"
        ? column.title
        : column.type === "selection"
          ? "选择"
          : String(column.key || ""),
    visible: true,
    fixed: column.type === "selection" || column.fixed === "right" || column.fixed === "left",
    original: column
  }))

export const buildContainerRuntimeSummary = (row: any) =>
  buildRuntimeSummaryText(row, {
    kindFallback: t("container.runtimeType"),
    rootlessLabel: t("container.rootless"),
    rootfulLabel: t("container.rootful"),
    defaultModeLabel: t("container.defaultMode"),
    userFallback: t("container.userDefault"),
    runUserPrefix: `${t("container.runUser")}: `
  })
