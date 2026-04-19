import type { Login } from "@/api/interface/auth"
import http from "@/api"
import type { AuthData, User } from "@/types/api/auth"
import type { Response } from "@/types/api/response"
import { request } from "@/utils/network"

export const loginApi = (params: Login.ReqLoginForm) => {
	return http.post<Login.ResLogin>(`/auth/login`, params)
}

export const mfaLoginApi = (params: Login.MFALoginForm) => {
	return http.post<Login.ResLogin>(`/auth/mfalogin`, params)
}

export const getCaptcha = () => {
	return http.get<Login.ResCaptcha>(`/auth/captcha`)
}

export const logOutApi = () => {
	return http.post<any>(`/auth/logout`)
}

export const checkIsDemo = () => {
	return http.get<boolean>("/auth/demo")
}

export const getLanguage = () => {
	return http.get<string>(`/auth/language`)
}

export const checkIsIntl = () => {
	return http.get<boolean>("/auth/intl")
}

export async function authSignInAPI(data: { email: string; password: string }) {
	return await request<Response<AuthData>>(`/auth/signin`, {
		method: "POST",
		data
	})
}
