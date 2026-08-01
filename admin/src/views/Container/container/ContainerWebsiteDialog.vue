<script setup lang="ts">
import type { Container } from "@/api/interface/container"
import type { Website } from "@/api/interface/website"
import { bindContainerWebsite, loadContainerInfo } from "@/api/modules/container"
import {
	remoteContainerBindWebsiteAPI,
	remoteContainerInfoAPI,
	remoteWebsiteListAPI,
} from "@/api/modules/nodeProxy"
import { websiteListAPI } from "@/api/modules/website"
import { t } from "@/i18n"
import { useMessage } from "naive-ui"
import { computed, ref } from "vue"

type BindTarget = {
	containerId: string
	containerName: string
	runtimeHost?: string
	nodeId?: number
}

type PortOption = {
	label: string
	value: number
}

const emit = defineEmits(["success"])
const message = useMessage()
const visible = ref(false)
const loading = ref(false)
const submitting = ref(false)
const target = ref<BindTarget | null>(null)
const websites = ref<Website.WebsiteDTO[]>([])
const ports = ref<PortOption[]>([])
const websiteId = ref<number | null>(null)
const hostPort = ref<number | null>(null)
const scheme = ref<"http" | "https">("http")

const websiteOptions = computed(() => websites.value.map(item => ({
	label: item.primaryDomain || item.alias,
	value: item.id,
})))

const schemeOptions = computed(() => [
	{ label: "HTTP", value: "http" },
	{ label: "HTTPS", value: "https" },
])

function normalizePorts(items?: Array<Pick<Container.Port, "hostPort" | "containerPort" | "protocol">>): PortOption[] {
	const seen = new Set<number>()
	const result: PortOption[] = []
	for (const item of items || []) {
		if (String(item.protocol || "").toLowerCase() !== "tcp") continue
		const value = Number(item.hostPort)
		if (!Number.isInteger(value) || value < 1 || seen.has(value)) continue
		seen.add(value)
		result.push({
			label: `${item.hostPort} → ${item.containerPort}/tcp`,
			value,
		})
	}
	return result.sort((left, right) => left.value - right.value)
}

async function acceptParams(params: BindTarget) {
	target.value = params
	websiteId.value = null
	hostPort.value = null
	scheme.value = "http"
	websites.value = []
	ports.value = []
	visible.value = true
	loading.value = true
	try {
		const nodeId = Number(params.nodeId || 0)
		const [websiteRes, containerRes] = nodeId > 0
			? await Promise.all([
				remoteWebsiteListAPI(nodeId),
				remoteContainerInfoAPI(nodeId, params.containerId),
			])
			: await Promise.all([
				websiteListAPI(),
				loadContainerInfo(params.containerId, params.runtimeHost || ""),
			])
		websites.value = (websiteRes.data?.items || []).filter((item: Website.WebsiteDTO) => (
			item.type === "proxy" && !item.appInstallId
		))
		ports.value = normalizePorts(containerRes.data?.exposedPorts)
		if (websites.value.length === 1) websiteId.value = websites.value[0].id
		if (ports.value.length === 1) hostPort.value = ports.value[0].value
	} catch (error: any) {
		websites.value = []
		ports.value = []
		message.error(error?.message || t("container.bindWebsiteLoadFailed"))
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
		const params = {
			containerId: target.value.containerId,
			websiteId: websiteId.value,
			hostPort: hostPort.value,
			scheme: scheme.value,
		}
		if (target.value.nodeId) {
			await remoteContainerBindWebsiteAPI(target.value.nodeId, params)
		} else {
			await bindContainerWebsite({ ...params, runtimeHost: target.value.runtimeHost || "" })
		}
		message.success(t("container.bindWebsiteSuccess"))
		visible.value = false
		emit("success")
	} catch (error: any) {
		message.error(error?.message || t("container.bindWebsiteFailed"))
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
    style="width: 560px"
    :title="t('container.bindWebsite')"
  >
    <n-spin :show="loading">
      <n-alert
        type="info"
        :show-icon="false"
        class="mb-4"
      >
        {{ t("container.bindWebsiteHelper", { name: target?.containerName || "-" }) }}
      </n-alert>
      <n-empty
        v-if="!loading && (!websiteOptions.length || !ports.length)"
        :description="!ports.length ? t('container.bindWebsiteNoPort') : t('container.bindWebsiteNoProxy')"
      />
      <n-form
        v-else
        label-placement="top"
      >
        <n-form-item
          :label="t('container.bindWebsiteTarget')"
          required
        >
          <n-select
            v-model:value="websiteId"
            :options="websiteOptions"
            :placeholder="t('container.bindWebsiteTargetPlaceholder')"
          />
        </n-form-item>
        <n-form-item
          :label="t('container.bindWebsitePort')"
          required
        >
          <n-select
            v-model:value="hostPort"
            :options="ports"
            :placeholder="t('container.bindWebsitePortPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('container.bindWebsiteScheme')">
          <n-select
            v-model:value="scheme"
            :options="schemeOptions"
          />
        </n-form-item>
      </n-form>
    </n-spin>
    <template #footer>
      <div class="flex justify-end gap-3">
        <n-button @click="visible = false">
          {{ t("commons.button.cancel") }}
        </n-button>
        <n-button
          type="primary"
          :loading="submitting"
          :disabled="loading || !websiteOptions.length || !ports.length"
          @click="submit"
        >
          {{ t("container.bindWebsiteConfirm") }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>
