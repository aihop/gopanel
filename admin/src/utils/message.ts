import type { MessageApi, MessageOptions, MessageReactive } from "naive-ui"
import { createDiscreteApi } from "naive-ui"

const { message } = createDiscreteApi(["message"])
const requestErrors: PendingRequestError[] = []
let activeMessage = message
let showError = message.error.bind(message)

interface PendingRequestError {
	content: string
	duration: number
	timer: ReturnType<typeof setTimeout>
}

function removeRequestError(error: PendingRequestError) {
	const index = requestErrors.indexOf(error)
	if (index !== -1) requestErrors.splice(index, 1)
}

function takeRequestError() {
	const error = requestErrors.shift()
	if (error) clearTimeout(error.timer)
	return error
}

function requestErrorOptions(error: PendingRequestError, options?: MessageOptions): MessageOptions {
	return { duration: error.duration, showIcon: true, closable: true, ...options }
}

export function setGlobalMessageApi(messageApi: MessageApi) {
	requestErrors.splice(0).forEach(error => clearTimeout(error.timer))
	const originalError = messageApi.error.bind(messageApi)

	messageApi.error = (content, options) => {
		const requestError = takeRequestError()
		return requestError
			? originalError(requestError.content, requestErrorOptions(requestError, options))
			: originalError(content, options)
	}

	activeMessage = messageApi
	showError = originalError
	window.$message = messageApi
}

export function MsgRequestError(content: string, duration = 3000) {
	const requestError: PendingRequestError = {
		content,
		duration,
		timer: setTimeout(() => {
			removeRequestError(requestError)
			showError(content, requestErrorOptions(requestError))
		}, 0)
	}
	requestErrors.push(requestError)
}

export function MsgSuccess(content: string, duration = 3000) {
	return activeMessage.success(content, { duration, showIcon: true, closable: true })
}

export function MsgInfo(content: string, duration = 3000) {
	return activeMessage.info(content, { duration, showIcon: true, closable: true })
}

export function MsgWarning(content: string, duration = 3000) {
	return activeMessage.warning(content, { duration, showIcon: true, closable: true })
}

export function MsgError(content: string, duration = 3000): MessageReactive {
	return activeMessage.error(content, { duration, showIcon: true, closable: true })
}
