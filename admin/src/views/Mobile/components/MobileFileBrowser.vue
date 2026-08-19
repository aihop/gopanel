<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useDialog, useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import {
	getMobileSessionFile,
	getMobileSessionStructure,
	saveMobileSessionFile,
	type MobileCodeStructureEntry
} from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"

const props = defineProps<{ show: boolean; sessionId: number }>()
const emit = defineEmits<{ (event: "update:show", value: boolean): void }>()
const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const dialog = useDialog()
const currentPath = ref("")
const entries = ref<MobileCodeStructureEntry[]>([])
const loading = ref(false)
const loadError = ref("")
const filePath = ref("")
const content = ref("")
const originalContent = ref("")
const fileVersion = ref("")
const saving = ref(false)

const editing = computed(() => !!filePath.value)
const dirty = computed(() => content.value !== originalContent.value)
const breadcrumbs = computed(() => (currentPath.value ? currentPath.value.split("/") : []))

async function loadDirectory(path = "") {
	loading.value = true
	loadError.value = ""
	try {
		const result = await getMobileSessionStructure(props.sessionId, path)
		currentPath.value = result.path
		entries.value = result.entries || []
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.filesLoadFailed")
	} finally {
		loading.value = false
	}
}

async function openEntry(entry: MobileCodeStructureEntry) {
	if (entry.isDir) {
		await loadDirectory(entry.path)
		return
	}
	loading.value = true
	try {
		const result = await getMobileSessionFile(props.sessionId, entry.path)
		filePath.value = result.path
		content.value = result.content
		originalContent.value = result.content
		fileVersion.value = result.version
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.filesLoadFailed"))
	} finally {
		loading.value = false
	}
}

function closeEditor() {
	if (!dirty.value) {
		filePath.value = ""
		return
	}
	dialog.warning({
		title: t("mobile.unsavedChanges"),
		content: t("mobile.unsavedChangesHint"),
		positiveText: t("mobile.discard"),
		negativeText: t("mobile.cancel"),
		onPositiveClick: () => {
			filePath.value = ""
		}
	})
}

function requestClose() {
	if (!editing.value || !dirty.value) {
		emit("update:show", false)
		return
	}
	dialog.warning({
		title: t("mobile.unsavedChanges"),
		content: t("mobile.unsavedChangesHint"),
		positiveText: t("mobile.discard"),
		negativeText: t("mobile.cancel"),
		onPositiveClick: () => emit("update:show", false)
	})
}

async function save() {
	if (!filePath.value || !dirty.value || saving.value) return
	saving.value = true
	try {
		const result = await saveMobileSessionFile(props.sessionId, filePath.value, content.value, fileVersion.value)
		originalContent.value = content.value
		fileVersion.value = result.version
		message.success(t("mobile.fileSaved"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.fileSaveFailed"))
	} finally {
		saving.value = false
	}
}

function openBreadcrumb(index: number) {
	void loadDirectory(breadcrumbs.value.slice(0, index + 1).join("/"))
}

watch(
	() => [props.show, props.sessionId] as const,
	([show]) => {
		if (!show || !props.sessionId) return
		filePath.value = ""
		void loadDirectory()
	}
)
</script>

<template>
	<n-drawer
		:show="show"
		placement="right"
		style="width: min(760px, 100vw)"
		@mask-click="requestClose"
		@esc="requestClose"
	>
		<n-drawer-content :closable="false" body-content-style="padding: 0;">
			<template #header>
				<div class="flex w-full min-w-0 items-center justify-between gap-3">
					<span class="min-w-0 truncate font-semibold">
						{{ editing ? filePath : t("mobile.projectFiles") }}
					</span>
					<n-button quaternary circle :title="t('mobile.close')" @click="requestClose">
						<template #icon><Icon name="mdi:close" /></template>
					</n-button>
				</div>
			</template>
			<n-spin :show="loading" class="h-full">
				<div v-if="editing" class="flex h-full min-h-0 flex-col">
					<div class="flex items-center justify-between border-b border-slate-200 px-3 py-2">
						<n-button quaternary size="small" @click="closeEditor">
							<template #icon><Icon name="mdi:arrow-left" /></template>
							{{ t("mobile.files") }}
						</n-button>
						<span class="text-xs" :class="dirty ? 'text-amber-600' : 'text-slate-400'">
							{{ dirty ? t("mobile.unsaved") : t("mobile.saved") }}
						</span>
					</div>
					<n-input
						v-model:value="content"
						type="textarea"
						:bordered="false"
						class="mobile-code-editor min-h-0 flex-1"
					/>
				</div>
				<div v-else class="h-full overflow-y-auto p-3">
					<div class="mb-3 flex flex-wrap items-center gap-1 text-sm">
						<n-button text type="primary" @click="loadDirectory()">
							{{ t("mobile.rootDirectory") }}
						</n-button>
						<template v-for="(part, index) in breadcrumbs" :key="`${part}-${index}`">
							<span class="text-slate-300">/</span>
							<n-button text @click="openBreadcrumb(index)">{{ part }}</n-button>
						</template>
					</div>
					<n-alert v-if="loadError" type="error" :title="t('mobile.filesLoadFailed')">
						<div class="flex items-center justify-between gap-3">
							<span>{{ loadError }}</span>
							<n-button size="small" @click="loadDirectory(currentPath)">
								{{ t("mobile.retry") }}
							</n-button>
						</div>
					</n-alert>
					<n-empty
						v-else-if="!loading && entries.length === 0"
						:description="t('mobile.directoryEmpty')"
						class="py-16"
					/>
					<button
						v-for="entry in entries"
						:key="entry.path"
						type="button"
						class="flex w-full items-center gap-3 rounded-xl border-0 bg-transparent px-3 py-3 text-left hover:bg-slate-100 active:bg-slate-200"
						@click="openEntry(entry)"
					>
						<Icon
							:name="entry.isDir ? 'mdi:folder-outline' : 'mdi:file-code-outline'"
							:size="20"
							:color="entry.isDir ? '#2563eb' : '#64748b'"
						/>
						<span class="min-w-0 flex-1 truncate text-sm">{{ entry.name }}</span>
						<Icon name="mdi:chevron-right" :size="18" color="#94a3b8" />
					</button>
				</div>
			</n-spin>
			<template v-if="editing" #footer>
				<n-button type="primary" block size="large" :loading="saving" :disabled="!dirty" @click="save">
					{{ t("mobile.saveFile") }}
				</n-button>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<style scoped>
.mobile-code-editor :deep(textarea) {
	height: 100% !important;
	min-height: 55dvh !important;
	padding: 16px !important;
	font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	font-size: 13px;
	line-height: 1.65;
	white-space: pre;
	overflow: auto;
}
</style>
