<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { GetFilesList } from "@/api/modules/file"
import type { File } from "@/api/interface/file"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const props = defineProps<{
	show: boolean
	initialPath: string
	rootPath: string
}>()

const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "select", value: string): void
}>()

const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()
const currentPath = ref("/")
const rootPath = ref("/")
const directories = ref<File.File[]>([])
const loading = ref(false)
const loadError = ref(false)

const canGoParent = computed(() => currentPath.value !== rootPath.value)

const loadDirectories = async (path: string, fallbackPath = "") => {
	loading.value = true
	loadError.value = false
	try {
		const response = await GetFilesList({
			path,
			page: 1,
			limit: 500,
			expand: true,
			dir: true,
			showHidden: false,
			sortBy: "name",
			sortOrder: "ascending"
		})
		currentPath.value = response.data?.path || path
		directories.value = (response.data?.items || []).filter(item => item.isDir)
	} catch {
		if (fallbackPath && fallbackPath !== path) {
			await loadDirectories(fallbackPath)
			return
		}
		loadError.value = true
		directories.value = []
		message.error(t("code.directoryLoadFailed"))
	} finally {
		loading.value = false
	}
}

const goParent = () => {
	if (!canGoParent.value) return
	const slashIndex = currentPath.value.lastIndexOf("/")
	const backslashIndex = currentPath.value.lastIndexOf("\\")
	const separatorIndex = Math.max(slashIndex, backslashIndex)
	const candidate = separatorIndex > 0 ? currentPath.value.slice(0, separatorIndex) : rootPath.value
	const parentPath = candidate.length >= rootPath.value.length ? candidate : rootPath.value
	void loadDirectories(parentPath || rootPath.value)
}

const selectCurrent = () => {
	emit("select", currentPath.value)
	emit("update:show", false)
}

watch(
	() => props.show,
	show => {
		if (show) {
			rootPath.value = props.rootPath || "/"
			void loadDirectories(props.initialPath || rootPath.value, rootPath.value)
		}
	}
)
</script>

<template>
	<n-modal
		:show="show"
		preset="card"
		style="width: min(680px, calc(100vw - 32px))"
		:title="t('code.selectProjectDirectory')"
		@update:show="emit('update:show', $event)"
	>
		<div class="flex min-h-[420px] flex-col gap-4">
			<div class="flex items-center gap-2 rounded-xl bg-[var(--n-color-embedded)] p-2">
				<n-button quaternary :disabled="!canGoParent || loading" @click="goParent">
					{{ t("code.parentDirectory") }}
				</n-button>
				<div class="min-w-0 flex-1 truncate px-2 font-mono text-sm" :title="currentPath">
					{{ currentPath }}
				</div>
				<n-button quaternary :loading="loading" @click="loadDirectories(currentPath)">
					{{ t("code.refreshDirectory") }}
				</n-button>
			</div>

			<n-spin v-if="loading" class="flex flex-1 items-center justify-center" />
			<n-alert v-else-if="loadError" type="error" :show-icon="false">
				<div class="flex items-center justify-between gap-3">
					<span>{{ t("code.directoryLoadFailed") }}</span>
					<n-button text type="primary" @click="loadDirectories(currentPath)">{{ t("code.retry") }}</n-button>
				</div>
			</n-alert>
			<n-empty v-else-if="directories.length === 0" class="flex-1" :description="t('code.noSubdirectories')" />
			<n-scrollbar v-else class="min-h-0 flex-1">
				<div class="grid gap-2 sm:grid-cols-2">
					<button
						v-for="directory in directories"
						:key="directory.path"
						type="button"
						class="flex items-center gap-3 rounded-xl border border-[var(--n-border-color)] p-3 text-left transition-colors hover:border-[var(--n-primary-color)] hover:bg-[var(--n-color-hover)]"
						@click="loadDirectories(directory.path)"
					>
						<Icon name="mdi:folder-outline" :size="22" class="text-amber-500" />
						<span class="min-w-0 flex-1 truncate text-sm">{{ directory.name }}</span>
					</button>
				</div>
			</n-scrollbar>
		</div>

		<template #footer>
			<div class="flex items-center justify-between gap-3">
				<span class="min-w-0 truncate text-xs text-[var(--n-text-color-3)]" :title="currentPath">
					{{ t("code.currentDirectory") }}：{{ currentPath }}
				</span>
				<div class="flex shrink-0 gap-3">
					<n-button @click="emit('update:show', false)">{{ t("code.cancel") }}</n-button>
					<n-button type="primary" :disabled="loading || loadError" @click="selectCurrent">
						{{ t("code.selectThisDirectory") }}
					</n-button>
				</div>
			</div>
		</template>
	</n-modal>
</template>
