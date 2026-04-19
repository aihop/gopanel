import http from "@/api"
import type { ReqPage, ResPage } from "../interface"
import type { Website } from "../interface/website"
import { TimeoutEnum } from "@/enums/http-enum"

 

export const SearchSSLPushRule = (req: ReqPage) => {
	return http.post<ResPage<Website.SSLPushRule>>(`/ssl/push-rule/search`, req)
}

export const CreateSSLPushRule = (req: Website.SSLPushRuleCreate) => {
	return http.post<any>(`/ssl/push-rule`, req)
}

export const UpdateSSLPushRule = (req: Website.SSLPushRuleUpdate) => {
	return http.post<any>(`/ssl/push-rule/update`, req)
}

export const DeleteSSLPushRule = (req: Website.DelReq) => {
	return http.post<any>(`/ssl/push-rule/del`, req)
}

export const SSLSearchAPI = (req: ReqPage) => {
	return http.post<ResPage<Website.SSLDTO>>(`/ssl/search`, req)
}
 

export const CreateSSL = (req: Website.SSLCreate) => {
	return http.post<Website.SSLCreate>(`/ssl`, req, TimeoutEnum.T_10M)
}

export const DeleteSSL = (req: Website.DelReq) => {
	return http.post<any>(`/ssl/del`, req)
}

 

export const GetSSL = (id: number) => {
	return http.get<Website.SSL>(`/ssl/${id}`)
}

export const ApplySSL = (req: Website.SSLApply) => {
	return http.post<Website.SSLApply>(`/ssl/apply`, req)
}

export const ObtainSSL = (req: Website.SSLObtain) => {
	return http.post<any>(`/ssl/obtain`, req)
}

export const RenewSSL = (req: Website.IDReq) => {
	return http.post<any>(`/ssl/renew`, req)
}

export const PushToCDNAPI = (req: Website.SSLPushCDN) => {
	return http.post<any>(`/ssl/push-cdn`, req)
}

export const UpdateSSL = (req: Website.SSLUpdate) => {
	return http.post<any>(`/ssl/update`, req)
}

 

export const UploadSSL = (req: Website.SSLUpload) => {
	return http.post<any>(`/ssl/upload`, req)
}
 

export const DownloadFile = (params: Website.SSLDownload) => {
	return http.download<BlobPart>(`/ssl/download`, params, {
		responseType: "blob",
		timeout: TimeoutEnum.T_40S
	})
}
 