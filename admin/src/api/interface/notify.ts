export namespace Notify {
	export interface Config {
		id?: number
		enabled: boolean
		smtpHost: string
		smtpPort: number
		smtpUser: string
		smtpFrom: string
		/** none | starttls | ssl */
		smtpTlsMode: string
		/** 收件人，逗号/分号/换行分隔都可以 */
		receivers: string
		/** 连续命中多少次才真正触发，去抖用 */
		debounceTimes: number
		/** 持续未恢复时隔多久再提醒一次，0 表示只发一次 */
		silenceHours: number
		notifyResolved: boolean
		enableDisk: boolean
		enableContainer: boolean
		enableOffline: boolean
		enableCert: boolean
		/** 后端只回是否已设置密码，不回明文 */
		hasPassword?: boolean
	}

	export interface ConfigSave extends Config {
		/** 留空表示不修改已保存的密码 */
		password?: string
	}

	export interface Event {
		id: number
		sourceType: string
		nodeId: number
		sourceName: string
		type: string
		level: string
		status: string
		value: number
		detail: string
		hitCount: number
		firstSeenAt: string
		lastSeenAt: string
		lastNotifiedAt: string
		resolvedAt: string
		updatedAt: string
	}
}
