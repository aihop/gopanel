import http from "@/api"
import type { Dashboard } from "../interface/dashboard"

export const loadOsInfo = () => {
	return http.get<Dashboard.OsInfo>(`/dashboard/base/os`)
}

export const loadBaseInfo = (ioOption: string, netOption: string) => {
	return http.get<Dashboard.BaseInfo>(`/dashboard/base/${ioOption}/${netOption}`)
}

export const loadCurrentInfo = (req: Dashboard.DashboardReq) => {
	return http.get<Dashboard.CurrentInfo>(`/dashboard/current`, req)
}

export const updateHostname = (hostname: string) => {
	return http.post<Dashboard.HostnameResult>(`/dashboard/hostname`, { hostname })
}

export const systemRestart = (operation: string) => {
	return http.post(`/dashboard/system/restart/${operation}`)
}
