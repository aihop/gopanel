import type { Article, NotificationRecord } from "@/types/api/notification"
import type { Response } from "@/types/api/response"
import { request, WEB_URL } from "@/utils/network"

export async function readNotificationAPI(params: object) {
	return await request(`${WEB_URL}/notification/read`, {
		method: "POST",
		data: params
	})
}

export async function notificationListAPI(params: object) {
	const { data } = await request<Response<NotificationRecord[]>>(`${WEB_URL}/notification/list`, {
		method: "POST",
		data: params
	})
	return data
}

export async function allReadNotificationAPI(params: object) {
	return await request(`${WEB_URL}/notification/allRead`, {
		method: "POST",
		data: params
	})
}

export async function delNotificationAPI(params: object) {
	return await request(`${WEB_URL}/notification/del`, {
		method: "POST",
		data: params
	})
}

export async function allDelNotificationAPI() {
	return await request(`${WEB_URL}/notification/allDel`, {
		method: "POST",
		data: {}
	})
}
