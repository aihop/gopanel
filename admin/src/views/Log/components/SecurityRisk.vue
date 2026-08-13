<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue"
import { useI18n } from "vue-i18n"
import { NButton, NTag, useMessage, type DataTableColumns } from "naive-ui"
import { formatTime } from "@/utils/date"
import type {
	SecurityEvent,
	SecurityEvidence,
	SecurityRecommendedAction,
	SecurityRiskLevel
} from "@/api/interface/securityMonitoring"
import {
	analyzeSecurityEvent,
	evaluateSecurityRisks,
	getSecurityEvents
} from "@/api/modules/securityMonitoring"
import SecurityRiskConfig from "./SecurityRiskConfig.vue"

const { t, te } = useI18n()
const message = useMessage()
const loading = ref(false)
const scanning = ref(false)
const analyzingId = ref(0)
const error = ref("")
const events = ref<SecurityEvent[]>([])
const total = ref(0)
const detail = ref<SecurityEvent>()
const detailVisible = ref(false)
const configVisible = ref(false)
const filters = reactive({ page: 1, limit: 20, status: "", level: "", sourceType: "" })
const pagination = reactive({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const levelTypes: Record<SecurityRiskLevel, "default" | "info" | "warning" | "error"> = {
	info: "default",
	low: "info",
	medium: "warning",
	high: "error",
	critical: "error"
}
const levelOptions = ["critical", "high", "medium", "low", "info"].map(value => ({
	label: t(`securityMonitoring.level.${value}`), value
}))
const statusOptions = ["firing", "pending", "resolved"].map(value => ({
	label: t(`securityMonitoring.status.${value}`), value
}))
const sourceOptions = ["website", "ssh", "panel"].map(value => ({
	label: t(`securityMonitoring.source.${value}`), value
}))
const levelCounts = computed(() => {
	const counts: Record<string, number> = { critical: 0, high: 0, medium: 0, low: 0 }
	for (const event of events.value) counts[event.level] = (counts[event.level] || 0) + 1
	return counts
})

function label(group: string, value: string) {
	const key = `securityMonitoring.${group}.${value}`
	return te(key) ? t(key) : value
}

function parseArray<T>(value: string): T[] {
	try {
		const parsed = JSON.parse(value || "[]")
		return Array.isArray(parsed) ? parsed : []
	} catch {
		return []
	}
}

const evidence = computed(() => parseArray<SecurityEvidence>(detail.value?.evidence || ""))
const aiEvidence = computed(() => parseArray<SecurityEvidence & { sample?: string }>(detail.value?.aiEvidence || ""))
const actions = computed(() => parseArray<SecurityRecommendedAction>(detail.value?.suggestedActions || ""))

async function loadEvents() {
	loading.value = true
	error.value = ""
	try {
		const response = await getSecurityEvents({ ...filters })
		events.value = response.data.items || []
		total.value = response.data.total || 0
		pagination.itemCount = total.value
	} catch {
		error.value = t("securityMonitoring.loadFailed")
		message.error(error.value)
	} finally {
		loading.value = false
	}
}

function search() {
	filters.page = 1
	pagination.page = 1
	void loadEvents()
}

function openDetail(event: SecurityEvent) {
	detail.value = event
	detailVisible.value = true
}

async function scanNow() {
	scanning.value = true
	try {
		await evaluateSecurityRisks()
		message.success(t("securityMonitoring.scanComplete"))
		await loadEvents()
	} catch {
		message.error(t("securityMonitoring.scanFailed"))
	} finally {
		scanning.value = false
	}
}

async function analyze(event: SecurityEvent) {
	analyzingId.value = event.id
	try {
		await analyzeSecurityEvent(event.id)
		message.success(t("securityMonitoring.analyzeComplete"))
		await loadEvents()
		const refreshed = events.value.find(item => item.id === event.id)
		if (refreshed && detailVisible.value) detail.value = refreshed
	} catch {
		message.error(t("securityMonitoring.analyzeFailed"))
	} finally {
		analyzingId.value = 0
	}
}

const columns: DataTableColumns<SecurityEvent> = [
	{
		title: t("securityMonitoring.riskLevel"), key: "level", width: 92,
		render: row => h(NTag, { size: "small", type: levelTypes[row.level], bordered: false }, { default: () => label("level", row.level) })
	},
	{ title: t("securityMonitoring.target"), key: "sourceName", width: 160, ellipsis: { tooltip: true } },
	{
		title: t("securityMonitoring.sourceLabel"), key: "sourceType", width: 90,
		render: row => label("source", row.sourceType)
	},
	{
		title: t("securityMonitoring.riskType"), key: "eventType", width: 150,
		render: row => label("eventType", row.eventType)
	},
	{ title: t("securityMonitoring.summary"), key: "summary", minWidth: 260, ellipsis: { tooltip: true } },
	{
		title: t("securityMonitoring.aiStatus"), key: "analysisStatus", width: 110,
		render: row => h(NTag, { size: "small", bordered: false }, { default: () => label("analysis", row.analysisStatus) })
	},
	{ title: t("securityMonitoring.lastSeen"), key: "lastSeenAt", width: 170, render: row => formatTime(row.lastSeenAt) },
	{
		title: t("securityMonitoring.actions"), key: "actions", width: 150, fixed: "right",
		render: row => [
			h(NButton, { size: "tiny", quaternary: true, onClick: () => openDetail(row) }, { default: () => t("securityMonitoring.details") }),
			h(NButton, { size: "tiny", quaternary: true, type: "primary", loading: analyzingId.value === row.id, disabled: row.status === "resolved", onClick: () => void analyze(row) }, { default: () => t("securityMonitoring.analyze") })
		]
	}
]

onMounted(() => void loadEvents())
</script>

<template>
	<div class="space-y-4">
		<div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
			<n-card v-for="level in ['critical', 'high', 'medium', 'low']" :key="level" size="small">
				<div class="text-xs text-slate-500">{{ label("level", level) }}</div>
				<div class="mt-1 text-2xl font-semibold">{{ levelCounts[level] || 0 }}</div>
			</n-card>
		</div>

		<n-alert type="warning" :show-icon="true">{{ t("securityMonitoring.approvalBoundary") }}</n-alert>

		<div class="flex flex-wrap items-center justify-between gap-3">
			<n-space :wrap="true">
				<n-select v-model:value="filters.status" clearable :options="statusOptions" :placeholder="t('securityMonitoring.statusLabel')" style="width: 140px" @update:value="search" />
				<n-select v-model:value="filters.level" clearable :options="levelOptions" :placeholder="t('securityMonitoring.riskLevel')" style="width: 140px" @update:value="search" />
				<n-select v-model:value="filters.sourceType" clearable :options="sourceOptions" :placeholder="t('securityMonitoring.sourceLabel')" style="width: 140px" @update:value="search" />
			</n-space>
			<n-space>
				<n-button :loading="loading" @click="loadEvents">{{ t("securityMonitoring.refresh") }}</n-button>
				<n-button :loading="scanning" @click="scanNow">{{ t("securityMonitoring.scanNow") }}</n-button>
				<n-button type="primary" @click="configVisible = true">{{ t("securityMonitoring.settings") }}</n-button>
			</n-space>
		</div>

		<n-alert v-if="error" type="error">{{ error }}</n-alert>
		<n-data-table
			:columns="columns"
			:data="events"
			:loading="loading"
			:pagination="pagination"
			:scroll-x="1250"
			:row-key="row => row.id"
			@update:page="page => { filters.page = page; pagination.page = page; loadEvents() }"
			@update:page-size="size => { filters.limit = size; pagination.pageSize = size; search() }"
		/>
		<n-empty v-if="!loading && !error && !events.length" :description="t('securityMonitoring.empty')" class="py-10" />

		<n-drawer v-model:show="detailVisible" :width="560">
			<n-drawer-content :title="t('securityMonitoring.details')" closable>
				<template v-if="detail">
					<n-descriptions :column="1" bordered label-placement="left">
						<n-descriptions-item :label="t('securityMonitoring.riskLevel')">{{ label("level", detail.level) }}</n-descriptions-item>
						<n-descriptions-item :label="t('securityMonitoring.statusLabel')">{{ label("status", detail.status) }}</n-descriptions-item>
						<n-descriptions-item :label="t('securityMonitoring.target')">{{ detail.sourceName }}</n-descriptions-item>
						<n-descriptions-item :label="t('securityMonitoring.firstSeen')">{{ formatTime(detail.firstSeenAt) }}</n-descriptions-item>
						<n-descriptions-item :label="t('securityMonitoring.lastSeen')">{{ formatTime(detail.lastSeenAt) }}</n-descriptions-item>
					</n-descriptions>
					<n-divider>{{ t("securityMonitoring.ruleEvidence") }}</n-divider>
					<n-empty v-if="!evidence.length" :description="t('securityMonitoring.noEvidence')" />
					<n-card v-for="(item, index) in evidence" :key="index" size="small" class="mb-2">
						<div class="font-medium">{{ item.description }} · {{ item.count }}</div>
						<div v-for="sample in item.samples || []" :key="sample" class="mt-2 break-all rounded bg-slate-50 p-2 font-mono text-xs">{{ sample }}</div>
					</n-card>
					<n-divider>{{ t("securityMonitoring.aiConclusion") }}</n-divider>
					<n-alert v-if="detail.analysisError" type="error">{{ detail.analysisError }}</n-alert>
					<n-empty v-else-if="!detail.aiConclusion" :description="t('securityMonitoring.noAiConclusion')" />
					<div v-else class="space-y-2">
						<p>{{ detail.aiConclusion }}</p>
						<n-tag type="info" :bordered="false">{{ t("securityMonitoring.confidence", { value: detail.confidence }) }}</n-tag>
						<div v-for="(item, index) in aiEvidence" :key="index" class="rounded bg-slate-50 p-2 text-sm">
							{{ item.description }} · {{ item.count }}<div v-if="item.sample" class="mt-1 break-all font-mono text-xs">{{ item.sample }}</div>
						</div>
					</div>
					<n-divider>{{ t("securityMonitoring.recommendedActions") }}</n-divider>
					<n-empty v-if="!actions.length" :description="t('securityMonitoring.noActions')" />
					<n-alert v-for="(action, index) in actions" :key="index" type="warning" class="mb-2">
						{{ action.action }} · {{ t("securityMonitoring.requiresApproval") }}
					</n-alert>
				</template>
			</n-drawer-content>
		</n-drawer>

		<SecurityRiskConfig v-if="configVisible" v-model:show="configVisible" />
	</div>
</template>
