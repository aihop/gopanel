import { createDiscreteApi } from "naive-ui"

const { message } = createDiscreteApi(["message"])

export function MsgSuccess(content: string, duration = 3000) {
	return message.success(content, { duration, showIcon: true, closable: true })
}

export function MsgInfo(content: string, duration = 3000) {
	return message.info(content, { duration, showIcon: true, closable: true })
}

export function MsgWarning(content: string, duration = 3000) {
	return message.warning(content, { duration, showIcon: true, closable: true })
}

export function MsgError(content: string, duration = 3000) {
	return message.error(content, { duration, showIcon: true, closable: true })
}
