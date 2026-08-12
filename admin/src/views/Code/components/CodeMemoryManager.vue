<script setup lang="ts">
import { ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { createCodeMemory, deleteCodeMemory, getAIProviderAccounts, getCodeMemories, getCodeMemorySetting, saveCodeMemorySetting } from "@/api/modules/code"
import type { CodeMemoryEntry, CodeMemorySetting } from "@/api/interface/codeMemories"
import type { AIProviderAccount } from "@/api/interface/aiAccounts"
import Icon from "@/components/common/Icon.vue"
import { codeMemoryMessages } from "../codeMemoryMessages"
import CodeMemoryDrawer from "./CodeMemoryDrawer.vue"

const props = defineProps<{ projectId: number }>()
const { t } = useI18n({ messages: codeMemoryMessages })
const message = useMessage()
const entries = ref<CodeMemoryEntry[]>([])
const loading = ref(false)
const loadFailed = ref(false)
const saving = ref(false)
const removingId = ref(0)
const showDrawer = ref(false)
const setting = ref<CodeMemorySetting | null>(null)
const savingSetting = ref(false)
const accounts = ref<AIProviderAccount[]>([])

// 不做轮询：记忆只在执行结束后变化，点开时拉一次就够。
async function load() {
	if (loading.value) return
	loading.value = true
	try {
		const response = await getCodeMemories(props.projectId)
		if (response.code !== 0) throw new Error(response.message)
		entries.value = response.data.entries || []
		loadFailed.value = false
	} catch {
		loadFailed.value = true
	} finally {
		loading.value = false
	}
}

// 设置和列表一起拉：未配置时列表必然是空的，得同时知道「是没配」
// 还是「配了但还没攒出记忆」，否则提示会给错。
async function loadSetting() {
	try {
		const response = await getCodeMemorySetting()
		if (response.code === 0) setting.value = response.data
	} catch {
		void 0
	}
	try {
		const response = await getAIProviderAccounts()
		if (response.code === 0) accounts.value = (response.data || []).filter(account => account.useForMemoryExtraction)
	} catch {
		void 0
	}
}

function openDrawer() {
	showDrawer.value = true
	void load()
	void loadSetting()
}

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
	<n-button text size="tiny" @click="openDrawer">
		<template #icon><Icon name="mdi:brain" :size="14" /></template>
		{{ t("code.memoryEntry") }}
	</n-button>

	<CodeMemoryDrawer
		v-model:show="showDrawer"
		:entries="entries"
		:setting="setting"
		:accounts="accounts"
		:loading="loading"
		:load-failed="loadFailed"
		:saving="saving"
		:saving-setting="savingSetting"
		:removing-id="removingId"
		@refresh="load"
		@add="add"
		@remove="remove"
		@save-setting="persistSetting"
	/>
</template>
