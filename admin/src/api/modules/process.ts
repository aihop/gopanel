import http from "@/api"
import type { Process } from "../interface/process"

export const ProcessList = (req: Process.ListReq) => {
	return http.post<any>(`/process/list`, req)
}

export const StopProcess = (req: Process.StopReq) => {
	return http.post<any>(`/process/stop`, req)
}

/** 检查端口是否被占用 */
export async function ProcessCheckPort(req: Process.PortReq) {
	return http.post<any>(`/process/checkPort`, req)
}
 
 
 

 
 

 
 

 
 
 
 
 