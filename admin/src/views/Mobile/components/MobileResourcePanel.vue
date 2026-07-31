<script setup lang="ts">
import {
	getMobileApps,
	getMobileDatabases,
	getMobileSSLs,
	getMobileWebsites,
	type MobileApp,
	type MobileDatabase,
	type MobileSSL,
	type MobileWebsite
} from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileResourceMessages } from "@/i18n/locales/mobileResources"
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import MobileContainerPanel from "./MobileContainerPanel.vue"

type ResourceTab = "websites" | "databases" | "ssl" | "containers" | "apps"
type ResourceItem = MobileWebsite | MobileDatabase | MobileSSL | MobileApp

const { locale, t } = useI18n({ messages: mobileResourceMessages })
const message = useMessage()
const router = useRouter()
const activeTab = ref<ResourceTab>("containers")
const loading = ref(false)
const loadError = ref("")
const keyword = ref("")
const websites = ref<MobileWebsite[]>([])
const databases = ref<MobileDatabase[]>([])
const certificates = ref<MobileSSL[]>([])
const apps = ref<MobileApp[]>([])
const databaseWarningCount = ref(0)
const loadedTabs = ref<ResourceTab[]>(["containers"])

const tabs = computed(() => [
	{ value: "websites" as const, label: t("mobile.resourceWebsite"), icon: "mdi:web" },
	{ value: "databases" as const, label: t("mobile.resourceDatabase"), icon: "mdi:database-outline" },
	{ value: "ssl" as const, label: t("mobile.resourceSSL"), icon: "mdi:certificate-outline" },
	{ value: "containers" as const, label: t("mobile.resourceContainer"), icon: "mdi:cube-outline" },
	{ value: "apps" as const, label: t("mobile.resourceApp"), icon: "mdi:apps" }
])

const activeLabel = computed(() => tabs.value.find(tab => tab.value === activeTab.value)?.label || "")
const activeItems = computed<ResourceItem[]>(() => {
	switch (activeTab.value) {
		case "websites":
			return websites.value
		case "databases":
			return databases.value
		case "ssl":
			return certificates.value
		case "apps":
			return apps.value
		default:
			return []
	}
})

const filteredItems = computed(() => {
	const search = keyword.value.trim().toLowerCase()
	if (!search) return activeItems.value
	return activeItems.value.filter(item =>
		Object.values(item).some(value => String(value).toLowerCase().includes(search))
	)
})

function isWebsite(item: ResourceItem): item is MobileWebsite {
	return activeTab.value === "websites"
}

function isDatabase(item: ResourceItem): item is MobileDatabase {
	return activeTab.value === "databases"
}

function isSSL(item: ResourceItem): item is MobileSSL {
	return activeTab.value === "ssl"
}

function isApp(item: ResourceItem): item is MobileApp {
	return activeTab.value === "apps"
}

function displayValue(value?: string | number) {
	return value === undefined || value === null || value === "" || value === 0 ? "-" : String(value)
}

function statusType(status: string) {
	const normalized = status.toLowerCase()
	if (["running", "enabled", "enable", "valid", "success", "issued"].includes(normalized)) return "success"
	if (["failed", "error", "invalid", "dead"].includes(normalized)) return "error"
	if (["pending", "installing", "restarting"].includes(normalized)) return "warning"
	return "default"
}

function formatDate(value: string) {
	if (!value || value.startsWith("0001-")) return "-"
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? "-" : date.toLocaleDateString(locale.value)
}

function appPorts(app: MobileApp) {
	return [app.httpPort, app.httpsPort].filter(Boolean).join(" / ") || "-"
}

async function loadActiveResource(silent = false) {
	if (activeTab.value === "containers") return
	if (!silent) loading.value = true
	try {
		switch (activeTab.value) {
			case "websites":
				websites.value = (await getMobileWebsites()).items
				break
			case "databases": {
				const result = await getMobileDatabases()
				databases.value = result.items
				databaseWarningCount.value = result.warningCount || 0
				break
			}
			case "ssl":
				certificates.value = (await getMobileSSLs()).items
				break
			case "apps":
				apps.value = (await getMobileApps()).items
				break
		}
		if (!loadedTabs.value.includes(activeTab.value)) loadedTabs.value.push(activeTab.value)
		loadError.value = ""
	} catch (error) {
		loadError.value =
			error instanceof Error ? error.message : t("mobile.resourceLoadFailed", { name: activeLabel.value })
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
		if (!silent) message.error(loadError.value)
	} finally {
		if (!silent) loading.value = false
	}
}

watch(activeTab, tab => {
	keyword.value = ""
	loadError.value = ""
	if (!loadedTabs.value.includes(tab)) void loadActiveResource()
})
</script>

<template>
	<div class="space-y-4">
		<div class="overflow-x-auto rounded-2xl border border-slate-200 bg-white p-1 shadow-sm">
			<div class="grid min-w-[460px] grid-cols-5 gap-1">
				<button
					v-for="tab in tabs"
					:key="tab.value"
					type="button"
					class="flex min-h-12 items-center justify-center gap-1.5 rounded-xl px-2 text-xs font-medium transition-colors"
					:class="activeTab === tab.value ? 'bg-blue-50 text-blue-600' : 'text-slate-500'"
					@click="activeTab = tab.value"
				>
					<Icon :name="tab.icon" :size="18" />
					<span>{{ tab.label }}</span>
				</button>
			</div>
		</div>

		<MobileContainerPanel v-if="activeTab === 'containers'" />

		<template v-else>
			<div class="flex gap-2">
				<n-input
					v-model:value="keyword"
					clearable
					:placeholder="t('mobile.resourceSearch', { name: activeLabel })"
				>
					<template #prefix><Icon name="mdi:magnify" /></template>
				</n-input>
				<n-button
					circle
					secondary
					:loading="loading"
					:title="t('mobile.resourceRefresh', { name: activeLabel })"
					:aria-label="t('mobile.resourceRefresh', { name: activeLabel })"
					@click="loadActiveResource()"
				>
					<template #icon><Icon name="mdi:refresh" /></template>
				</n-button>
			</div>

			<n-alert v-if="loadError" type="error" :title="t('mobile.resourceLoadFailed', { name: activeLabel })">
				<div class="flex items-center justify-between gap-3">
					<span class="min-w-0 break-words">{{ loadError }}</span>
					<n-button text type="primary" @click="loadActiveResource()">{{ t("mobile.retry") }}</n-button>
				</div>
			</n-alert>

			<n-alert v-if="activeTab === 'databases' && databaseWarningCount" type="warning">
				{{ t("mobile.databaseWarnings", { count: databaseWarningCount }) }}
			</n-alert>

			<n-spin :show="loading">
				<div class="mb-2 text-xs text-slate-400">
					{{ t("mobile.resourceTotal", { count: activeItems.length }) }}
				</div>
				<n-empty
					v-if="!filteredItems.length"
					class="rounded-2xl bg-white py-16"
					:description="
						activeItems.length
							? t('mobile.resourceNoMatch', { name: activeLabel })
							: t('mobile.resourceEmpty', { name: activeLabel })
					"
				/>
				<div v-else class="space-y-3">
					<article
						v-for="(item, index) in filteredItems"
						:key="'id' in item ? item.id : `${activeTab}-${index}`"
						class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
					>
						<template v-if="isWebsite(item)">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<div class="truncate font-semibold">{{ item.alias || item.primaryDomain }}</div>
									<div class="mt-1 truncate text-xs text-slate-500">{{ item.primaryDomain }}</div>
								</div>
								<n-tag size="small" round :bordered="false" :type="statusType(item.status)">
									{{ displayValue(item.status) }}
								</n-tag>
							</div>
							<div class="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-500">
								<div>{{ t("mobile.resourceType") }}：{{ displayValue(item.type) }}</div>
								<div>{{ t("mobile.resourceAppName") }}：{{ displayValue(item.appName) }}</div>
								<div class="col-span-2">
									{{ t("mobile.resourcePipeline") }}：{{
										item.pipelineId ? `#${item.pipelineId}` : "-"
									}}
								</div>
							</div>
						</template>

						<template v-else-if="isDatabase(item)">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0 truncate font-semibold">{{ item.name }}</div>
								<n-tag size="small" round :bordered="false">{{ displayValue(item.type) }}</n-tag>
							</div>
							<div class="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-500">
								<div>{{ t("mobile.resourceServer") }}：{{ displayValue(item.server) }}</div>
								<div>{{ t("mobile.resourceEncoding") }}：{{ displayValue(item.encoding) }}</div>
								<div v-if="item.comment" class="col-span-2 break-words">{{ item.comment }}</div>
							</div>
						</template>

						<template v-else-if="isSSL(item)">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0 truncate font-semibold">
									{{ item.primaryDomain || `#${item.id}` }}
								</div>
								<n-tag size="small" round :bordered="false" :type="statusType(item.status)">
									{{ displayValue(item.status) }}
								</n-tag>
							</div>
							<div class="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-500">
								<div>{{ t("mobile.resourceType") }}：{{ displayValue(item.type) }}</div>
								<div>{{ t("mobile.resourceProvider") }}：{{ displayValue(item.provider) }}</div>
								<div>
									{{ t("mobile.resourceAutoRenew") }}：{{
										item.autoRenew ? t("mobile.resourceEnabled") : t("mobile.resourceDisabled")
									}}
								</div>
								<div>{{ t("mobile.resourceExpires") }}：{{ formatDate(item.expireDate) }}</div>
							</div>
						</template>

						<template v-else-if="isApp(item)">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0 truncate font-semibold">{{ item.name }}</div>
								<n-tag size="small" round :bordered="false" :type="statusType(item.status)">
									{{ displayValue(item.status) }}
								</n-tag>
							</div>
							<div class="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-500">
								<div>{{ t("mobile.resourceVersion") }}：{{ displayValue(item.version) }}</div>
								<div>{{ t("mobile.resourcePort") }}：{{ appPorts(item) }}</div>
								<div class="col-span-2">
									{{ t("mobile.resourceRuntime") }}：{{
										displayValue(item.runtimeHost || item.runtimeKind)
									}}
								</div>
								<div v-if="item.description" class="col-span-2 break-words">{{ item.description }}</div>
							</div>
						</template>
					</article>
				</div>
			</n-spin>
		</template>
	</div>
</template>
