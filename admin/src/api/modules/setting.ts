import http from "@/api"
import type { Setting } from "../interface/setting"

export const UploadFileData = (params: FormData) => {
	return http.upload("/licenses/upload", params)
}

export const getSettingInfo = () => {
	return http.post<Setting.SettingInfo>(`/setting/system/info`)
}

export const settingSystemBaseDirAPI = () => {
	return http.post<string>(`/setting/system/base-dir`)
}

export const updateSetting = (param: Setting.SettingUpdate) => {
	return http.post(`/setting/system/update`, param)
}

// 获取当前版本
export const settingSystemVersionAPI = () => {
	return http.get("/setting/system/version")
}

// 检查更新
export const settingSystemCheckAPI = (params: { lang: string, appBrand: string }) => {
	return http.get("/setting/system/check", params)
}

// 更新
export const settingSystemUpgradeAPI = (params: {
	containerName: string
	currentVersion: string
	targetVersion: string
}) => {
	return http.post("/setting/system/upgrade", params)
}

export const settingSystemConfig = () => {
	return http.post<any>(`/setting/system/config`)
}

export const settingSystemPort = (param: { serverPort: number }) => {
	return http.post<any>(`/setting/system/port`, param)
}

export const settingSystemEntrance = (param: { entrance: string }) => {
	return http.post<any>(`/setting/system/entrance`, param)
}

export const settingSystemClear = (param: { key: "log" | "tmp" | "cache" }) => {
	return http.post<any>(`/setting/system/clear`, param)
}

export const settingSystemApiTokenUpdate = (params: { apiInterfaceStatus: string, apiKey: string }) => {
	return http.post(`/setting/system/api-token`, params)
}

export const settingSystemRestart = (operation: "panel" | "server" = "panel") => {
	return http.post<any>(`/setting/system/restart/${operation}`)
}
