import http from "@/api"
import { TimeoutEnum } from "@/enums/http-enum"
import type { Backup } from "../interface/backup"

export const backupRecordListAPI = (params: any) => {
	return http.post(`/backup/record/list`, params)
}
 

export const backupRecordDeletesAPI = (params: { ids: number[] }) => {
	return http.post(`/backup/record/deletes`, params)
}
export const backupRecordDownloadAPI = (params: any) => {
	return http.post(`/backup/record/download`, params, TimeoutEnum.T_10M)
}

export const backupRecordSizeAPI = (params: any) => {
	return http.post(`/backup/record/size`, params)
}

export const backupListAPI = (params: any) => {
	return http.post(`/backup/list`, params)
}

export const backupHandleAPI = (params: any) => {
	return http.post(`/backup/handle`, params)
}

export const backupRecoverAPI = (params: any) => {
	return http.post(`/backup/recover`, params)
}

export const backupRecoverByUploadAPI = (params: Backup.Recover) => {
	return http.post(`/backup/recover/byUpload`, params, TimeoutEnum.T_1D)
}
