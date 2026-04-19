import http from "@/api"
import type { Website } from "../interface/website"

 
export const ListWebsitesAPI = () => {
	return http.post<Website.WebsiteDTO[]>(`/website/list`)
}

export const CreateWebsiteAPI = (req: Website.WebSiteCreateReq) => {
	return http.post<any>(`/website/create`, req)
}

 
 
export const UpdateWebsiteAPI = (req: Website.WebSiteUpdateReq) => {
	return http.post<any>(`/website/update`, req)
}

export const DeleteWebsiteAPI = (req: Website.WebSiteDel) => {
	return http.post<any>(`/website/delete`, req)
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

 
 

 
 
 
