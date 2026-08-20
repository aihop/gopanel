<script setup lang="ts">
import { computed, h, ref, watch } from "vue"
import type { TreeOption } from "naive-ui"
import { NButton, NPopover, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { getCodeSessionStructure, searchCodeSessionStructure, type CodeStructureEntry, type CodeStructureSearchHit } from "@/api/modules/codeEditor"
import { codeEditorMessages } from "../codeEditorMessages"
import { writeStructureDragData } from "./codeConversationAttachments"
import CodeStructureSearchHits from "./CodeStructureSearchHits.vue"
import CodeStructureSnippetPopover from "./CodeStructureSnippetPopover.vue"

interface StructureTreeOption extends TreeOption {
	key: string
	label: string
	isDir: boolean
	children?: StructureTreeOption[]
}

const props = withDefaults(
	defineProps<{
		sessionId: number
		changedFiles?: string[]
		selectedPath?: string
		attachToChat?: boolean
	}>(),
	{ changedFiles: () => [], selectedPath: "", attachToChat: false }
)

const emit = defineEmits<{
	(event: "select-file", file: { path: string; extension: string; isDir?: boolean }): void
	(event: "insert-snippet", snippet: { path: string; startLine: number; endLine: number }): void
}>()

const { t } = useI18n({ messages: codeEditorMessages })
const message = useMessage()
const loading = ref(false)
const loadError = ref(false)
const truncated = ref(false)
const pattern = ref("")
const nodes = ref<StructureTreeOption[]>([])
const expandedKeys = ref<Array<string | number>>([])
const selectedKeys = ref<Array<string | number>>(props.selectedPath ? [props.selectedPath] : [])
const snippetPath = ref("")
const searchHits = ref<CodeStructureSearchHit[]>([])
const searchLoading = ref(false)
const searchTruncated = ref(false)
const searchSnippetPath = ref("")
const searchSnippetLine = ref(0)
const searching = computed(() => pattern.value.trim().length >= 2)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const normalizedChangedFiles = computed(() =>
	props.changedFiles.map(file => file.replaceAll("\\", "/").replace(/^\.\//, "").replace(/^\//, ""))
)

const isChangedPath = (filePath: string, isDir: boolean) =>
	normalizedChangedFiles.value.some(changedPath =>
		isDir
			? changedPath.startsWith(`${filePath}/`) || changedPath.includes(`/${filePath}/`)
			: changedPath === filePath || changedPath.endsWith(`/${filePath}`)
	)

const toTreeOption = (entry: CodeStructureEntry): StructureTreeOption => ({
	key: entry.path,
	label: entry.name,
	isDir: entry.isDir,
	isLeaf: !entry.isDir
})

const loadRoot = async () => {
	loading.value = true
	loadError.value = false
	try {
		const response = await getCodeSessionStructure(props.sessionId)
		nodes.value = response.data.entries.map(toTreeOption)
		truncated.value = response.data.truncated
		const directories = nodes.value.filter(node => node.isDir).slice(0, 40)
		await Promise.all(directories.map(node => loadChildren(node).catch(() => undefined)))
		expandedKeys.value = directories.map(node => node.key)
	} catch {
		nodes.value = []
		expandedKeys.value = []
		loadError.value = true
	} finally {
		loading.value = false
	}
}

const loadChildren = async (option: TreeOption) => {
	const node = option as StructureTreeOption
	try {
		const response = await getCodeSessionStructure(props.sessionId, node.key)
		node.children = response.data.entries.map(toTreeOption)
		if (response.data.truncated) message.warning(t("code.structureTruncated"))
	} catch (error) {
		throw error
	}
}

const renderPrefix = ({ option }: { option: TreeOption }) => {
	const node = option as StructureTreeOption
	return h(Icon, {
		name: node.isDir ? "mdi:folder-outline" : "mdi:file-code-outline",
		size: 17,
		color: isChangedPath(node.key, node.isDir) ? "#2563eb" : undefined
	})
}

const renderLabel = ({ option }: { option: TreeOption }) => {
	const node = option as StructureTreeOption
	const label = h(
		"span",
		{
			class: [
				"min-w-0 flex-1 truncate",
				isChangedPath(node.key, node.isDir) ? "font-semibold text-blue-600" : "text-[var(--n-text-color)]",
				props.attachToChat ? "cursor-grab" : "",
			],
			title: node.key,
			draggable: true,
			onDragstart: (event: DragEvent) => {
				if (!event.dataTransfer) return
				event.stopPropagation()
				writeStructureDragData(event.dataTransfer, node.key)
			},
		},
		node.label,
	)
	if (node.isDir) {
		return h("span", { class: "structure-label flex min-w-0 items-center gap-1" }, [label])
	}
	return h("span", { class: "structure-label flex min-w-0 items-center gap-1" }, [
		label,
		h("span", { class: "structure-edit shrink-0" }, [
			h(
				NPopover,
				{
					show: snippetPath.value === node.key,
					trigger: "click",
					placement: "left",
					showArrow: true,
					to: "body",
					style: "padding: 12px;",
					onUpdateShow: (show: boolean) => {
						snippetPath.value = show ? node.key : snippetPath.value === node.key ? "" : snippetPath.value
					},
				},
				{
					trigger: () =>
						h(
							NButton,
							{
								quaternary: true,
								circle: true,
								size: "tiny",
								title: t("code.editFileSnippet"),
								onClick: (event: MouseEvent) => event.stopPropagation(),
							},
							{ icon: () => h(Icon, { name: "mdi:pencil-outline", size: 14 }) },
						),
					default: () =>
						h(CodeStructureSnippetPopover, {
							sessionId: props.sessionId,
							path: node.key,
							attachToChat: props.attachToChat,
							onInsert: (snippet: { path: string; startLine: number; endLine: number }) => emit("insert-snippet", snippet),
							onClose: () => {
								snippetPath.value = ""
							},
						}),
				},
			),
		]),
	])
}

const runStructureSearch = async (query: string) => {
	searchLoading.value = true
	try {
		const response = await searchCodeSessionStructure(props.sessionId, query)
		if (pattern.value.trim() !== query) return
		searchHits.value = response.data.hits || []
		searchTruncated.value = Boolean(response.data.truncated)
	} catch {
		if (pattern.value.trim() === query) searchHits.value = []
	} finally {
		if (pattern.value.trim() === query) searchLoading.value = false
	}
}

const openSearchHit = (hit: CodeStructureSearchHit) => {
	selectedKeys.value = [hit.path]
	emit("select-file", { path: hit.path, extension: hit.extension || "", isDir: hit.isDir })
	if (hit.isDir) return
	searchSnippetPath.value = hit.path
	searchSnippetLine.value = hit.line || 0
	snippetPath.value = ""
}

const selectFile = (node: StructureTreeOption) => {
	selectedKeys.value = [node.key]
	emit("select-file", {
		path: node.key,
		extension: node.isDir ? "" : node.label.split(".").pop() || "",
		isDir: node.isDir,
	})
}

const handleSelectedKeys = (
	_keys: Array<string | number>,
	_options: Array<TreeOption | null>,
	meta: { node: TreeOption | null; action: "select" | "unselect" }
) => {
	if (meta.action === "select" && meta.node) selectFile(meta.node as StructureTreeOption)
}

watch(
	() => props.sessionId,
	() => void loadRoot(),
	{ immediate: true }
)
watch(
	() => props.selectedPath,
	path => {
		if (path) selectedKeys.value = [path]
	}
)
watch(
	() => pattern.value.trim(),
	query => {
		if (searchTimer) clearTimeout(searchTimer)
		if (query.length < 2) {
			searchHits.value = []
			searchTruncated.value = false
			searchLoading.value = false
			return
		}
		searchTimer = setTimeout(() => void runStructureSearch(query), 300)
	}
)
</script>

<template>
	<div
		class="structure-panel flex h-full min-h-0 flex-col overflow-hidden"
		:class="attachToChat ? 'structure-panel--chat bg-transparent' : 'bg-white'"
	>
		<div class="border-b border-slate-200/70 px-3 py-3 dark:border-white/10">
			<div class="flex items-center justify-between gap-2">
				<div class="flex min-w-0 items-center gap-2">
					<Icon name="mdi:file-tree-outline" :size="19" />
					<div class="min-w-0">
						<div class="truncate text-sm font-semibold text-slate-700 dark:text-[var(--n-text-color)]">{{ t("code.projectStructure") }}</div>
						<div v-if="attachToChat" class="truncate text-[11px] tracking-[0.01em] text-[var(--n-text-color-3)]">
							{{ t("code.dragStructureToChat") }}
						</div>
					</div>
				</div>
				<n-button
					quaternary
					circle
					size="small"
					:loading="loading"
					:title="t('code.refreshStructure')"
					@click="loadRoot"
				>
					<template #icon><Icon name="mdi:refresh" :size="17" /></template>
				</n-button>
			</div>
			<n-input
				v-model:value="pattern"
				clearable
				size="small"
				class="mt-3"
				:placeholder="t('code.filterStructure')"
			>
				<template #prefix><Icon name="mdi:magnify" :size="16" /></template>
			</n-input>
		</div>

		<div class="relative min-h-0 flex-1 overflow-hidden">
			<CodeStructureSearchHits
				v-if="searching"
				:hits="searchHits"
				:loading="searchLoading"
				:truncated="searchTruncated"
				@select="openSearchHit"
			/>
			<div
				v-else
				class="structure-scroll absolute inset-0 overflow-auto px-2 py-3"
			>
				<div
					v-if="loadError"
					class="flex h-full min-h-48 flex-col items-center justify-center gap-3 px-4 text-center"
				>
					<span class="text-sm text-slate-500">{{ t("code.structureLoadFailed") }}</span>
					<n-button size="small" @click="loadRoot">{{ t("code.retry") }}</n-button>
				</div>
				<n-empty
					v-else-if="!loading && nodes.length === 0"
					size="small"
					:description="t('code.structureEmpty')"
					class="py-16"
				/>
				<n-tree
					v-else
					block-line
					:cancelable="false"
					:data="nodes"
					:expanded-keys="expandedKeys"
					:on-load="loadChildren"
					:render-prefix="renderPrefix"
					:render-label="renderLabel"
					:selected-keys="selectedKeys"
					@update:expanded-keys="expandedKeys = $event"
					@update:selected-keys="handleSelectedKeys"
				/>
			</div>
			<div
				v-if="loading && !searching"
				class="absolute inset-0 z-10 flex items-center justify-center bg-white/70 backdrop-blur-[1px]"
			>
				<n-spin />
			</div>
			<n-popover
				:show="Boolean(searchSnippetPath)"
				trigger="manual"
				placement="left"
				to="body"
				style="padding: 12px"
				@update:show="show => { if (!show) searchSnippetPath = '' }"
			>
				<template #trigger>
					<span class="pointer-events-none absolute right-2 top-2 h-px w-px" />
				</template>
				<CodeStructureSnippetPopover
					v-if="searchSnippetPath"
					:session-id="sessionId"
					:path="searchSnippetPath"
					:attach-to-chat="attachToChat"
					:initial-query="pattern.trim()"
					:initial-line="searchSnippetLine"
					@insert="emit('insert-snippet', $event)"
					@close="searchSnippetPath = ''"
				/>
			</n-popover>
		</div>

		<div v-if="truncated || changedFiles.length" class="border-t border-slate-200 px-3 py-2 text-xs text-slate-400">
			<span v-if="changedFiles.length" class="text-blue-600">● {{ t("code.changedFileMarker") }}</span>
			<span v-if="truncated" class="ml-2">{{ t("code.structureTruncated") }}</span>
		</div>
	</div>
</template>

<style scoped>
.theme-dark .structure-panel {
	background-color: var(--bg-default-color);
}

.theme-dark .structure-panel--chat {
	background-color: transparent;
}

:deep(.structure-edit) {
	opacity: 0;
	pointer-events: none;
	transition: opacity 0.12s ease;
}

:deep(.n-tree-node:hover .structure-edit),
:deep(.n-tree-node-content:hover .structure-edit),
:deep(.structure-label:hover .structure-edit),
:deep(.structure-label:focus-within .structure-edit) {
	opacity: 1;
	pointer-events: auto;
}

.structure-scroll {
	scrollbar-gutter: stable;
	scrollbar-color: rgba(100, 116, 139, 0.65) rgba(226, 232, 240, 0.55);
	scrollbar-width: thin;
}

.structure-scroll::-webkit-scrollbar {
	width: 8px;
}

.structure-scroll::-webkit-scrollbar-thumb {
	border-radius: 999px;
	background: rgba(100, 116, 139, 0.65);
}

.structure-scroll::-webkit-scrollbar-track {
	background: rgba(226, 232, 240, 0.55);
}
</style>
