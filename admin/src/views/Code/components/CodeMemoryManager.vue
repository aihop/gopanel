<script setup lang="ts">
import { ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { createCodeMemory, deleteCodeMemory, getCodeMemories } from "@/api/modules/code"
import type { CodeMemoryEntry } from "@/api/interface/codeMemories"
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

function openDrawer() {
	showDrawer.value = true
	void load()
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
		:loading="loading"
		:load-failed="loadFailed"
		:saving="saving"
		:removing-id="removingId"
		@refresh="load"
		@add="add"
		@remove="remove"
	/>
</template>
