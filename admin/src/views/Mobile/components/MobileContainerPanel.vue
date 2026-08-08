<script setup lang="ts">
import { getMobileContainers, operateMobileContainer, type MobileContainer, type MobileContainerList } from "@/api/modules/mobile"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import MobileContainerWebsiteModal from "./MobileContainerWebsiteModal.vue"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"

type ContainerOperation = "start" | "stop" | "restart"

const { t } = useI18n({ messages: mobileMessages })
const dialog = useDialog()
const message = useMessage()
const router = useRouter()
const loading = ref(false)
const loadError = ref("")
const keyword = ref("")
const state = ref("all")
const operationKey = ref("")
const websiteModalRef = ref<InstanceType<typeof MobileContainerWebsiteModal>>()
const containerList = ref<MobileContainerList>({ items: [], total: 0, running: 0, stopped: 0 })
let refreshTimer: ReturnType<typeof setInterval> | null = null

const stateOptions = computed(() => [
	{ label: t("mobile.containerStateAll"), value: "all" },
	{ label: t("mobile.containerStateRunning"), value: "running" },
	{ label: t("mobile.containerStateStopped"), value: "stopped" }
])

const filteredContainers = computed(() => {
	const search = keyword.value.trim().toLowerCase()
	return containerList.value.items.filter(container => {
		const stateMatches = state.value === "all" || (state.value === "running" ? container.state === "running" : container.state !== "running")
		const keywordMatches = !search || container.name.toLowerCase().includes(search) || container.imageName.toLowerCase().includes(search)
		return stateMatches && keywordMatches
	})
})

function stateType(containerState: string) {
	if (containerState === "running") return "success"
	if (containerState === "paused" || containerState === "restarting") return "warning"
	if (containerState === "dead") return "error"
	return "default"
}

function stateLabel(containerState: string) {
	const knownStates = ["created", "running", "paused", "restarting", "removing", "exited", "dead"]
	const normalized = knownStates.includes(containerState) ? containerState : "unknown"
	return t(`mobile.containerState_${normalized}`)
}

function formatPercent(value: number) {
	return `${Math.max(0, value || 0).toFixed(1)}%`
}

function formatBytes(value: number) {
	if (!value) return "0 B"
	const units = ["B", "KB", "MB", "GB", "TB"]
	const unitIndex = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
	return `${(value / 1024 ** unitIndex).toFixed(unitIndex > 1 ? 1 : 0)} ${units[unitIndex]}`
}

function hasPublishedTCPPort(container: MobileContainer) {
	return (container.ports || []).some(port => /->.*\/tcp$/i.test(port))
}

async function loadContainers(silent = false) {
	if (!silent) loading.value = true
	try {
		containerList.value = await getMobileContainers()
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.containerLoadFailed")
		if (loadError.value.includes("手机授权已失效")) await router.replace("/mobile/auth")
	} finally {
		if (!silent) loading.value = false
	}
}

function operationLabel(operation: ContainerOperation) {
	return t(`mobile.containerOperation_${operation}`)
}

function confirmOperation(container: MobileContainer, operation: ContainerOperation) {
	dialog.warning({
		title: operationLabel(operation),
		content: t("mobile.containerOperationConfirm", { operation: operationLabel(operation), name: container.name }),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: () => runOperation(container, operation)
	})
}

async function runOperation(container: MobileContainer, operation: ContainerOperation) {
	operationKey.value = `${container.containerID}:${operation}`
	try {
		await operateMobileContainer(container, operation)
		message.success(t("mobile.containerOperationSuccess", { operation: operationLabel(operation) }))
		await loadContainers(true)
	} catch (error) {
	} finally {
		operationKey.value = ""
	}
}

onMounted(async () => {
	await loadContainers()
	refreshTimer = setInterval(() => void loadContainers(true), 5000)
})

onBeforeUnmount(() => {
	if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
	<div class="space-y-4">
		<div class="grid grid-cols-3 gap-3">
			<div class="rounded-2xl bg-white p-3 shadow-sm">
				<div class="text-xs text-slate-500">{{ t("mobile.containerTotal") }}</div>
				<div class="mt-1 text-2xl font-bold">{{ containerList.total }}</div>
			</div>
			<div class="rounded-2xl bg-white p-3 shadow-sm">
				<div class="text-xs text-slate-500">{{ t("mobile.containerRunning") }}</div>
				<div class="mt-1 text-2xl font-bold text-emerald-600">{{ containerList.running }}</div>
			</div>
			<div class="rounded-2xl bg-white p-3 shadow-sm">
				<div class="text-xs text-slate-500">{{ t("mobile.containerStopped") }}</div>
				<div class="mt-1 text-2xl font-bold text-slate-500">{{ containerList.stopped }}</div>
			</div>
		</div>

		<div class="flex gap-2">
			<n-input v-model:value="keyword" clearable :placeholder="t('mobile.containerSearch')">
				<template #prefix><Icon name="mdi:magnify" /></template>
			</n-input>
			<n-select v-model:value="state" style="width: 112px" :options="stateOptions" />
			<n-button circle secondary :loading="loading" :title="t('mobile.refreshContainers')" :aria-label="t('mobile.refreshContainers')" @click="loadContainers()">
				<template #icon><Icon name="mdi:refresh" /></template>
			</n-button>
		</div>

		<n-alert v-if="loadError" type="error" :title="t('mobile.containerLoadFailed')">
			<div class="flex items-center justify-between gap-3">
				<span class="min-w-0 break-words">{{ loadError }}</span>
				<n-button text type="primary" @click="loadContainers()">{{ t("mobile.retry") }}</n-button>
			</div>
		</n-alert>

		<n-spin :show="loading">
			<n-empty v-if="!filteredContainers.length" class="rounded-2xl bg-white py-16" :description="containerList.total ? t('mobile.noMatchingContainers') : t('mobile.noContainers')" />
			<div v-else class="space-y-3">
				<article v-for="container in filteredContainers" :key="container.containerID" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0 flex-1">
							<div class="truncate font-semibold">{{ container.name }}</div>
							<div class="mt-1 truncate text-xs text-slate-500">{{ container.imageName }}</div>
						</div>
						<n-tag size="small" round :bordered="false" :type="stateType(container.state)">
							{{ stateLabel(container.state) }}
						</n-tag>
					</div>
					<div class="mt-3 grid grid-cols-2 gap-2 rounded-xl bg-slate-50 p-3 text-xs">
						<div>
							<span class="text-slate-400">{{ t("mobile.cpu") }}</span>
							<span class="ml-2 font-medium">{{ formatPercent(container.cpuPercent) }}</span>
						</div>
						<div>
							<span class="text-slate-400">{{ t("mobile.memory") }}</span>
							<span class="ml-2 font-medium">{{ formatPercent(container.memoryPercent) }}</span>
						</div>
						<div class="col-span-2 truncate text-slate-500">{{ formatBytes(container.memoryUsage) }} / {{ formatBytes(container.memoryLimit) }}</div>
						<div v-if="container.runTime" class="col-span-2 truncate text-slate-500">
							{{ container.runTime }}
						</div>
					</div>
					<div class="mt-3 flex flex-wrap justify-end gap-2">
						<n-button
							v-if="container.state === 'running' && hasPublishedTCPPort(container)"
							size="small"
							type="primary"
							secondary
							@click="websiteModalRef?.acceptParams(container)"
						>
							{{ t("container.bindWebsite") }}
						</n-button>
						<n-button v-if="container.state !== 'running'" size="small" type="primary" secondary :loading="operationKey === `${container.containerID}:start`" :disabled="Boolean(operationKey)" @click="confirmOperation(container, 'start')">
							{{ t("mobile.containerOperation_start") }}
						</n-button>
						<n-button v-if="container.state === 'running'" size="small" secondary :loading="operationKey === `${container.containerID}:restart`" :disabled="Boolean(operationKey)" @click="confirmOperation(container, 'restart')">
							{{ t("mobile.containerOperation_restart") }}
						</n-button>
						<n-button v-if="container.state === 'running'" size="small" type="warning" secondary :loading="operationKey === `${container.containerID}:stop`" :disabled="Boolean(operationKey)" @click="confirmOperation(container, 'stop')">
							{{ t("mobile.containerOperation_stop") }}
						</n-button>
					</div>
				</article>
			</div>
		</n-spin>
		<MobileContainerWebsiteModal ref="websiteModalRef" @success="loadContainers(true)" />
	</div>
</template>
