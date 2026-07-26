import http from "@/api"
import type { Disk } from "../interface/disk"

export const getDiskOverview = () => {
	return http.get<Disk.DiskUsage[]>(`/host/disk/overview`)
}

export const startDiskScan = (params: Disk.ScanRequest) => {
	return http.post<Disk.ScanTask>(`/host/disk/scan`, params)
}

export const getDiskScanResult = (taskId: string) => {
	return http.get<Disk.ScanTask>(`/host/disk/scan/result?taskId=${encodeURIComponent(taskId)}`)
}

export const cancelDiskScan = (taskId: string) => {
	return http.post(`/host/disk/scan/cancel`, { taskId })
}

/** truncate=true 为清空（保留 inode），日志类文件必须用这个 */
export const cleanDiskPaths = (taskId: string, paths: string[], truncate = false) => {
	return http.post<Disk.CleanResult[]>(`/host/disk/clean`, { taskId, paths, truncate })
}
