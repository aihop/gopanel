import type { AxiosRequestConfig, AxiosRequestHeaders } from "axios"
import { useAuthStore } from "@/store/auth"
import axios from "axios"

import { useRouter } from "vue-router"

import codeMessage from "./codeMessage"

const API_URL = import.meta.env.VITE_API_URL

const WEB_URL = `/api/web`

const axiosInstance = axios.create({
	baseURL: API_URL,
	timeout: 1_000 * 20
})

axiosInstance.interceptors.request.use(
	value => {
		const authStore = useAuthStore()
		const auth = authStore.getAuth() || ""
		if (auth) {
			return {
				...value,
				headers: {
					...value.headers,
					"x-auth": auth
				} as unknown as AxiosRequestHeaders
			}
		}
		return value
	},
	error => {
		return Promise.reject(error)
	}
)

axiosInstance.interceptors.response.use(
	response => {
		if (response?.status < 300) {
			const code = response?.data?.code
			const showToast = (response.config as ExtendedAxiosRequestConfig)._toast

			if (code === null || code === undefined) {
				return Promise.resolve(response)
			} else {
				if (code === 41) {
					const msg = response.data.msg || "系统错误"
					showToast && window.$message?.error(msg)
					return Promise.reject(new Error(msg))
				} else if (code === 50) {
					const authStore = useAuthStore()
					authStore.setLogout()
					showToast && window.$message?.error("登录已经失效")
					const router = useRouter()
					router.push("/login")
					return Promise.reject(new Error("登录已经失效"))
				} else {
					return Promise.resolve(response)
				}
			}
		} else {
			return Promise.reject(response)
		}
	},
	error => {
		let errMsg
		if (error?.message?.includes?.("timeout")) {
			errMsg = "请求超时"
		} else {
			errMsg = codeMessage?.[error?.response?.status as keyof typeof codeMessage] ?? "请求错误"
		}
		const showToast = error.config?._toast
		showToast && window.$message?.error(errMsg)
		return Promise.reject(new Error(errMsg))
	}
)

interface ExtendedAxiosRequestConfig extends AxiosRequestConfig {
	_toast?: boolean
}

function request<ResponseType = unknown>(
	url: string,
	options?: AxiosRequestConfig<unknown>,
	config: { _toast?: boolean } = { _toast: true }
): Promise<ResponseType> {
	return new Promise((resolve, reject) => {
		axiosInstance({
			url,
			...options,
			...config
		})
			.then(res => {
				resolve(res.data)
			})
			.catch(err => reject(err))
	})
}

export { axiosInstance, request, WEB_URL }
