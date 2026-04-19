import * as locales from "./locales"

export type Locale = keyof typeof locales
export type MessageSchema = (typeof locales)[Locale]

const numberFormats = {
	zh: {
		currency: {
			style: "currency",
			currency: "CNY"
		},
		decimal: {
			style: "decimal",
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		},
		percent: {
			style: "percent",
			useGrouping: false
		}
	},
	en: {
		currency: {
			style: "currency",
			currency: "USD"
		},
		decimal: {
			style: "decimal",
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		},
		percent: {
			style: "percent",
			useGrouping: false
		}
	},
	USD: {
		currency: {
			style: "currency",
			currency: "USD"
		}
	},
	RMB: {
		currency: {
			style: "currency",
			currency: "CNY"
		}
	}
}

const datetimeFormats = {
	zh: {
		short: {
			year: "numeric",
			month: "2-digit",
			day: "2-digit"
		},
		long: {
			year: "numeric",
			month: "2-digit",
			day: "2-digit",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
			hour12: false
		}
	},
	en: {
		short: {
			year: "numeric",
			month: "short",
			day: "numeric"
		},
		long: {
			year: "numeric",
			month: "long",
			day: "numeric",
			hour: "numeric",
			minute: "numeric",
			second: "numeric",
			hour12: true
		}
	}
} as const
export function getI18NConf() {
	// @ts-expect-error "locales" don't match with  [key: string]: { default: MessageSchema } }
	const localesEntries = Object.entries<{ default: MessageSchema }>(locales)

	const messages = localesEntries.reduce(
		(acc: { [key: string]: MessageSchema }, cur: [string, { default: MessageSchema }]) => {
			acc[cur[0]] = cur[1].default
			return acc
		},
		{}
	) as { [key in Locale]: MessageSchema }

	return {
		legacy: false,
		locale: "zh",
		messages,
		numberFormats,
		datetimeFormats
	}
}
