import type { DataTableColumns } from "naive-ui"
import { h } from "vue"
import { NButton, NSpace, NTag } from "naive-ui"
import { useI18n } from "vue-i18n"

const { t: $t } = useI18n()

export type ProcessStatusTagType = "success" | "default" | "primary" | "info" | "warning" | "error"

export interface ProcessMemoryInfo {
  rss: string
  swap: string
  vms: string
  hwm: string
  data: string
  stack: string
  locked: string
}

export interface ProcessOpenFile {
  path: string
  fd: number
}

export interface ProcessConnection {
  type: string
  status: string
  localaddr: { ip: string; port: number }
  remoteaddr: { ip: string; port: number }
  PID: number
  name: string
}

export interface ProcessData {
  PID: number
  name: string
  PPID: number
  username: string
  status: string
  startTime: string
  numThreads: number
  numConnections: number
  cpuPercent: string
  diskRead: string
  diskWrite: string
  cmdLine: string
  rss: string
  vms: string
  hwm: string
  data: string
  stack: string
  locked: string
  swap: string
  cpuValue: number
  rssValue: number
  envs: string[]
  openFiles: ProcessOpenFile[]
  connects: ProcessConnection[]
  memoryInfo?: ProcessMemoryInfo
  environmentVariables?: string
}

export const normalizeProcessRows = (items: ProcessData[]) =>
  items.map((item) => ({
    ...item,
    memoryInfo: {
      rss: item.rss,
      swap: item.swap,
      vms: item.vms,
      hwm: item.hwm,
      data: item.data,
      stack: item.stack,
      locked: item.locked
    },
    environmentVariables: item.envs?.join("\n") || ""
  }))

export const createProcessColumns = (
  getStatusType: (status: string | undefined) => ProcessStatusTagType,
  openDetailDrawer: (row: ProcessData) => void,
  handleStopProcess: (row: ProcessData) => void
): DataTableColumns<ProcessData> => [
  { title: "PID", key: "PID", sorter: true },
  { title: () => $t("process.pid"), key: "name", sorter: true },
  { title: () => $t("process.ppid"), key: "PPID", sorter: true },
  { title: () => $t("process.numThreads"), key: "numThreads" },
  { title: "用户", key: "username" },
  { title: "CPU", key: "cpuPercent", sorter: (row1: ProcessData, row2: ProcessData) => row1.cpuValue - row2.cpuValue },
  { title: () => $t("process.memory"), key: "rss", sorter: (row1: ProcessData, row2: ProcessData) => row1.rssValue - row2.rssValue },
  { title: () => $t("process.numConnections"), key: "numConnections" },
  {
    title: () => $t("process.status"),
    key: "status",
    render: (row: ProcessData) => h(NTag, { type: getStatusType(row.status) }, { default: () => row.status }),
    filter: true,
    filterOptions: [
      { label: $t("process.running"), value: "running" },
      { label: $t("process.sleep"), value: "sleep" },
      { label: $t("process.stop"), value: "stop" },
      { label: $t("process.idle"), value: "idle" },
      { label: $t("process.wait"), value: "wait" },
      { label: $t("process.lock"), value: "lock" },
      { label: $t("process.zombie"), value: "zombie" }
    ]
  },
  { title: () => $t("process.startTime"), key: "startTime" },
  {
    title: "操作",
    key: "actions",
    fixed: "right",
    width: 150,
    render: (row: ProcessData) =>
      h(NSpace, null, {
        default: () => [
          h(
            NButton,
            { strong: true, tertiary: true, type: "primary", ghost: true, size: "small", onClick: () => openDetailDrawer(row) },
            { default: () => $t("process.viewDetails") }
          ),
          h(
            NButton,
            { strong: true, tertiary: true, type: "error", size: "small", onClick: () => handleStopProcess(row) },
            { default: () => $t("process.stopProcess") }
          )
        ]
      })
  }
]

export const createNetworkColumns = (getStatusType: (status: string | undefined) => ProcessStatusTagType): DataTableColumns<ProcessConnection> => [
  { title: "类型", key: "type", sorter: true },
  { title: "PID", key: "PID", sorter: true },
  { title: () => $t("process.processName"), key: "name", sorter: true },
  {
    title: () => $t("process.laddr"),
    key: "localaddr",
    render: (row: ProcessConnection) => (row.localaddr && row.localaddr.port ? `${row.localaddr.ip}:${row.localaddr.port}` : row.localaddr?.ip || "")
  },
  {
    title: () => $t("process.raddr"),
    key: "remoteaddr",
    render: (row: ProcessConnection) => (row.remoteaddr && row.remoteaddr.port ? `${row.remoteaddr.ip}:${row.remoteaddr.port}` : row.remoteaddr?.ip || "")
  },
  {
    title: () => $t("process.status"),
    key: "status",
    sorter: true,
    render: (row: ProcessConnection) => h(NTag, { type: getStatusType(row.status) }, { default: () => row.status })
  }
]

export const openFilesColumns = [
  { title: () => $t("process.file"), key: "path" },
  { title: "fd", key: "fd", width: 60 }
]

export const createDrawerNetworkConnectionsColumns = (getStatusType: (status: string | undefined) => ProcessStatusTagType): DataTableColumns<ProcessConnection> => [
  {
    title: () => $t("process.laddr"),
    key: "localaddr",
    render: (row: ProcessConnection) => (row.localaddr.port ? `${row.localaddr.ip}:${row.localaddr.port}` : row.localaddr.ip)
  },
  {
    title: () => $t("process.raddr"),
    key: "remoteaddr",
    render: (row: ProcessConnection) => (row.remoteaddr.port ? `${row.remoteaddr.ip}:${row.remoteaddr.port}` : row.remoteaddr.ip)
  },
  {
    title: () => $t("process.status"),
    key: "status",
    width: 100,
    render: (row: ProcessConnection) => h(NTag, { type: getStatusType(row.status), size: "small" }, { default: () => row.status })
  }
]
