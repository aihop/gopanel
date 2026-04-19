import http from "@/api"
import type { HostTool } from "../interface/host-tool"
import { TimeoutEnum } from "@/enums/http-enum"

export const GetDaemonStatus = () => {
	return http.post<HostTool.HostTool>(`/host/tool`, { type: "supervisord", operate: "status" })
}

export const OperateDaemon = (operate: string) => {
	return http.post<any>(`/host/tool/operate`, { type: "supervisord", operate: operate })
}

export const OperateDaemonConfig = (req: HostTool.DaemonConfig) => {
	return http.post<HostTool.DaemonConfigRes>(`/host/tool/config`, req)
}

export const GetDaemonLog = () => {
	return http.post<any>(`/host/tool/log`, { type: "supervisord" })
}

export const InitDaemon = (req: HostTool.DaemonInit) => {
	return http.post<any>(`/host/tool/init`, req)
}

export const CreateDaemonProcess = (req: HostTool.DaemonProcess) => {
	return http.post<any>(`/host/tool/daemon/process`, req)
}

export const OperateDaemonProcess = (req: HostTool.ProcessReq) => {
	return http.post<any>(`/host/tool/daemon/process`, req, TimeoutEnum.T_60S)
}

export const LoadProcessStatus = () => {
	return http.post<Array<HostTool.ProcessStatus>>(`/host/tool/daemon/process/load`, {}, TimeoutEnum.T_40S)
}

export const GetDaemonProcess = () => {
	return http.get<HostTool.DaemonProcess[]>(`/host/tool/daemon/process`)
}

export const OperateDaemonProcessFile = (req: HostTool.ProcessFileReq) => {
	return http.post<any>(`/host/tool/daemon/process/file`, req, TimeoutEnum.T_60S)
}
