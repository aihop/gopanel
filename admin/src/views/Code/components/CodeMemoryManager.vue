<script setup lang="ts">
import { computed, ref } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import {
	createCodeMemory,
	deleteCodeMemorySummary,
	deleteCodeMemory,
	extractCodeSessionMemory,
	getAIProviderAccounts,
	getCodeMemories,
	getCodeMemoryAuditEvents,
	getCodeMemorySetting,
	getCodeSessionMemoryStatus,
	saveCodeMemorySummary,
	saveCodeMemorySetting
} from "@/api/modules/code"
import type {
	CodeMemoryAuditEvent,
	CodeMemoryEntry,
	CodeMemoryExtractionStatus,
	CodeMemorySetting
} from "@/api/interface/codeMemories"
import type { AIProviderAccount } from "@/api/interface/aiAccounts"
import { codeMemoryMessages } from "../codeMemoryMessages"
import CodeMemoryDrawer from "./CodeMemoryDrawer.vue"

const props = defineProps<{ projectId: number; sessionId: number | null }>()
const { t } = useI18n({ messages: codeMemoryMessages })
const message = useMessage()
const dialog = useDialog()
const entries = ref<CodeMemoryEntry[]>([])
const summary = ref("")
const loading = ref(false)
const loadFailed = ref(false)
const saving = ref(false)
const removingId = ref(0)
const showDrawer = ref(false)
const setting = ref<CodeMemorySetting | null>(null)
const savingSetting = ref(false)
const accounts = ref<AIProviderAccount[]>([])
const settingLoading = ref(false)
const settingLoadFailed = ref(false)
const extractionStatus = ref<CodeMemoryExtractionStatus | null>(null)
const statusLoading = ref(false)
const statusLoadFailed = ref(false)
const extracting = ref(false)
const auditEvents = ref<CodeMemoryAuditEvent[]>([])
const auditLoading = ref(false)
const auditLoadFailed = ref(false)
const savingSummary = ref(false)
const extractionActive = computed(
	() => extractionStatus.value?.status === "queued" || extractionStatus.value?.status === "running"
)

// 列表只在打开和抽取完成时刷新；轮询只追踪正在进行的抽取状态。
async function load() {
	if (loading.value) return
	loading.value = true
	try {
		const response = await getCodeMemories(props.projectId)
		if (response.code !== 0) throw new Error(response.message)
		entries.value = response.data.entries || []
		summary.value = response.data.summary || ""
		loadFailed.value = false
	} catch {
		loadFailed.value = true
		message.error(t("code.memoryLoadFailed"))
	} finally {
		loading.value = false
	}
}

// 设置和列表一起拉：未配置时列表必然是空的，得同时知道「是没配」
// 还是「配了但还没攒出记忆」，否则提示会给错。
async function loadSetting() {
	settingLoading.value = true
	settingLoadFailed.value = false
	try {
		const [settingResponse, accountResponse] = await Promise.all([getCodeMemorySetting(), getAIProviderAccounts()])
		if (settingResponse.code !== 0) throw new Error(settingResponse.message)
		if (accountResponse.code !== 0) throw new Error(accountResponse.message)
		setting.value = settingResponse.data
		accounts.value = (accountResponse.data || []).filter(account => account.useForMemoryExtraction)
	} catch {
		settingLoadFailed.value = true
		message.error(t("code.memorySettingLoadFailed"))
	} finally {
		settingLoading.value = false
	}
}

async function loadStatus(silent = false) {
	if (!props.sessionId) {
		extractionStatus.value = null
		return
	}
	if (!silent) statusLoading.value = true
	try {
		const response = await getCodeSessionMemoryStatus(props.sessionId)
		if (response.code !== 0) throw new Error(response.message)
		const wasActive = extractionActive.value
		extractionStatus.value = response.data
		statusLoadFailed.value = false
		if (wasActive && response.data.status === "success") {
			await Promise.all([load(), loadAudit(true)])
		}
	} catch {
		statusLoadFailed.value = true
		if (!silent) message.error(t("code.memoryStatusLoadFailed"))
	} finally {
		if (!silent) statusLoading.value = false
	}
}

async function loadAudit(silent = false) {
	if (!silent) auditLoading.value = true
	try {
		const response = await getCodeMemoryAuditEvents()
		if (response.code !== 0) throw new Error(response.message)
		auditEvents.value = response.data || []
		auditLoadFailed.value = false
	} catch {
		auditLoadFailed.value = true
		if (!silent) message.error(t("code.memoryAuditLoadFailed"))
	} finally {
		if (!silent) auditLoading.value = false
	}
}

function openDrawer() {
	showDrawer.value = true
	void load()
	void loadSetting()
	void loadStatus()
	void loadAudit()
}

defineExpose({ open: openDrawer })

async function persistSetting(value: { enabled: boolean; accountId: number; growthThreshold: number }) {
	savingSetting.value = true
	try {
		const response = await saveCodeMemorySetting(value)
		if (response.code !== 0) throw new Error(response.message)
		setting.value = response.data
		message.success(t("code.memorySettingSaved"))
	} catch {
		message.error(t("code.memorySettingSaveFailed"))
	} finally {
		savingSetting.value = false
	}
}

async function extractNow() {
	if (!props.sessionId || extracting.value) return
	extracting.value = true
	try {
		const response = await extractCodeSessionMemory(props.sessionId)
		if (response.code !== 0) throw new Error(response.message)
		extractionStatus.value = response.data.status
		message.success(t(response.data.queued ? "code.memoryExtractQueued" : "code.memoryExtractAlreadyRunning"))
	} catch {
		message.error(t("code.memoryExtractFailed"))
	} finally {
		extracting.value = false
	}
}

async function persistSummary(content: string) {
	savingSummary.value = true
	try {
		const response = await saveCodeMemorySummary(content)
		if (response.code !== 0) throw new Error(response.message)
		summary.value = response.data.content
		message.success(t("code.memoryProfileSaved"))
		await loadAudit(true)
	} catch {
		message.error(t("code.memoryProfileSaveFailed"))
	} finally {
		savingSummary.value = false
	}
}

function clearSummary() {
	dialog.warning({
		title: t("code.memoryProfileClearTitle"),
		content: t("code.memoryProfileClearConfirm"),
		positiveText: t("code.clear"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			savingSummary.value = true
			try {
				const response = await deleteCodeMemorySummary()
				if (response.code !== 0) throw new Error(response.message)
				summary.value = ""
				message.success(t("code.memoryProfileCleared"))
				await loadAudit(true)
			} catch {
				message.error(t("code.memoryProfileClearFailed"))
			} finally {
				savingSummary.value = false
			}
		}
	})
}

useIntervalFn(() => {
	if (showDrawer.value && extractionActive.value) void loadStatus(true)
}, 2000)

async function add(content: string, allProjects: boolean) {
	saving.value = true
	try {
		const response = await createCodeMemory({ content, projectId: props.projectId, allProjects })
		if (response.code !== 0) throw new Error(response.message)
		message.success(t("code.memoryAdded"))
		await load()
	} catch {
		message.error(t("code.memoryAddFailed"))
	} finally {
		saving.value = false
	}
}

// 不做二次确认：记忆是软提示不是数据，删错了重新加一条就行，
// 为它加一层弹窗只会让「快速纠正」这件事变慢。
async function remove(id: number) {
	removingId.value = id
	try {
		const response = await deleteCodeMemory(id)
		if (response.code !== 0) throw new Error(response.message)
		entries.value = entries.value.filter(entry => entry.id !== id)
		message.success(t("code.memoryRemoved"))
	} catch {
		message.error(t("code.memoryRemoveFailed"))
	} finally {
		removingId.value = 0
	}
}
</script>

<template>
	<CodeMemoryDrawer
		v-model:show="showDrawer"
		:entries="entries"
		:summary="summary"
		:has-session="sessionId !== null"
		:setting="setting"
		:accounts="accounts"
		:extraction-status="extractionStatus"
		:audit-events="auditEvents"
		:loading="loading"
		:load-failed="loadFailed"
		:setting-loading="settingLoading"
		:setting-load-failed="settingLoadFailed"
		:status-loading="statusLoading"
		:status-load-failed="statusLoadFailed"
		:audit-loading="auditLoading"
		:audit-load-failed="auditLoadFailed"
		:saving="saving"
		:saving-setting="savingSetting"
		:saving-summary="savingSummary"
		:extracting="extracting"
		:removing-id="removingId"
		@refresh="load"
		@refresh-setting="loadSetting"
		@refresh-status="loadStatus"
		@refresh-audit="loadAudit"
		@add="add"
		@remove="remove"
		@extract="extractNow"
		@save-summary="persistSummary"
		@clear-summary="clearSummary"
		@save-setting="persistSetting"
	/>
</template>
