import { t } from "@/i18n"
import { MsgError } from "@/utils/message"
import GlobalStore from "@/store/modules/global"
import { useRouter } from "vue-router"

export const checkStatus = (status: number, msg: string): void => {
	const globalStore = GlobalStore()

	switch (status) {
		case 400:
			MsgError(msg ? msg : t("commons.res.paramError"))
			break
		case 404:
			MsgError(msg ? msg : t("commons.res.notFound"))
			break
		case 403:
			globalStore.setLogStatus(false)
			const router = useRouter()
			router.replace({ name: "entrance", params: { code: globalStore.entrance } })
			MsgError(msg ? msg : t("commons.res.forbidden"))
			break
		case 500:
			MsgError(msg ? msg : t("commons.res.serverError"))
			break
		default:
			MsgError(msg ? msg : t("commons.res.commonError"))
	}
}
