<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import FtEditor from "@/components/FtEditor/index.vue"
import Icon from "@/components/common/Icon.vue"
import { Languages } from "@/global/mimetype"
import { getCodeSessionFile, saveCodeSessionFile } from "@/api/modules/codeEditor"
import { codeEditorMessages } from "../codeEditorMessages"

interface EditorTab {
	path: string
	extension: string
	content: string
	originalContent: string
	loading: boolean
	error: string
}

const props = defineProps<{
	sessionId: number | null
	path: string
	extension: string
}>()

const emit = defineEmits<{
	(event: "active-path", path: string): void
	(event: "saved", path: string): void
}>()

const { t } = useI18n({ messages: codeEditorMessages })
const dialog = useDialog()
const message = useMessage()
const tabs = ref<EditorTab[]>([])
const activePath = ref("")
const saving = ref(false)

const activeTab = computed(() => tabs.value.find(tab => tab.path === activePath.value) || null)
const activeContent = computed({
	get: () => activeTab.value?.content || "",
	set: value => {
		if (activeTab.value) activeTab.value.content = value
	}
})
const activeLanguage = computed(() => {
	const extension = (activeTab.value?.extension || "").replace(/^\./, "").toLowerCase()
	return Languages.find(item => item.value.some(value => value.toLowerCase() === extension))?.label || "plaintext"
})
const isDirty = (tab: EditorTab) => tab.content !== tab.originalContent
const hasUnsavedChanges = computed(() => tabs.value.some(isDirty))

const activateTab = (path: string) => {
	activePath.value = path
	emit("active-path", path)
}

const loadTab = async (tab: EditorTab) => {
	if (!props.sessionId || tab.loading) return
	tab.loading = true
	tab.error = ""
	try {
		const response = await getCodeSessionFile(props.sessionId, tab.path)
		tab.content = response.data.content
		tab.originalContent = response.data.content
	} catch (error) {
		tab.error = error instanceof Error ? error.message : t("code.fileOpenFailed")
		message.error(tab.error)
	} finally {
		tab.loading = false
	}
}

const openFile = async () => {
	if (!props.sessionId || !props.path) return
	const existing = tabs.value.find(tab => tab.path === props.path)
	if (existing) {
		activateTab(existing.path)
		return
	}
	const tab: EditorTab = {
		path: props.path,
		extension: props.extension,
		content: "",
		originalContent: "",
		loading: false,
		error: ""
	}
	tabs.value.push(tab)
	activateTab(tab.path)
	await loadTab(tab)
}

const saveTab = async (tab = activeTab.value) => {
	if (!props.sessionId || !tab || tab.loading || saving.value || !isDirty(tab)) return
	saving.value = true
	try {
		await saveCodeSessionFile(props.sessionId, tab.path, tab.content)
		tab.originalContent = tab.content
		emit("saved", tab.path)
		message.success(t("code.fileSaved"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.fileSaveFailed"))
	} finally {
		saving.value = false
	}
}

const removeTab = (tab: EditorTab) => {
	const index = tabs.value.indexOf(tab)
	if (index < 0) return
	tabs.value.splice(index, 1)
	if (activePath.value === tab.path) {
		const nextTab = tabs.value[Math.min(index, tabs.value.length - 1)]
		activateTab(nextTab?.path || "")
	}
}

const closeTab = (tab: EditorTab) => {
	if (!isDirty(tab)) {
		removeTab(tab)
		return
	}
	dialog.warning({
		title: t("code.unsavedChanges"),
		content: t("code.unsavedChangesHint"),
		positiveText: t("code.saveAndClose"),
		negativeText: t("code.discardChanges"),
		onPositiveClick: async () => {
			await saveTab(tab)
			if (!isDirty(tab)) removeTab(tab)
		},
		onNegativeClick: () => removeTab(tab)
	})
}

const handleKeydown = (event: KeyboardEvent) => {
	if (event.key.toLowerCase() !== "s" || (!event.metaKey && !event.ctrlKey)) return
	event.preventDefault()
	void saveTab()
}

watch(
	() => props.sessionId,
	() => {
		tabs.value = []
		activateTab("")
	}
)
watch(
	() => [props.sessionId, props.path],
	() => void openFile()
)
window.addEventListener("keydown", handleKeydown)
onBeforeUnmount(() => window.removeEventListener("keydown", handleKeydown))
defineExpose({ hasUnsavedChanges })
</script>

<template>
	<div class="flex h-full min-h-0 flex-col bg-white">
		<div
			v-if="tabs.length"
			class="flex h-10 shrink-0 items-stretch overflow-x-auto border-b border-slate-200 bg-slate-50"
		>
			<button
				v-for="tab in tabs"
				:key="tab.path"
				type="button"
				class="group flex max-w-60 shrink-0 items-center gap-2 border-r border-slate-200 px-3 text-xs text-slate-500"
				:class="activePath === tab.path ? 'bg-white text-slate-800' : 'hover:bg-slate-100'"
				:title="tab.path"
				@click="activateTab(tab.path)"
			>
				<span v-if="isDirty(tab)" class="h-2 w-2 shrink-0 rounded-full bg-blue-500" />
				<Icon v-else name="mdi:file-code-outline" :size="15" />
				<span class="truncate">{{ tab.path.split("/").pop() }}</span>
				<span
					class="ml-auto rounded p-0.5 opacity-60 hover:bg-slate-200 hover:opacity-100"
					@click.stop="closeTab(tab)"
				>
					<Icon name="mdi:close" :size="14" />
				</span>
			</button>
		</div>

		<div v-if="activeTab" class="flex min-h-0 flex-1 flex-col">
			<div class="flex h-10 shrink-0 items-center justify-between border-b border-slate-200 px-3">
				<span class="truncate text-xs text-slate-400">{{ activeTab.path }}</span>
				<n-button
					size="tiny"
					type="primary"
					:loading="saving"
					:disabled="activeTab.loading || !isDirty(activeTab)"
					@click="saveTab()"
				>
					{{ t("code.saveFile") }}
				</n-button>
			</div>
			<n-spin :show="activeTab.loading" class="min-h-0 flex-1">
				<div v-if="activeTab.error" class="flex h-full items-center justify-center px-6">
					<n-result status="error" :title="t('code.fileOpenFailed')" :description="activeTab.error">
						<template #footer>
							<n-button @click="loadTab(activeTab)">{{ t("code.retry") }}</n-button>
						</template>
					</n-result>
				</div>
				<FtEditor
					v-else
					:key="`${sessionId}-${activeTab.path}`"
					v-model="activeContent"
					:language="activeLanguage"
					height="100%"
					:show-toolbar="false"
				/>
			</n-spin>
		</div>

		<div
			v-else
			class="flex min-h-0 flex-1 items-center justify-center bg-[linear-gradient(180deg,#ffffff,#f8fafc)] px-6"
		>
			<n-empty :description="sessionId ? t('code.selectFileToEdit') : t('code.selectSessionToEdit')">
				<template #icon><Icon name="mdi:file-code-outline" :size="42" /></template>
			</n-empty>
		</div>
	</div>
</template>

<style scoped>
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
