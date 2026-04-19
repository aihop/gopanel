import http from "@/api"
import type { ResPage } from "../interface"
import type { App } from "../interface/apps"
import { TimeoutEnum } from "@/enums/http-enum"

export interface AppsInstalledSearchParams {
	page: number
	pageSize: number
	name?: string
	key?: string
}

export interface AppsSearchParams {
	page: number
	pageSize: number
	name?: string
	type?: string
	resource?: string
	recommend?: boolean
}

export const appsSearchAPI = (data: AppsSearchParams) => {
	return http.post("/apps/search", data)
}

export const appsInstalledList = () => {
	return http.get("/apps/installed/list")
}

export const appsInstalledSearch = (data: AppsInstalledSearchParams) => {
	return http.post("/apps/installed/search", data)
}

export const appsUninstall = (data: { containerName: string; deleteDir?: boolean }) => {
	return http.post("/apps/uninstall", data)
}

export const appsGetBaseDir = () => {
	return http.get<any>("/apps/baseDir")
}

 

export const GetAppListUpdate = () => {
	return http.get<App.AppUpdateRes>("/apps/checkupdate")
}

export function appsSyncAPI() {
	return http.post(`/apps/sync`)
}

export const GetApp = (key: string) => {
	return http.get<App.AppDTO>("/apps/" + key)
}

export const GetAppTags = () => {
	return http.get<App.Tag[]>("/apps/tags")
}

export const GetAppDetail = (id: number, version: string = "") => {
	const url = version ? `/apps/detail/${id}?version=${version}` : `/apps/detail/${id}`
	return http.get<App.AppDetail>(url)
}

export const GetAppDetailByID = (id: number) => {
	// The endpoint is /apps/detail/:id in Go code, not /apps/details/:id
	return http.get<App.AppDetail>(`/apps/detail/${id}`)
}

export const InstallApp = (install: App.AppInstall) => {
	return http.post<any>("/apps/install", install)
}

export const ChangePort = (params: App.ChangePort) => {
	return http.post<any>("/apps/installed/port/change", params)
}

export const SearchAppInstalled = (search: App.AppInstallSearch) => {
	return http.post<ResPage<App.AppInstallDto>>("/apps/installed/search", search)
}

export const ListAppInstalled = () => {
	return http.get<Array<App.AppInstalledInfo>>("/apps/installed/list")
}

export const GetAppPort = (type: string, name: string) => {
	return http.post<number>("/apps/installed/loadport", { type: type, name: name })
}

export const GetAppConnInfo = (type: string, name: string) => {
	return http.post<App.DatabaseConnInfo>("/apps/installed/conninfo", { type: type, name: name })
}

export const CheckAppInstalled = (key: string, name: string) => {
	return http.post<App.CheckInstalled>("/apps/installed/check", { key: key, name: name })
}

export const AppInstalledDeleteCheck = (appInstallId: number) => {
	return http.get<App.AppInstallResource[]>(`/apps/installed/delete/check/${appInstallId}`)
}

export const GetAppInstalled = (search: App.AppInstalledSearch) => {
	return http.post<ResPage<App.AppInstalled>>("/apps/installed/search", search)
}

export const InstalledOp = (op: App.AppInstalledOp) => {
	return http.post<any>("/apps/installed/op", op, TimeoutEnum.T_40S)
}

export const SyncInstalledApp = () => {
	return http.post<any>("/apps/installed/sync", {})
}

export const GetAppService = (key: string | undefined) => {
	return http.get<App.AppService[]>(`/apps/services/${key}`)
}

export const GetAppUpdateVersions = (req: App.AppUpdateVersionReq) => {
	return http.post<any>(`/apps/installed/update/versions`, req)
}

export const GetAppDefaultConfig = (key: string, name: string) => {
	return http.post<string>(`/apps/installed/conf`, { type: key, name: name })
}

export const GetAppInstallParams = (id: number) => {
	return http.get<App.AppConfig>(`/apps/installed/params/${id}`)
}

export const UpdateAppInstallParams = (req: any) => {
	return http.post<any>(`/apps/installed/params/update`, req)
}

export const IgnoreUpgrade = (req: any) => {
	return http.post<any>(`/apps/installed/ignore`, req)
}

export const GetIgnoredApp = () => {
	return http.get<App.IgnoredApp>(`/apps/ignored/detail`)
}
