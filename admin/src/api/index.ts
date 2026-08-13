import axios, { type AxiosInstance, AxiosError } from "axios"
import type { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from "axios"
import type { ResultData } from "@/api/interface"
import { ResultEnum } from "@/enums/http-enum"
import { checkStatus } from "./helper/check-status"
import GlobalStore from "@/store/modules/global"
import { MsgError } from "@/utils/message"
import { useAuthStore } from "@/store/auth"
import type { Router } from "vue-router"
import { enc } from "crypto-js"

let router: Router | null = null

const buildLoginRoute = (entrance: string) => {
	const value = String(entrance || "").trim()
	if (value) {
		return { path: "/login", query: { entrance: value } }
	}
	return { path: "/login" }
}

// 模块顶部router还未被初始化 useRouter() 无效,初始化后使用参数传进来
export const setRouter = (r: Router) => {
	router = r
}

const config = {
	baseURL: import.meta.env.VITE_API_URL as string,
	timeout: ResultEnum.TIMEOUT as number,
	withCredentials: true
}

class RequestHttp {
	service: AxiosInstance
	public constructor(config: AxiosRequestConfig) {
		this.service = axios.create(config)
		this.service.interceptors.request.use(
			(config: AxiosRequestConfig) => {
				const globalStore = GlobalStore()

				let language = globalStore.language === "tw" ? "zh-Hant" : globalStore.language
				const authStore = useAuthStore()
				config.headers = {
					"Accept-Language": language,
					...config.headers,
					"x-auth": authStore.auth || ""
				}
				if (globalStore.entrance) {
					let entrance = enc.Base64.stringify(enc.Utf8.parse(globalStore.entrance))
					config.headers.EntranceCode = entrance
				}
				return {
					...config
				} as InternalAxiosRequestConfig<any>
			},
			(error: AxiosError) => {
				return Promise.reject(error)
			}
		)

		this.service.interceptors.response.use(
			(response: AxiosResponse) => {
				const globalStore = GlobalStore()

				globalStore.errStatus = ""
				const { data } = response

				if (data.code === ResultEnum.TOKEN_EXPIRED) {
					const authStore = useAuthStore()
					authStore.setLogout()
					router?.replace({ path: "/login" })
					MsgError("登录已经失效")
					return Promise.reject(new Error("登录已经失效"))
				}
				if (data.code === ResultEnum.FAIL) {
					const msg = response.data.msg || "系统错误"
					MsgError(msg)
					return Promise.reject(new Error(msg))
				}

				if (data.code == ResultEnum.OVERDUE || data.code == ResultEnum.FORBIDDEN) {
					globalStore.setLogStatus(false)
					router?.push(buildLoginRoute(globalStore.entrance))
					return Promise.reject(data)
				}
				if (data.code == ResultEnum.NOTFOUND) {
					globalStore.errStatus = "err-found"
					return
				}
				if (data.code == ResultEnum.ERRIP) {
					globalStore.errStatus = "err-ip"
					return
				}
				if (data.code == ResultEnum.ERRDOMAIN) {
					globalStore.errStatus = "err-domain"
					return
				}
				if (data.code == ResultEnum.UNSAFETY) {
					globalStore.errStatus = "err-unsafe"
					return
				}
				if (data.code == ResultEnum.EXPIRED) {
					router?.push({ name: "Expired" })
					return
				}
				if (data.code == ResultEnum.ERRXPACK) {
					globalStore.isProductPro = false
					window.location.reload()
					return Promise.reject(data)
				}
				if (data.code == ResultEnum.ERRGLOBALLOADDING) {
					globalStore.setGlobalLoading(true)
					globalStore.setLoadingText(data.msg)
					return
				} else {
					if (globalStore.isLoading) {
						globalStore.setGlobalLoading(false)
					}
				}
				if (data.code == ResultEnum.ERRAUTH) {
					return data
				}
				// if (data.code && data.code !== ResultEnum.SUCCESS) {
				// 	MsgError(data.msg)
				// 	return Promise.reject(data)
				// }
				return data
			},
			async (error: AxiosError) => {
				const globalStore = GlobalStore()

				globalStore.errStatus = ""
				const { response } = error
				if (error.message.indexOf("timeout") !== -1) MsgError("请求超时！请您稍后重试")
				if (response) {
					switch (response.status) {
						case 310:
							globalStore.errStatus = "err-ip"
							router?.push(buildLoginRoute(globalStore.entrance))
							return
						case 311:
							globalStore.errStatus = "err-domain"
							router?.push(buildLoginRoute(globalStore.entrance))
							return
						case 312:
							globalStore.errStatus = "err-entrance"
							router?.push(buildLoginRoute(globalStore.entrance))
							return
						case 313:
							router?.push({ name: "Expired" })
							return
						case 500:
						case 407:
							checkStatus(response.status, (response.data as any).msg || "")
							return Promise.reject(error)
						default:
							globalStore.isLogin = false
							globalStore.errStatus = "code-" + response.status
							router?.push(buildLoginRoute(globalStore.entrance))
							return Promise.reject(error)
					}
				}
				return Promise.reject(error)
			}
		)
	}

	get<T>(url: string, params?: object, _object = {}): Promise<ResultData<T>> {
		return this.service.get(url, { params, ..._object })
	}
	post<T>(url: string, params?: object, timeout?: number): Promise<ResultData<T>> {
		return this.service.post(url, params, {
			baseURL: import.meta.env.VITE_API_URL as string,
			timeout: timeout ? timeout : (ResultEnum.TIMEOUT as number),
			withCredentials: true
		})
	}
	put<T>(url: string, params?: object, _object = {}): Promise<ResultData<T>> {
		return this.service.put(url, params, _object)
	}
	delete<T>(url: string, params?: any, _object = {}): Promise<ResultData<T>> {
		return this.service.delete(url, { params, ..._object })
	}
	download<BlobPart>(url: string, params?: object, _object = {}): Promise<BlobPart> {
		return this.service.post(url, params, _object)
	}
	upload<T>(url: string, params: object = {}, config?: AxiosRequestConfig): Promise<T> {
		return this.service.post(url, params, config)
	}
}

// 此时拦截器可能尚未设置 router
export default new RequestHttp(config)
