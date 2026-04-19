import http from "@/api"
import type { ResPage, SearchWithPage } from "../interface"
import type { Command } from "../interface/command"
import type { Host } from "../interface/host"
import { deepCopy } from "@/utils/util"
import { TimeoutEnum } from "@/enums/http-enum"
import { enc } from "crypto-js"

export const searchHosts = (params: Host.SearchWithPage) => {
	return http.post<ResPage<Host.Host>>(`/host/search`, params)
}
export const getHostTree = (params: Host.ReqSearch) => {
	return http.post<Array<Host.HostTree>>(`/host/tree`, params)
}
export const addHost = (params: Host.HostOperate) => {
	let request = deepCopy(params) as Host.HostOperate
	if (request.password) {
		request.password = enc.Base64.stringify(enc.Utf8.parse(request.password))
	}
	if (request.privateKey) {
		request.privateKey = enc.Base64.stringify(enc.Utf8.parse(request.privateKey))
	}
	return http.post<Host.HostOperate>(`/hosts`, request)
}
export const testByInfo = (params: Host.HostConnTest) => {
	let request = deepCopy(params) as Host.HostOperate
	if (request.password) {
		request.password = enc.Base64.stringify(enc.Utf8.parse(request.password))
	}
	if (request.privateKey) {
		request.privateKey = enc.Base64.stringify(enc.Utf8.parse(request.privateKey))
	}
	return http.post<boolean>(`/host/test/byinfo`, request)
}
export const testByID = (id: number) => {
	return http.post<boolean>(`/host/test/byid/${id}`)
}
export const editHost = (params: Host.HostOperate) => {
	let request = deepCopy(params) as Host.HostOperate
	if (request.password) {
		request.password = enc.Base64.stringify(enc.Utf8.parse(request.password))
	}
	if (request.privateKey) {
		request.privateKey = enc.Base64.stringify(enc.Utf8.parse(request.privateKey))
	}
	return http.post(`/host/update`, request)
}
export const editHostGroup = (params: Host.GroupChange) => {
	return http.post(`/host/update/group`, params)
}
export const deleteHost = (params: { ids: number[] }) => {
	return http.post(`/host/del`, params)
}

// command
export const getCommandList = () => {
	return http.get<Array<Command.CommandInfo>>(`/host/command`, {})
}
export const getCommandPage = (params: SearchWithPage) => {
	return http.post<ResPage<Command.CommandInfo>>(`/host/command/search`, params)
}
export const getCommandTree = () => {
	return http.get<any>(`/host/command/tree`)
}
export const addCommand = (params: Command.CommandOperate) => {
	return http.post<Command.CommandOperate>(`/host/command`, params)
}
export const editCommand = (params: Command.CommandOperate) => {
	return http.post(`/host/command/update`, params)
}
export const deleteCommand = (params: { ids: number[] }) => {
	return http.post(`/host/command/del`, params)
}

export const getRedisCommandList = () => {
	return http.get<Array<Command.RedisCommand>>(`/host/command/redis`, {})
}
export const getRedisCommandPage = (params: SearchWithPage) => {
	return http.post<ResPage<Command.RedisCommand>>(`/host/command/redis/search`, params)
}
export const saveRedisCommand = (params: Command.RedisCommand) => {
	return http.post(`/host/command/redis`, params)
}
export const deleteRedisCommand = (params: { ids: number[] }) => {
	return http.post(`/host/command/redis/del`, params)
}

// firewall
export const loadFireBaseInfo = () => {
	return http.get<Host.FirewallBase>(`/host/firewall/base`)
}

export const operateFire = (operation: string) => {
	return http.post(`/host/firewall/operate`, { operation: operation }, TimeoutEnum.T_40S)
}
export const operatePortRule = (params: Host.RulePort) => {
	return http.post<Host.RulePort>(`/host/firewall/port`, params, TimeoutEnum.T_40S)
}
export const operateForwardRule = (params: { rules: Host.RuleForward[]; forceDelete: boolean }) => {
	return http.post<Host.RulePort>(`/host/firewall/forward`, params, TimeoutEnum.T_40S)
}
export const operateIPRule = (params: Host.RuleIP) => {
	return http.post<Host.RuleIP>(`/host/firewall/ip`, params, TimeoutEnum.T_40S)
}
export const updatePortRule = (params: Host.UpdatePortRule) => {
	return http.post(`/host/firewall/update/port`, params, TimeoutEnum.T_40S)
}
export const updateAddrRule = (params: Host.UpdateAddrRule) => {
	return http.post(`/host/firewall/update/addr`, params, TimeoutEnum.T_40S)
}
export const updateFirewallDescription = (params: Host.UpdateDescription) => {
	return http.post(`/host/firewall/update/description`, params)
}
export const batchOperateRule = (params: Host.BatchRule) => {
	return http.post(`/host/firewall/batch`, params, TimeoutEnum.T_60S)
}

// monitors
export const loadMonitor = (param: Host.MonitorSearch) => {
	return http.post<Array<Host.MonitorData>>(`/host/monitor/search`, param)
}
export const getNetworkOptions = () => {
	return http.get<Array<string>>(`/host/monitor/netoptions`)
}
export const getIOOptions = () => {
	return http.get<Array<string>>(`/host/monitor/iooptions`)
}
export const cleanMonitors = () => {
	return http.post(`/host/monitor/clean`, {})
}

export const clearMemoryCaches = (params?: { mode?: number }) => {
	return http.post(`/host/maintenance/clear`, { mode: 3, ...(params || {}) }, TimeoutEnum.T_40S)
}

export const relieveCPU = (params?: { level?: number }) => {
	return http.post(`/host/maintenance/relieve-cpu`, { level: 10, ...(params || {}) }, TimeoutEnum.T_40S)
}

// ssh
export const getSSHInfo = () => {
	return http.post<Host.SSHInfo>(`/host/ssh/search`)
}
export const getSSHConf = () => {
	return http.get<string>(`/host/ssh/conf`)
}
export const operateSSH = (operation: string) => {
	return http.post(`/host/ssh/operate`, { operation: operation }, TimeoutEnum.T_40S)
}
export const updateSSH = (params: Host.SSHUpdate) => {
	return http.post(`/host/ssh/update`, params, TimeoutEnum.T_40S)
}
export const updateSSHByfile = (file: string) => {
	return http.post(`/host/ssh/conffile/update`, { file: file }, TimeoutEnum.T_40S)
}
export const generateSecret = (params: Host.SSHGenerate) => {
	return http.post(`/host/ssh/generate`, params)
}
export const loadSecret = (mode: string) => {
	return http.post<string>(`/host/ssh/secret`, { encryptionMode: mode })
}
export const loadSSHLogs = (params: Host.searchSSHLog) => {
	return http.post<Host.sshLog>(`/host/ssh/log`, params)
}
