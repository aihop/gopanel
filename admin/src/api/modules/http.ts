import http from "@/api"

export const httpDefaultCheckAPI = (params: { domain: string }) => {
	return http.post<any>("/http/default/check", params)
}

export const httpDefaultResolveAPI = (params: { domain: string }) => {
	return http.post<any>("/http/default/resolve", params)
}

export const httpDefaultStatusAPI = () => {
	return http.get<any>("/http/default/status")
}

export const httpDefaultConfigAPI = () => {
	return http.get<any>("/http/default/config")
}

export const httpDefaultUpdateAPI = (params: { content: string }) => {
	return http.post<any>("/http/default/update", params)
}

export const httpDefaultListAPI = () => {
	return http.get<any>("/http/default/list")
}

export const httpDefaultDeleteAPI = (data: { domain: string }) => {
	return http.post<any>("/http/default/delete", data)
}

export const httpDefaultGetAPI = (data: { domain: string }) => {
	return http.post<any>("/http/default/get", data)
}

export const httpDefaultReloadAPI = () => {
	return http.post("/http/default/reload")
}

export const httpDefaultStopAPI = () => {
	return http.post("/http/default/stop")
}
