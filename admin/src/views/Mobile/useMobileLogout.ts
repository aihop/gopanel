import { logoutMobileDevice } from "@/api/modules/mobile"
import type { Router } from "vue-router"
import { useDialog } from "naive-ui"

export function useMobileLogout(t: (key: string) => string, router: Router) {
	const dialog = useDialog()
	return () => {
		dialog.warning({
			title: t("mobile.logout"),
			content: t("mobile.logoutConfirm"),
			positiveText: t("mobile.logout"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: async () => {
				await logoutMobileDevice()
				await router.replace("/mobile/auth")
			}
		})
	}
}
