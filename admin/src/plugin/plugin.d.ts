import type { DateTimeOptions, NumberOptions } from "vue-i18n"

declare module "vue" {
	interface ComponentCustomProperties {
		$dd: (value: number | Date | string, options?: DateTimeOptions | string, locale?: string) => string

		$nn: (value: number | string, options?: NumberOptions | string, locale?: string) => string

		$formatCurrent: (
			value: number | string | undefined | null,
			currency: string,
			options?: Intl.NumberFormatOptions
		) => string
	}
}
