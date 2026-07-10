import http from "@/api"

export const cronjobListAPI = (params: any) => {
	return http.post<any>(`/cronjob/list`, params)
}

export const cronjobCreateAPI = (params: any) => {
	return http.post<any>(`/cronjob/create`, params)
}

export const cronjobUpdateAPI = (params: any) => {
	return http.post<any>(`/cronjob/update`, params)
}

export const cronjobGetAPI = (params: { id: number }) => {
	return http.post<any>(`/cronjob/get`, params)
}

export const cronjobDeleteAPI = (params: { id: number }) => {
	return http.post<any>(`/cronjob/delete`, params)
}

export const cronjobSetStatusAPI = (params: { id: number; status: "Enable" | "Disable" }) => {
	return http.post<any>(`/cronjob/status`, params)
}

export const cronjobRunAPI = (params: { id: number }) => {
	return http.post<any>(`/cronjob/run`, params)
}

export const cronjobRecordListAPI = (params: { cronjobID: number; limit?: number }) => {
	return http.post<any>(`/cronjob/record/list`, params)
}

export const cronjobRecordDeleteAPI = (params: { cronjobID: number }) => {
	return http.post<any>(`/cronjob/record/delete`, params)
}
