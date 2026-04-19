<template>
	<div>
		<n-modal v-model:show="dialogVisible" :mask-closable="false">
			<n-card
				:title="$t('app.checkTitle')"
				style="width: 30%; min-width: 360px"
				closable
				@close="dialogVisible = false"
			>
				<n-alert :show-icon="false" :closable="false" :title="$t('setting.systemIPWarning')" type="info">
					<n-button text type="primary" @click="goRouter('/setting')">
						{{ $t("firewall.quickJump") }}
					</n-button>
				</n-alert>
				<template #footer>
					<n-space justify="end">
						<n-button @click="dialogVisible = false">{{ $t("commons.button.cancel") }}</n-button>
					</n-space>
				</template>
			</n-card>
		</n-modal>
	</div>
</template>
<script lang="ts" setup>
import { NAlert, NButton, NCard, NModal, NSpace } from "naive-ui"
import { ref } from "vue"
import { getSettingInfo } from "@/api/modules/setting"
import { t } from "@/i18n"
import { MsgError, MsgWarning } from "@/utils/message"
import { useRouter } from "vue-router"
const router = useRouter()

const dialogVisible = ref()

interface DialogProps {
	port: any
	ip: string
	protocol: string
}

const acceptParams = async (params: DialogProps): Promise<void> => {
	if (Number(params.port) === 0) {
		MsgError(t("setting.errPort"))
		return
	}
	let protocol = params.protocol === "https" ? "https" : "http"
	const res = await getSettingInfo()
	if (!res.data.systemIP) {
		dialogVisible.value = true
		return
	}
	if (res.data.systemIP.indexOf(":") === -1) {
		if (params.ip && params.ip === "ipv6") {
			MsgWarning(t("setting.systemIPWarning1", ["IPv4"]))
			return
		}
		window.open(`${protocol}://${res.data.systemIP}:${params.port}`, "_blank")
	} else {
		if (params.ip && params.ip === "ipv4") {
			MsgWarning(t("setting.systemIPWarning1", ["IPv6"]))
			return
		}
		window.open(`${protocol}://[${res.data.systemIP}]:${params.port}`, "_blank")
	}
}

const goRouter = async (path: string) => {
	router.push({ path: path })
}

defineExpose({ acceptParams })
</script>
