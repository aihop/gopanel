import type { App } from "vue"
import type { DateTimeOptions, NumberOptions } from "vue-i18n"

export default {
	install(app: App) {
		app.config.globalProperties.$dd = function (
			value: number | Date | string,
			options?: DateTimeOptions | string,
			locale?: string
		): string {
			let dateValue: Date | number

			if (typeof value === "string") {
				dateValue = new Date(value)
				if (Number.isNaN(dateValue.getTime())) {
					console.warn("[i18n] Invalid date string value:", value)
					return String(value)
				}
			} else {
				dateValue = value
			}

			try {
				if (options) {
					return app.config.globalProperties.$d(dateValue, options as any, locale as string)
				} else {
					return app.config.globalProperties.$d(dateValue, "long", locale as string)
				}
			} catch (e) {
				console.error("[i18n] Error formatting date:", e)
				return String(value)
			}
		}

		app.config.globalProperties.$nn = function (
			value: number | string,
			options?: NumberOptions | string,
			locale?: string
		): string {
			let numValue: number

			if (typeof value === "string") {
				numValue = Number(value)
				if (Number.isNaN(numValue)) {
					console.warn("[i18n] Invalid number string value:", value)
					return String(value)
				}
			} else {
				numValue = value
			}

			try {
				if (options) {
					return app.config.globalProperties.$n(numValue, options as any, locale as string)
				} else {
					return app.config.globalProperties.$n(numValue, "decimal", locale as string)
				}
			} catch (e) {
				console.error("[i18n] Error formatting number:", e)
				return String(value)
			}
		}
	}
}
