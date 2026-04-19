import type { Response } from "@/types/api/response"

import { request, WEB_URL } from "@/utils/network"
import http from "@/api"

export async function userEditPasswordAPI(params: object) {
	const { data } = await request<Response<any>>(`/user/editPassword`, {
		method: "POST",
		data: params
	})
	return data
}
// 修改头像
export async function userEditAvatarAPI(params: object) {
	const { data } = await request<Response<any>>(`/user/editAvatar`, {
		method: "POST",
		data: params,
		headers: {
			"Content-Type": "multipart/form-data"
		}
	})
	return data
}
// 修改用户信息
export async function userEditInfoAPI(params: object) {
	const { data } = await request<Response<any>>(`/user/editInfo`, {
		method: "POST",
		data: params
	})
	return data
}
export async function userInfoAPI() {
	const { data } = await request<Response<any>>(`/user/info`, {
		method: "POST"
	})
	return data
}

export async function userReset(params: { email: string; password: string }) {
	const { data } = await request<Response<any>>(`/user/reset`, {
		method: "POST",
		data: params
	})
	return data
}

export const userTokenAPI = (params: any) => {
	return http.post(`/user/token`, params)
}

// 子账号管理 API
export async function createUserAPI(params: any) {
	return http.post(`/user/create`, params)
}

export async function updateUserAPI(params: any) {
	return http.post(`/user/update`, params)
}

export async function deleteUserAPI(params: { id: number }) {
	return http.post(`/user/delete`, params)
}

export async function pageUserAPI(params: any) {
	return http.post(`/user/search`, params)
}
