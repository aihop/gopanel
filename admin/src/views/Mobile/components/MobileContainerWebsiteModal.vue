<script setup lang="ts">
import {
	getMobileContainerPublishOptions,
	publishMobileContainerWebsite,
	type MobileContainer,
	type MobileContainerPublishOptions,
} from "@/api/modules/mobile"
import { useMessage } from "naive-ui"
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"

const emit = defineEmits(["success"])
const { t } = useI18n()
const message = useMessage()
const visible = ref(false)
const loading = ref(false)
const submitting = ref(false)
const target = ref<MobileContainer | null>(null)
const options = ref<MobileContainerPublishOptions>({ ports: [], websites: [] })
const websiteId = ref<number | null>(null)
const hostPort = ref<number | null>(null)
const scheme = ref<"http" | "https">("http")

const websiteOptions = computed(() => options.value.websites.map(website => ({
	label: website.primaryDomain || website.alias,
	value: website.id,
})))

const portOptions = computed(() => options.value.ports.map(port => ({
	label: `${port.hostPort} → ${port.containerPort}/tcp`,
	value: port.hostPort,
})))

const schemeOptions = computed(() => [
	{ label: "HTTP", value: "http" },
	{ label: "HTTPS", value: "https" },
])

async function acceptParams(container: MobileContainer) {
	target.value = container
	websiteId.value = null
	hostPort.value = null
	scheme.value = "http"
	options.value = { ports: [], websites: [] }
	visible.value = true
	loading.value = true
	try {
		options.value = await getMobileContainerPublishOptions(container)
		if (websiteOptions.value.length === 1) websiteId.value = websiteOptions.value[0].value
		if (portOptions.value.length === 1) hostPort.value = portOptions.value[0].value
	} catch (error) {
	} finally {
		loading.value = false
	}
}

async function submit() {
	if (!target.value || !websiteId.value || !hostPort.value) {
		message.error(t("container.bindWebsiteRequired"))
		return
	}
	submitting.value = true
	try {
		await publishMobileContainerWebsite({
			containerId: target.value.containerID,
			runtimeHost: target.value.runtimeHost || "",
			websiteId: websiteId.value,
			hostPort: hostPort.value,
			scheme: scheme.value,
		})
		message.success(t("container.bindWebsiteSuccess"))
		visible.value = false
		emit("success")
	} catch (error) {
	} finally {
		submitting.value = false
	}
}

defineExpose({ acceptParams })
</script>

<template>
	<n-modal
		v-model:show="visible"
		preset="card"
		style="width: min(560px, calc(100vw - 24px))"
		:title="t('container.bindWebsite')"
	>
		<n-spin :show="loading">
			<n-alert type="info" :show-icon="false" class="mb-4">
				{{ t("container.bindWebsiteHelper", { name: target?.name || "-" }) }}
			</n-alert>
			<n-empty
				v-if="!loading && (!websiteOptions.length || !portOptions.length)"
				:description="!portOptions.length ? t('container.bindWebsiteNoPort') : t('container.bindWebsiteNoProxy')"
			/>
			<n-form v-else label-placement="top">
				<n-form-item :label="t('container.bindWebsiteTarget')" required>
					<n-select v-model:value="websiteId" :options="websiteOptions" :placeholder="t('container.bindWebsiteTargetPlaceholder')" />
				</n-form-item>
				<n-form-item :label="t('container.bindWebsitePort')" required>
					<n-select v-model:value="hostPort" :options="portOptions" :placeholder="t('container.bindWebsitePortPlaceholder')" />
				</n-form-item>
				<n-form-item :label="t('container.bindWebsiteScheme')">
					<n-select v-model:value="scheme" :options="schemeOptions" />
				</n-form-item>
			</n-form>
		</n-spin>
		<template #footer>
			<div class="flex justify-end gap-3">
				<n-button @click="visible = false">{{ t("commons.button.cancel") }}</n-button>
				<n-button
					type="primary"
					:loading="submitting"
					:disabled="loading || !websiteOptions.length || !portOptions.length"
					@click="submit"
				>
					{{ t("container.bindWebsiteConfirm") }}
				</n-button>
			</div>
		</template>
	</n-modal>
</template>
