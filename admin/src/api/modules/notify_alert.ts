import http from "@/api"
import type { ResPage } from "../interface"
import type { Notify } from "../interface/notify"

export const getNotifyConfig = () => {
	return http.get<Notify.Config>(`/setting/notify/config`)
}

export const saveNotifyConfig = (params: Notify.ConfigSave) => {
	return http.post<Notify.Config>(`/setting/notify/config`, params)
}

/** 发测试邮件。SMTP 配错的方式太多，没有即时反馈用户不知道配没配对 */
export const testNotifyMail = (params: Notify.ConfigSave) => {
	return http.post<string>(`/setting/notify/test`, params)
}

export const getNotifyEvents = (page = 1, limit = 20) => {
	return http.get<ResPage<Notify.Event>>(`/setting/notify/events?page=${page}&limit=${limit}`)
}

/** 立即跑一轮评估，配置完马上验证，不用等下一个采集周期 */
export const evaluateAlerts = () => {
	return http.post<string>(`/setting/notify/evaluate`)
}
