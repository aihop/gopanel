import type { Response } from "@/types/api/response"
import { request } from "@/utils/network"

export async function hostsFirewallBaseAPI() {
	return await request<Response<any>>(`/host/firewall/base`, {
		method: "GET"
	})
}

export async function hostsFirewallSearchAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/search`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallOperateAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/operate`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallPortAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/port`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallForwardAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/forward`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallIPAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/ip`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallUpdatePortAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/update/port`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallUpdateAddrAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/update/addr`, {
		method: "POST",
		data
	})
}

export async function hostsFirewallBatchAPI(data: any) {
	return await request<Response<any>>(`/host/firewall/batch`, {
		method: "POST",
		data
	})
}
