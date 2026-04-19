import http from "@/api"

export async function DaemonStatus() {
	return http.get<any>(`/daemon/status`)
}

/** 启动进程守护全部进程 */
export async function DaemonStart(req?: any) {
	return http.post<any>(`/daemon/start`, req)
}

/** 停止进程守护全部进程 */
export async function DaemonStop(req?: any) {
	return http.post<any>(`/daemon/stop`, req)
}

/** 重载进程守护配置 */
export async function DaemonReload(req?: any) {
	return http.post<any>(`/daemon/reload`, req)
}

/** 进程列表 */
export async function DaemonProcessList() {
	return http.get(`/daemon/process/list`)
}

/** 启动单个进程 */
export async function DaemonProcessStart(name: string) {
	return http.post<any>(`/daemon/process/start/${name}`)
}

/** 停止单个进程 */
export async function DaemonProcessStop(name: string) {
	return http.post<any>(`/daemon/process/stop/${name}`)
}


export async function DaemonProcessReload(name: string) {
	return http.post<any>(`/daemon/process/reload/${name}`)
}

export async function DaemonProcessGracefulAPI(name: string) {
	return http.post<any>(`/daemon/process/graceful/${name}`)
}

export async function DaemonConfigFileLoad() {
	return http.get<any>(`/daemon/config/file/load`)
}

export async function DaemonConfigFileUpdate(parames: { content: string }) {
	return http.post<any>(`/daemon/config/file/update`, parames)
}

export async function DaemonProcessLog(parames: { name: string; offset: number; length: number }) {
	return http.post<any>(`/daemon/process/log`, parames)
}

export async function DaemonProcessLogClearAPI(parames: { name: string }) {
	return http.post<any>(`/daemon/process/log/clear`, parames)
}


export async function DaemonConfigAdd(parames: any) {
	return http.post<any>(`/daemon/config/add`, parames)
}

export async function DaemonConfigUpdate(parames: any) {
	return http.post<any>(`/daemon/config/update`, parames)
}
 
export async function DaemonConfigDelete(parames: { names: string[] }) {
	return http.post<any>(`/daemon/config/delete`, parames)
}
