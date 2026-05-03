export const busyStatuses = new Set(["Installing", "Upgrading", "Rebuilding", "Syncing"])
export const errorStatuses = new Set(["UpErr", "DownloadErr", "SyncFailed"])

export function statusLabel(status: string) {
  switch (status) {
    case "Running":
      return "已启动"
    case "Stopped":
      return "已停止"
    case "Installing":
      return "安装中"
    case "Upgrading":
      return "升级中"
    case "Rebuilding":
      return "重建中"
    case "Syncing":
      return "同步中"
    case "SyncFailed":
      return "同步失败"
    case "DownloadErr":
      return "下载失败"
    case "UpErr":
      return "启动失败"
    default:
      return status || "-"
  }
}

export function statusType(status: string) {
  if (status === "Running") return "success"
  if (busyStatuses.has(status)) return "warning"
  if (errorStatuses.has(status)) return "error"
  return "default"
}

export function isBusy(item: any) {
  return busyStatuses.has(item?.status)
}

export function disableStart(item: any) {
  return isBusy(item) || item?.status === "Running"
}

export function disableStop(item: any) {
  return isBusy(item) || item?.status === "Stopped"
}

export function disableRestart(item: any) {
  return isBusy(item)
}

export function disableRebuild(item: any) {
  return isBusy(item)
}

export function disableUninstall(item: any) {
  return isBusy(item)
}

export function canCancelInstall(item: any) {
  return item?.status === "Installing"
}
