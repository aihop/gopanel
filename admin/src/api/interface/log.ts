import type { DateTimeFormats } from "@intlify/core-base"
import type { ReqPage } from "."

export namespace Log {
	export interface OperationLog {
		id: number
		userId: number
		source: string
		authMethod: string
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
	export interface SearchSSHLog extends ReqPage {
		ip: string
		status: string
		username: string
	}
	export interface LoginLog {
		ip: string
		address: string
		agent: string
		status: string
		message: string
		createdAt: string
	}
	export interface SSHLoginLog {
		createdAt: string
		status: string
		username: string
		sourceIp: string
		sourcePort: string
		authMethod: string
		message: string
		raw: string
		platform: string
		source: string
	}
	export interface SSHLoginLogResult {
		supported: boolean
		platform: string
		source: string
		partial: boolean
		warning: string
		items: SSHLoginLog[]
		total: number
		successfulCount: number
		failedCount: number
	}
	export interface CleanLog {
		logType: string
	}
}
