import http from "@/api"
import type { Website } from "../interface/website"
import type { ResPage } from "../interface"
 
export const websiteListAPI = () => {
	return http.post<ResPage<any>>(`/website/list`)
}

export const websiteCreateAPI = (req: Website.WebSiteCreateReq) => {
	return http.post<any>(`/website/create`, req)
}
 
export const websiteUpdateAPI = (req: Website.WebSiteUpdateReq) => {
	return http.put<any>(`/website/${req.id}`, req)
}

export const websiteDeleteAPI = (params: Website.WebSiteDel) => {
	return http.delete<any>(`/website/${params.id}`,params)
}


export const WebsiteDeployListAPI = (req: { websiteId: number }) => {
	return http.post<any[]>(`/website/deploy/list`, req)
}

export const WebsiteDeploySwitchAPI = (req: { deployId: number }) => {
	return http.post<any>(`/website/deploy/switch`, req)
}

export const WebsiteDeployDeleteAPI = (req: { deployId: number }) => {
	return http.post<any>(`/website/deploy/delete`, req)
}

export const WebsiteDeployTriggerAPI = (req: { websiteId: number; zipPath: string }) => {
	return http.post<any>(`/website/deploy/trigger`, req)
}

export const WebsiteDeploySnapshotAPI = (req: { websiteId: number }) => {
	return http.post<any>(`/website/deploy/snapshot`, req)
}

 
 

 
 
 
