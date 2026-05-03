import type { DataTableColumns } from "naive-ui"
import { h } from "vue"
import { NButton, NSpace, NTag } from "naive-ui"

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
  { title: "名称", key: "name", sorter: true },
  { title: "父进程ID", key: "PPID", sorter: true },
  { title: "线程", key: "numThreads" },
  { title: "用户", key: "username" },
  { title: "CPU", key: "cpuPercent", sorter: (row1: ProcessData, row2: ProcessData) => row1.cpuValue - row2.cpuValue },
  { title: "内存", key: "rss", sorter: (row1: ProcessData, row2: ProcessData) => row1.rssValue - row2.rssValue },
  { title: "连接", key: "numConnections" },
  {
    title: "状态",
    key: "status",
    render: (row: ProcessData) => h(NTag, { type: getStatusType(row.status) }, { default: () => row.status }),
    filter: true,
    filterOptions: [
      { label: "运行中", value: "running" },
      { label: "睡眠", value: "sleep" },
      { label: "停止", value: "stop" },
      { label: "空闲", value: "idle" },
      { label: "等待", value: "wait" },
      { label: "锁定", value: "lock" },
      { label: "僵尸", value: "zombie" }
    ]
  },
  { title: "启动时间", key: "startTime" },
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
            { default: () => "详情" }
          ),
          h(
            NButton,
            { strong: true, tertiary: true, type: "error", size: "small", onClick: () => handleStopProcess(row) },
            { default: () => "结束" }
          )
        ]
      })
  }
]

export const createNetworkColumns = (getStatusType: (status: string | undefined) => ProcessStatusTagType): DataTableColumns<ProcessConnection> => [
  { title: "类型", key: "type", sorter: true },
  { title: "PID", key: "PID", sorter: true },
  { title: "进程名称", key: "name", sorter: true },
  {
    title: "本地地址/端口",
    key: "localaddr",
    render: (row: ProcessConnection) => (row.localaddr && row.localaddr.port ? `${row.localaddr.ip}:${row.localaddr.port}` : row.localaddr?.ip || "")
  },
  {
    title: "远程地址/端口",
    key: "remoteaddr",
    render: (row: ProcessConnection) => (row.remoteaddr && row.remoteaddr.port ? `${row.remoteaddr.ip}:${row.remoteaddr.port}` : row.remoteaddr?.ip || "")
  },
  {
    title: "状态",
    key: "status",
    sorter: true,
    render: (row: ProcessConnection) => h(NTag, { type: getStatusType(row.status) }, { default: () => row.status })
  }
]

export const openFilesColumns = [
  { title: "文件", key: "path" },
  { title: "fd", key: "fd", width: 60 }
]

export const createDrawerNetworkConnectionsColumns = (getStatusType: (status: string | undefined) => ProcessStatusTagType): DataTableColumns<ProcessConnection> => [
  {
    title: "本地地址/端口",
    key: "localaddr",
    render: (row: ProcessConnection) => (row.localaddr.port ? `${row.localaddr.ip}:${row.localaddr.port}` : row.localaddr.ip)
  },
  {
    title: "远程地址/端口",
    key: "remoteaddr",
    render: (row: ProcessConnection) => (row.remoteaddr.port ? `${row.remoteaddr.ip}:${row.remoteaddr.port}` : row.remoteaddr.ip)
  },
  {
    title: "状态",
    key: "status",
    width: 100,
    render: (row: ProcessConnection) => h(NTag, { type: getStatusType(row.status), size: "small" }, { default: () => row.status })
  }
]
