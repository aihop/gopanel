import http from "@/api"
import type { ReqPage, ResPage } from "../interface"
import type { Website } from "../interface/website"

 
export const cloudAccountListAPI = (req: ReqPage) => {
	return http.post<ResPage<Website.CloudAccount>>(`/cloud/account/list`, req)
}

export const CreateCloudAccount = (req: Website.CloudAccountCreate) => {
	return http.post<any>(`/cloud/account`, req)
}

export const UpdateCloudAccount = (req: Website.CloudAccountUpdate) => {
	return http.post<any>(`/cloud/account/update`, req)
}

export const DeleteCloudAccount = (req: Website.DelReq) => {
	return http.post<any>(`/cloud/account/del`, req)
}

export const CloudCdnDomainsAPI = (id: number) => {
	return http.get<string[]>(`/cloud/cdn/${id}/domains`)
}