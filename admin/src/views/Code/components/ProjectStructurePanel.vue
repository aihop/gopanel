<script setup lang="ts">
import { computed, h, ref, watch } from "vue"
import type { TreeOption } from "naive-ui"
import { NButton, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { getCodeSessionStructure, searchCodeSessionStructure, type CodeStructureEntry, type CodeStructureSearchHit } from "@/api/modules/codeEditor"
import { codeEditorMessages } from "../codeEditorMessages"
import { writeStructureDragData } from "./codeConversationAttachments"
import { codeFileIcon } from "./codeFileIcon"
import { structureAncestorDirs } from "./codeFileSnippet"
import CodeStructureSearchHits from "./CodeStructureSearchHits.vue"

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
		closable?: boolean
	}>(),
	{ changedFiles: () => [], selectedPath: "", attachToChat: false, closable: false }
)

const emit = defineEmits<{
	(event: "select-file", file: { path: string; extension: string; isDir?: boolean }): void
	(event: "attach-file", file: { path: string; extension: string; isDir?: boolean }): void
	(event: "locate-file", target: { path: string; line?: number; query?: string }): void
	(event: "close"): void
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
const searchHits = ref<CodeStructureSearchHit[]>([])
const searchLoading = ref(false)
const searchTruncated = ref(false)
const searchRoot = ref("")
const searching = computed(() => pattern.value.trim().length >= 2)
const searchHitPaths = computed(() => new Set(searchHits.value.map(hit => hit.path)))
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
	const icon = codeFileIcon(node.key, node.isDir)
	return h(Icon, {
		name: icon.name,
		size: 17,
		color: isChangedPath(node.key, node.isDir) ? "#2563eb" : icon.color
	})
}

const renderLabel = ({ option }: { option: TreeOption }) => {
	const node = option as StructureTreeOption
	const hit = searchHitPaths.value.has(node.key)
	const label = h(
		"span",
		{
			class: [
				"min-w-0 flex-1 truncate",
				isChangedPath(node.key, node.isDir)
					? "font-semibold text-blue-600"
					: hit
						? "font-medium text-amber-700 dark:text-amber-300"
						: "text-[var(--n-text-color)]",
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
	const actions = []
	if (props.attachToChat) {
		actions.push(
			h(
				NButton,
				{
					quaternary: true,
					circle: true,
					size: "tiny",
					title: t("code.attachStructureFile"),
					onClick: (event: MouseEvent) => {
						event.stopPropagation()
						emit("attach-file", {
							path: node.key,
							extension: node.isDir ? "" : node.label.split(".").pop() || "",
							isDir: node.isDir,
						})
					},
				},
				{ icon: () => h(Icon, { name: "mdi:paperclip", size: 14 }) },
			),
		)
	}
	if (!node.isDir) {
		actions.push(
			h(
				NButton,
				{
					quaternary: true,
					circle: true,
					size: "tiny",
					title: t("code.editFileSnippet"),
					onClick: (event: MouseEvent) => {
						event.stopPropagation()
						emit("locate-file", { path: node.key })
					},
				},
				{ icon: () => h(Icon, { name: "mdi:code-braces", size: 14 }) },
			),
		)
	}
	return h("span", { class: "structure-label flex min-w-0 items-center gap-1" }, [
		label,
		actions.length ? h("span", { class: "structure-edit flex shrink-0 items-center" }, actions) : null,
	])
}

const findStructureNode = (list: StructureTreeOption[], key: string): StructureTreeOption | null => {
	for (const node of list) {
		if (node.key === key) return node
		if (node.children?.length) {
			const found = findStructureNode(node.children, key)
			if (found) return found
		}
	}
	return null
}

const expandAncestors = async (filePath: string) => {
	const dirs = structureAncestorDirs(filePath)
	const expanded = new Set(expandedKeys.value.map(String))
	for (const dir of dirs) {
		expanded.add(dir)
		const node = findStructureNode(nodes.value, dir)
		if (node && !node.children) await loadChildren(node).catch(() => undefined)
	}
	expandedKeys.value = [...expanded]
}

const revealPath = async (filePath: string) => {
	await expandAncestors(filePath)
	selectedKeys.value = [filePath]
}

const runStructureSearch = async (query: string) => {
	searchLoading.value = true
	try {
		const response = await searchCodeSessionStructure(props.sessionId, query, searchRoot.value)
		if (pattern.value.trim() !== query) return
		searchHits.value = response.data.hits || []
		searchTruncated.value = Boolean(response.data.truncated)
		const unique = [...new Set(searchHits.value.map(hit => hit.path))].slice(0, 8)
		for (const path of unique) await expandAncestors(path)
	} catch {
		if (pattern.value.trim() === query) searchHits.value = []
	} finally {
		if (pattern.value.trim() === query) searchLoading.value = false
	}
}

const openSearchHit = async (hit: CodeStructureSearchHit) => {
	await revealPath(hit.path)
	if (hit.isDir) searchRoot.value = hit.path
	emit("select-file", { path: hit.path, extension: hit.extension || "", isDir: hit.isDir })
	if (!hit.isDir) emit("locate-file", { path: hit.path, line: hit.line, query: pattern.value.trim() })
}

const selectFile = (node: StructureTreeOption) => {
	selectedKeys.value = [node.key]
	searchRoot.value = node.isDir ? node.key : ""
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
	[() => pattern.value.trim(), searchRoot],
	([query]) => {
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
				<div class="flex shrink-0 items-center">
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
					<n-button
						v-if="closable"
						quaternary
						circle
						size="small"
						:title="t('code.hideProjectStructure')"
						@click="emit('close')"
					>
						<template #icon><Icon name="mdi:close" :size="17" /></template>
					</n-button>
				</div>
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

		<div class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
			<div v-if="searching" class="max-h-40 shrink-0 border-b border-slate-200/70 dark:border-white/10">
				<CodeStructureSearchHits
					class="max-h-40"
					:hits="searchHits"
					:loading="searchLoading"
					:truncated="searchTruncated"
					@select="openSearchHit"
				/>
			</div>
			<div
				class="structure-scroll relative min-h-0 flex-1 overflow-auto px-2 py-3"
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
				v-if="loading"
				class="absolute inset-0 z-10 flex items-center justify-center bg-white/70 backdrop-blur-[1px]"
			>
				<n-spin />
			</div>
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

.structure-scroll,
:deep(.structure-search-scroll) {
	scrollbar-gutter: stable;
	scrollbar-width: thin;
	scrollbar-color: rgb(148 163 184 / 0.28) transparent;
}

.structure-scroll::-webkit-scrollbar,
:deep(.structure-search-scroll::-webkit-scrollbar) {
	width: 5px;
	height: 5px;
}

.structure-scroll::-webkit-scrollbar-track,
:deep(.structure-search-scroll::-webkit-scrollbar-track) {
	background: transparent;
}

.structure-scroll::-webkit-scrollbar-thumb,
:deep(.structure-search-scroll::-webkit-scrollbar-thumb) {
	border-radius: 999px;
	background: rgb(148 163 184 / 0.28);
}

.structure-scroll::-webkit-scrollbar-thumb:hover,
:deep(.structure-search-scroll::-webkit-scrollbar-thumb:hover) {
	background: rgb(148 163 184 / 0.42);
}

.structure-scroll::-webkit-scrollbar-corner,
:deep(.structure-search-scroll::-webkit-scrollbar-corner) {
	background: transparent;
}
</style>
