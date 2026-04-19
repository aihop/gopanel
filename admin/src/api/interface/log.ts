import type { DateTimeFormats } from "@intlify/core-base"
import type { ReqPage } from "."

export namespace Log {
	export interface OperationLog {
		id: number
		source: string
		ip: string
		path: string
		method: string
		userAgent: string

		status: string
		message: string
		latency: number

		detailZH: string
		detailEN: string
		createdAt: string
	}
	export interface SearchOpLog extends ReqPage {
		source: string
		status: string
		operation: string
	}
	export interface SearchLgLog extends ReqPage {
		ip: string
		status: string
	}
	export interface LoginLog {
		ip: string
		address: string
		agent: string
		status: string
		message: string
		createdAt: string
	}
	export interface CleanLog {
		logType: string
	}
}
