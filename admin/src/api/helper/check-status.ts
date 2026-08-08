import { t } from "@/i18n"
import { MsgRequestError } from "@/utils/message"
import GlobalStore from "@/store/modules/global"

export const checkStatus = (status: number, msg: string): void => {
	const globalStore = GlobalStore()
	const entrance = String(globalStore.entrance || "").trim()
	const loginUrl = entrance ? `/login?entrance=${encodeURIComponent(entrance)}` : "/login"

	switch (status) {
		case 400:
			MsgRequestError(msg ? msg : t("commons.res.paramError"))
			break
		case 404:
			MsgRequestError(msg ? msg : t("commons.res.notFound"))
			break
		case 403:
			globalStore.setLogStatus(false)
			window.location.replace(loginUrl)
			MsgRequestError(msg ? msg : t("commons.res.forbidden"))
			break
		case 500:
			MsgRequestError(msg ? msg : t("commons.res.serverError"))
			break
		default:
			MsgRequestError(msg ? msg : t("commons.res.commonError"))
	}
}
