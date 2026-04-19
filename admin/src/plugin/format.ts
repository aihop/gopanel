import type { App } from "vue"
import { formatCurrency } from "@/utils/format"

export default {
	install(app: App) {
		app.config.globalProperties.$formatCurrent = formatCurrency
	}
}
