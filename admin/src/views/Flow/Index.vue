<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useDialog, useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import Icon from "@/components/common/Icon.vue"
import type { Flow } from "@/api/interface/flow"
import { deleteFlow, getFlowPage } from "@/api/modules/flow"
import { flowMessages } from "./flowMessages"
import CreateFlowModal from "./CreateFlowModal.vue"
import StartFlowRunModal from "./StartFlowRunModal.vue"
import FlowRunPanel from "./FlowRunPanel.vue"

const { t } = useI18n({ messages: flowMessages })
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const loading = ref(false)
const loadError = ref(false)
const createVisible = ref(false)
const runVisible = ref(false)
const selectedFlow = ref<Flow.Item | null>(null)
const editingFlow = ref<Flow.Item | null>(null)
const deletingFlowId = ref<number | null>(null)
const runPanel = ref<InstanceType<typeof FlowRunPanel> | null>(null)
const total = ref(0)
const flows = ref<Flow.Item[]>([])

const configuredEnvironmentCount = computed(() => flows.value.reduce((sum, item) => sum + item.environments.length, 0))

function environmentLabel(name: Flow.Environment["name"]) {
	return t(name === "production" ? "flow.production" : "flow.preview")
}

function flowTime(value: string) {
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

async function loadFlows() {
	loading.value = true
	loadError.value = false
	try {
		const response = await getFlowPage({ page: 1, limit: 50 })
		flows.value = response.data.items || []
		total.value = response.data.total || 0
	} catch {
		loadError.value = true
		message.error(t("flow.loadFailed"))
	} finally {
		loading.value = false
	}
}

function created() {
	editingFlow.value = null
	void loadFlows()
}

function openCreate() {
	editingFlow.value = null
	createVisible.value = true
}

function editFlow(item: Flow.Item) {
	editingFlow.value = item
	createVisible.value = true
}

function confirmDelete(item: Flow.Item) {
	dialog.warning({
		title: t("flow.deleteTitle"),
		content: t("flow.deleteDescription", { name: item.name }),
		positiveText: t("flow.deleteConfirm"),
		negativeText: t("flow.cancel"),
		onPositiveClick: async () => {
			deletingFlowId.value = item.id
			try {
				await deleteFlow(item.id)
				message.success(t("flow.deleteSuccess"))
				await loadFlows()
			} catch {
				message.error(t("flow.deleteFailed"))
				throw new Error(t("flow.deleteFailed"))
			} finally {
				deletingFlowId.value = null
			}
		}
	})
}

function startRun(flow: Flow.Item) {
	selectedFlow.value = flow
	runVisible.value = true
}

function runCreated(run: Flow.Run) {
	runPanel.value?.refresh()
	void runPanel.value?.openRun(run)
}

onMounted(loadFlows)
</script>

<template>
	<div class="space-y-6 p-4 md:p-6">
		<section class="overflow-hidden rounded-3xl border border-slate-200 bg-base-100 shadow-sm">
			<div class="flex flex-col gap-6 p-6 lg:flex-row lg:items-end lg:justify-between lg:p-8">
				<div class="max-w-3xl">
					<div class="text-xs font-semibold uppercase tracking-[0.2em] text-blue-600">{{ t("flow.eyebrow") }}</div>
					<h1 class="mt-3 text-3xl font-semibold fg-base-100">{{ t("flow.title") }}</h1>
					<p class="mt-3 text-sm leading-7 text-slate-500">{{ t("flow.description") }}</p>
				</div>
				<div class="flex flex-wrap gap-2">
					<n-button secondary @click="router.push({ name: 'Code-Index' })">{{ t("flow.goCode") }}</n-button>
					<n-button type="primary" @click="openCreate">
						<template #icon><Icon name="mdi:plus" :size="18" /></template>{{ t("flow.createFlow") }}
					</n-button>
					<n-button quaternary circle :loading="loading" :aria-label="t('flow.refresh')" @click="loadFlows">
						<template #icon><Icon name="mdi:refresh" :size="18" /></template>
					</n-button>
				</div>
			</div>
			<div class="grid border-t border-slate-200 bg-slate-50/70 dark:bg-white/5 sm:grid-cols-3">
				<div class="px-6 py-4"><div class="text-xs text-slate-500">{{ t("flow.flowCount") }}</div><div class="mt-1 text-xl font-semibold fg-base-100">{{ total }}</div></div>
				<div class="border-slate-200 px-6 py-4 sm:border-l"><div class="text-xs text-slate-500">{{ t("flow.environmentCount") }}</div><div class="mt-1 text-xl font-semibold fg-base-100">{{ configuredEnvironmentCount }}</div></div>
				<div class="border-slate-200 px-6 py-4 sm:border-l"><div class="text-xs text-slate-500">{{ t("flow.runnerStatus") }}</div><div class="mt-1 text-sm font-semibold text-amber-600">{{ t("flow.runnerPending") }}</div></div>
			</div>
		</section>

		<n-alert type="info" :title="t('flow.boundaryTitle')">{{ t("flow.boundaryDescription") }}</n-alert>
		<n-alert v-if="loadError" type="error" :title="t('flow.loadFailed')">
			<n-button class="mt-3" size="small" @click="loadFlows">{{ t("flow.retry") }}</n-button>
		</n-alert>

		<n-spin :show="loading">
			<n-empty v-if="!loading && !flows.length" :description="t('flow.emptyDescription')" class="rounded-3xl border border-slate-200 bg-base-100 py-16 shadow-sm">
				<template #icon><Icon name="mdi:transit-connection-variant" :size="46" class="text-blue-500" /></template>
				<template #extra><n-button type="primary" @click="openCreate">{{ t("flow.createFirst") }}</n-button></template>
			</n-empty>
			<div v-else class="grid gap-4 xl:grid-cols-2">
				<article v-for="item in flows" :key="item.id" class="rounded-3xl border border-slate-200 bg-base-100 p-6 shadow-sm">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0"><div class="flex items-center gap-2"><h2 class="truncate text-lg font-semibold fg-base-100">{{ item.name }}</h2><n-tag size="small" type="success">{{ t("flow.flowId", { id: item.id }) }}</n-tag></div><div class="mt-2 text-xs text-slate-400">{{ flowTime(item.createdAt) }}</div></div>
						<div class="flex items-center gap-1">
							<n-tag size="small" :type="item.enabled ? 'success' : 'default'">{{ t(item.enabled ? "flow.enabled" : "flow.disabled") }}</n-tag>
							<n-button quaternary circle size="small" :aria-label="t('flow.edit')" @click="editFlow(item)"><template #icon><Icon name="mdi:pencil-outline" :size="16" /></template></n-button>
							<n-button quaternary circle size="small" type="error" :loading="deletingFlowId === item.id" :aria-label="t('flow.delete')" @click="confirmDelete(item)"><template #icon><Icon name="mdi:trash-can-outline" :size="16" /></template></n-button>
						</div>
					</div>

					<div class="mt-6 flex flex-wrap items-center gap-2 text-sm">
						<div class="flex items-center gap-2 rounded-xl bg-slate-50 px-3 py-2 dark:bg-white/5"><Icon name="mdi:code-braces" :size="17" class="text-blue-600" /><span>{{ item.projectName || `#${item.projectId}` }}</span></div>
						<Icon name="mdi:chevron-right" :size="18" class="text-slate-300" />
						<div class="flex items-center gap-2 rounded-xl bg-slate-50 px-3 py-2 dark:bg-white/5"><Icon name="mdi:source-merge" :size="17" class="text-blue-600" /><span>{{ item.pipelineName || `#${item.pipelineId}` }}</span></div>
						<Icon name="mdi:chevron-right" :size="18" class="text-slate-300" />
						<div class="flex items-center gap-2 rounded-xl bg-slate-50 px-3 py-2 dark:bg-white/5"><Icon name="mdi:package-variant-closed" :size="17" class="text-blue-600" /><span>{{ t("flow.releasePerRun") }}</span></div>
					</div>

					<div class="mt-5 grid gap-3 sm:grid-cols-2">
						<div v-for="environment in item.environments" :key="environment.id" class="rounded-2xl border border-slate-200 p-4">
							<div class="flex items-center justify-between gap-3"><div class="font-medium fg-base-100">{{ environmentLabel(environment.name) }}</div><n-tag size="small" :type="environment.approvalRequired ? 'warning' : 'info'">{{ t(environment.approvalRequired ? "flow.approvalRequired" : "flow.autoDeploy") }}</n-tag></div>
							<div class="mt-2 flex items-center gap-2 text-xs text-slate-500"><Icon name="mdi:web" :size="15" />{{ environment.websiteName || `#${environment.websiteId}` }}</div>
						</div>
					</div>
					<div class="mt-5 flex items-center justify-between gap-4 border-t border-slate-200 pt-4"><span class="text-xs text-slate-500">{{ t(item.autoStartAfterCodeDelivery ? "flow.autoStartEnabled" : "flow.manualStart") }}</span><n-button size="small" type="primary" :disabled="!item.enabled" @click="startRun(item)"><template #icon><Icon name="mdi:play" :size="16" /></template>{{ t("flow.startRun") }}</n-button></div>
				</article>
			</div>
		</n-spin>
		<FlowRunPanel ref="runPanel" />

		<CreateFlowModal v-model:show="createVisible" :flow="editingFlow" @success="created" />
		<StartFlowRunModal v-model:show="runVisible" :flow="selectedFlow" @success="runCreated" />
	</div>
</template>
