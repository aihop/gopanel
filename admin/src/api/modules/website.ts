import http from "@/api"
import type { Website } from "../interface/website"
import type { ResPage, Result } from "../interface"
import type { Pipeline } from "../interface/pipeline"
 
export const websiteListAPI = () => {
	return http.post<ResPage<Website.WebsiteDTO>>(`/website/list`)
}

export const websiteCreateAPI = (req: Website.WebSiteCreateReq) => {
	return http.post<void>(`/website/create`, req)
}
 
export const websiteUpdateAPI = (req: Website.WebSiteUpdateReq) => {
	return http.put<void>(`/website/${req.id}`, req)
}

export const websiteDeleteAPI = (params: Website.WebSiteDel) => {
	return http.delete<void>(`/website/${params.id}`,params)
}

export const WebsiteLogAPI = (req: Website.WebSiteLogReadReq) => {
	return http.post<Website.WebSiteLog>(`/website/log`, req)
}

export const WebsiteTodayIPStatsAPI = (req: Website.WebSiteTodayIPStatsReq) => {
	return http.post<Website.WebSiteTodayIPStats>(`/website/log/today-ip`, req)
}


export const AppDeployListAPI = (req: { websiteId: number }) => {
	return http.post<Website.AppDeployRecord[]>(`/website/app-deploy/list`, req)
}

export const WebsiteReleasePageAPI = (params: { websiteId: number; page: number; limit: number }) => {
	return http.get<ResPage<Pipeline.ResRelease>>(`/website/releases`, params)
}

export const AppDeploySwitchAPI = (req: { deployId: number }) => {
	return http.post<void>(`/website/app-deploy/switch`, req)
}

export const AppDeployDeleteAPI = (req: { deployId: number }) => {
	return http.post<void>(`/website/app-deploy/delete`, req)
}

export const AppDeployTriggerAPI = (req: Website.AppDeployTriggerReq) => {
	return http.post<void>(`/website/app-deploy/trigger`, req)
}

export const AppDeploySnapshotAPI = (req: { websiteId: number }) => {
	return http.post<void>(`/website/app-deploy/snapshot`, req)
}

 
 

 
 
 
