import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { ControlPlaneStatusAPI, type ControlPlaneStatus } from "@/api/modules/agent"
import { controlPlaneMessages } from "@/i18n/locales/controlPlane"

export const useControlPlaneStatus = () => {
	const { t } = useI18n({ messages: controlPlaneMessages })
	const message = useMessage()
	const controlPlaneStatus = ref<ControlPlaneStatus | null>(null)
	const controlPlaneLoading = ref(false)
	const controlPlaneError = ref("")

	const fetchControlPlaneStatus = async (notify = false) => {
		if (controlPlaneLoading.value) return
		controlPlaneLoading.value = true
		controlPlaneError.value = ""
		try {
			const response = await ControlPlaneStatusAPI()
			controlPlaneStatus.value = response?.data || null
			if (!controlPlaneStatus.value) controlPlaneError.value = t("controlPlane.empty")
		} catch (error: any) {
			controlPlaneError.value = error?.message || t("controlPlane.loadFailed")
		} finally {
			controlPlaneLoading.value = false
		}
	}

	return {
		controlPlaneStatus,
		controlPlaneLoading,
		controlPlaneError,
		fetchControlPlaneStatus
	}
}
