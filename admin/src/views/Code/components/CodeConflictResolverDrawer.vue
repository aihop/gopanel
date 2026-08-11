<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import FtEditor from "@/components/FtEditor/index.vue"
import { Languages } from "@/global/mimetype"
import {
	completeCodeDeliveryConflicts,
	getCodeDeliveryConflictFile,
	getCodeDeliveryConflicts,
	saveCodeDeliveryConflictFile
} from "@/api/modules/codeGit"
import type {
	CodeDeliveryConflictFile,
	CodeDeliveryConflictRepository,
	CodeDeliveryConflictResolution,
	CodeDeliveryConflicts,
	CodeDeliveryJob
} from "@/api/interface/codeGit"
import { codeConflictResolverMessages } from "../codeConflictResolverMessages"

const props = defineProps<{
	show: boolean
	sessionId: number
	mobile?: boolean
	loadConflicts?: (sessionId: number) => Promise<CodeDeliveryConflicts>
	loadConflictFile?: (sessionId: number, repositoryId: string, path: string) => Promise<CodeDeliveryConflictFile>
	saveConflictFile?: (
		sessionId: number,
		repositoryId: string,
		path: string,
		resolution: CodeDeliveryConflictResolution,
		content: string,
		baseVersion: string
	) => Promise<CodeDeliveryConflictFile>
	completeConflicts?: (sessionId: number) => Promise<CodeDeliveryJob>
}>()
const emit = defineEmits<{
	(event: "update:show", show: boolean): void
	(event: "completed"): void
}>()
const { t } = useI18n({ messages: codeConflictResolverMessages })
const dialog = useDialog()
const message = useMessage()
const repositories = ref<CodeDeliveryConflictRepository[]>([])
const selectedRepositoryId = ref("")
const selectedPath = ref("")
const file = ref<CodeDeliveryConflictFile | null>(null)
const resultContent = ref("")
const originalContent = ref("")
const resolution = ref<CodeDeliveryConflictResolution>("content")
const loading = ref(false)
const fileLoading = ref(false)
const saving = ref(false)
const completing = ref(false)
const loadError = ref("")
const mobilePane = ref<"main" | "task" | "result">("result")

const totalFiles = computed(() => repositories.value.reduce((total, repository) => total + repository.total, 0))
const resolvedFiles = computed(() => repositories.value.reduce((total, repository) => total + repository.resolved, 0))
const allResolved = computed(() => totalFiles.value > 0 && resolvedFiles.value === totalFiles.value)
const isDirty = computed(
	() => Boolean(file.value) && (resultContent.value !== originalContent.value || resolution.value !== "content")
)
const language = computed(() => {
	const extension = (selectedPath.value.split(".").pop() || "").toLowerCase()
	return Languages.find(item => item.value.some(value => value.toLowerCase() === extension))?.label || "plaintext"
})
const editableContent = computed({
	get: () => resultContent.value,
	set: value => {
		resultContent.value = value
		resolution.value = "content"
	}
})

const fileKey = (repositoryId: string, path: string) => `${repositoryId}:${path}`
const isUnresolved = (repository: CodeDeliveryConflictRepository, path: string) =>
	repository.unresolvedFiles.includes(path)

const requestConflicts = async () =>
	props.loadConflicts ? props.loadConflicts(props.sessionId) : (await getCodeDeliveryConflicts(props.sessionId)).data
const requestConflictFile = async (repositoryId: string, path: string) =>
	props.loadConflictFile
		? props.loadConflictFile(props.sessionId, repositoryId, path)
		: (await getCodeDeliveryConflictFile(props.sessionId, repositoryId, path)).data
const requestSave = async () => {
	if (!file.value) throw new Error(t("code.conflictSelectFile"))
	return props.saveConflictFile
		? props.saveConflictFile(
				props.sessionId,
				file.value.repositoryId,
				file.value.path,
				resolution.value,
				resultContent.value,
				file.value.version
			)
		: (
				await saveCodeDeliveryConflictFile(
					props.sessionId,
					file.value.repositoryId,
					file.value.path,
					resolution.value,
					resultContent.value,
					file.value.version
				)
			).data
}
const requestComplete = async () =>
	props.completeConflicts
		? props.completeConflicts(props.sessionId)
		: (await completeCodeDeliveryConflicts(props.sessionId)).data

const loadRepositories = async () => {
	repositories.value = (await requestConflicts()).repositories
}

const loadFile = async (repositoryId: string, path: string) => {
	selectedRepositoryId.value = repositoryId
	selectedPath.value = path
	fileLoading.value = true
	file.value = null
	try {
		file.value = await requestConflictFile(repositoryId, path)
		resultContent.value = file.value.resultContent || ""
		originalContent.value = resultContent.value
		resolution.value = "content"
		mobilePane.value = "result"
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.conflictResolverLoadFailed"))
	} finally {
		fileLoading.value = false
	}
}

const selectFile = (repositoryId: string, path: string) => {
	if (fileKey(repositoryId, path) === fileKey(selectedRepositoryId.value, selectedPath.value)) return
	const select = () => void loadFile(repositoryId, path)
	if (!isDirty.value) {
		select()
		return
	}
	dialog.warning({
		title: t("code.conflictResolverTitle"),
		content: t("code.conflictUnsavedChanges"),
		positiveText: t("code.conflictDiscard"),
		negativeText: t("code.conflictKeepEditing"),
		onPositiveClick: select
	})
}

const chooseResolution = (next: "main" | "task" | "delete") => {
	if (!file.value) return
	resolution.value = next
	if (next === "main") resultContent.value = file.value.mainContent || ""
	if (next === "task") resultContent.value = file.value.taskContent || ""
	if (next === "delete") resultContent.value = ""
}

const saveFile = async () => {
	if (!file.value || saving.value) return
	saving.value = true
	try {
		file.value = await requestSave()
		resultContent.value = file.value.resultContent || ""
		originalContent.value = resultContent.value
		resolution.value = "content"
		await loadRepositories()
		message.success(t("code.conflictSaved"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.conflictSaveFailed"))
	} finally {
		saving.value = false
	}
}

const complete = async () => {
	if (!allResolved.value || completing.value) return
	completing.value = true
	try {
		await requestComplete()
		message.success(t("code.conflictCompleteSuccess"))
		emit("update:show", false)
		emit("completed")
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.conflictCompleteFailed"))
	} finally {
		completing.value = false
	}
}

const initialize = async () => {
	loading.value = true
	loadError.value = ""
	repositories.value = []
	file.value = null
	selectedRepositoryId.value = ""
	selectedPath.value = ""
	try {
		await loadRepositories()
		const firstRepository = repositories.value[0]
		if (firstRepository?.files[0]) {
			await nextTick()
			await loadFile(firstRepository.id, firstRepository.files[0])
		}
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.conflictResolverLoadFailed")
	} finally {
		loading.value = false
	}
}

const close = () => {
	if (!isDirty.value) {
		emit("update:show", false)
		return
	}
	dialog.warning({
		title: t("code.conflictResolverTitle"),
		content: t("code.conflictUnsavedChanges"),
		positiveText: t("code.conflictDiscard"),
		negativeText: t("code.conflictKeepEditing"),
		onPositiveClick: () => emit("update:show", false)
	})
}

watch(
	() => props.show,
	show => {
		if (show) void initialize()
	}
)
</script>

<template>
	<n-drawer
		:show="show"
		:width="mobile ? undefined : 'min(1280px, 96vw)'"
		:height="mobile ? '96dvh' : undefined"
		:placement="mobile ? 'bottom' : 'right'"
		:mask-closable="false"
		@update:show="value => !value && close()"
	>
		<n-drawer-content closable :native-scrollbar="false" @close="close">
			<template #header>
				<div class="flex min-w-0 items-center gap-3">
					<span>{{ t("code.conflictResolverTitle") }}</span>
					<n-tag size="small" :type="allResolved ? 'success' : 'warning'">
						{{ t("code.conflictRepositoryProgress", { resolved: resolvedFiles, total: totalFiles }) }}
					</n-tag>
				</div>
			</template>

			<n-spin :show="loading" class="h-full" content-class="h-full">
				<n-result
					v-if="loadError"
					status="error"
					:title="t('code.conflictResolverLoadFailed')"
					:description="loadError"
				>
					<template #footer>
						<n-button @click="initialize">{{ t("code.conflictResolverLoading") }}</n-button>
					</template>
				</n-result>
				<div
					v-else
					class="flex h-full min-h-0 overflow-hidden rounded-lg border border-slate-200"
					:class="mobile ? 'flex-col' : ''"
				>
					<aside
						class="flex shrink-0 flex-col overflow-y-auto bg-slate-50"
						:class="mobile ? 'max-h-48 w-full border-b border-slate-200' : 'w-72 border-r border-slate-200'"
					>
						<div
							v-for="repository in repositories"
							:key="repository.id"
							class="border-b border-slate-200 p-3"
						>
							<div class="mb-2 flex items-center justify-between gap-2">
								<span class="min-w-0 truncate text-sm font-medium" :title="repository.name">
									{{ repository.name }}
								</span>
								<span class="shrink-0 text-[11px] text-slate-400">
									{{
										t("code.conflictRepositoryProgress", {
											resolved: repository.resolved,
											total: repository.total
										})
									}}
								</span>
							</div>
							<div
								class="mb-2 truncate text-[11px] text-slate-400"
								:title="`${repository.branch} → ${repository.targetBranch}`"
							>
								{{ repository.branch }} → {{ repository.targetBranch }}
							</div>
							<button
								v-for="path in repository.files"
								:key="fileKey(repository.id, path)"
								type="button"
								class="mb-1 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs hover:bg-slate-200"
								:class="
									fileKey(repository.id, path) === fileKey(selectedRepositoryId, selectedPath)
										? 'bg-blue-50 text-blue-700'
										: 'text-slate-600'
								"
								:title="path"
								@click="selectFile(repository.id, path)"
							>
								<span
									class="h-2 w-2 shrink-0 rounded-full"
									:class="isUnresolved(repository, path) ? 'bg-amber-500' : 'bg-emerald-500'"
								/>
								<span class="min-w-0 flex-1 truncate">{{ path }}</span>
								<span class="shrink-0 text-[10px] text-slate-400">
									{{
										t(
											isUnresolved(repository, path)
												? "code.conflictUnresolved"
												: "code.conflictResolved"
										)
									}}
								</span>
							</button>
						</div>
					</aside>

					<main class="flex min-w-0 flex-1 flex-col bg-white">
						<n-spin :show="fileLoading" class="min-h-0 flex-1" content-class="h-full">
							<div v-if="file" class="flex h-full min-h-0 flex-col">
								<div
									class="flex shrink-0 gap-3 border-b border-slate-200 px-4 py-2"
									:class="mobile ? 'flex-col' : 'items-center justify-between'"
								>
									<span class="min-w-0 truncate text-sm text-slate-600" :title="file.path">
										{{ file.path }}
									</span>
									<div class="flex shrink-0 flex-wrap items-center gap-2">
										<n-button size="tiny" @click="chooseResolution('main')">
											{{ t("code.conflictUseMain") }}
										</n-button>
										<n-button size="tiny" @click="chooseResolution('task')">
											{{ t("code.conflictUseTask") }}
										</n-button>
										<n-button size="tiny" type="error" ghost @click="chooseResolution('delete')">
											{{ t("code.conflictDelete") }}
										</n-button>
									</div>
								</div>

								<div v-if="file.binary" class="flex min-h-0 flex-1 items-center justify-center p-8">
									<n-alert type="warning" :title="t('code.conflictResultVersion')">
										{{ t("code.conflictBinaryHint") }}
									</n-alert>
								</div>
								<n-tabs
									v-else-if="mobile"
									v-model:value="mobilePane"
									type="segment"
									class="min-h-0 flex-1 p-2"
									pane-class="h-full min-h-0"
								>
									<n-tab-pane name="main" :tab="t('code.conflictMainVersion')">
										<pre
											v-if="file.mainExists"
											class="h-full overflow-auto whitespace-pre p-3 font-mono text-xs"
											>{{ file.mainContent }}</pre
										>
										<n-empty v-else class="mt-16" :description="t('code.conflictVersionDeleted')" />
									</n-tab-pane>
									<n-tab-pane name="task" :tab="t('code.conflictTaskVersion')">
										<pre
											v-if="file.taskExists"
											class="h-full overflow-auto whitespace-pre p-3 font-mono text-xs"
											>{{ file.taskContent }}</pre
										>
										<n-empty v-else class="mt-16" :description="t('code.conflictVersionDeleted')" />
									</n-tab-pane>
									<n-tab-pane name="result" :tab="t('code.conflictResultVersion')">
										<FtEditor
											v-model="editableContent"
											:language="language"
											height="100%"
											:show-toolbar="false"
										/>
									</n-tab-pane>
								</n-tabs>
								<div v-else class="grid min-h-0 flex-1 grid-rows-2">
									<div class="grid min-h-0 grid-cols-2 border-b border-slate-200">
										<section class="flex min-h-0 flex-col border-r border-slate-200">
											<div class="shrink-0 px-3 py-1.5 text-xs font-medium text-slate-500">
												{{ t("code.conflictMainVersion") }}
											</div>
											<FtEditor
												v-if="file.mainExists"
												:model-value="file.mainContent || ''"
												:language="language"
												height="100%"
												:readonly="true"
												:show-toolbar="false"
											/>
											<n-empty
												v-else
												class="my-auto"
												:description="t('code.conflictVersionDeleted')"
											/>
										</section>
										<section class="flex min-h-0 flex-col">
											<div class="shrink-0 px-3 py-1.5 text-xs font-medium text-slate-500">
												{{ t("code.conflictTaskVersion") }}
											</div>
											<FtEditor
												v-if="file.taskExists"
												:model-value="file.taskContent || ''"
												:language="language"
												height="100%"
												:readonly="true"
												:show-toolbar="false"
											/>
											<n-empty
												v-else
												class="my-auto"
												:description="t('code.conflictVersionDeleted')"
											/>
										</section>
									</div>
									<section class="flex min-h-0 flex-col">
										<div class="shrink-0 px-3 py-1.5 text-xs font-medium text-slate-500">
											{{ t("code.conflictResultVersion") }}
										</div>
										<FtEditor
											v-model="editableContent"
											:language="language"
											height="100%"
											:show-toolbar="false"
										/>
									</section>
								</div>
							</div>
							<n-empty v-else class="mt-32" :description="t('code.conflictSelectFile')" />
						</n-spin>

						<footer
							class="flex shrink-0 gap-3 border-t border-slate-200 px-4 py-3"
							:class="mobile ? 'flex-col' : 'items-center justify-between'"
						>
							<span class="text-xs text-slate-400">{{ t("code.conflictCompleteHint") }}</span>
							<div class="flex shrink-0 gap-2" :class="mobile ? 'w-full' : ''">
								<n-button
									:block="mobile"
									:loading="saving"
									:disabled="!file || !isDirty"
									@click="saveFile"
								>
									{{ t("code.conflictSave") }}
								</n-button>
								<n-button
									:block="mobile"
									type="primary"
									:loading="completing"
									:disabled="!allResolved || isDirty"
									@click="complete"
								>
									{{ t("code.conflictComplete") }}
								</n-button>
							</div>
						</footer>
					</main>
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>
</template>

<style scoped>
:deep(.n-drawer-body-content-wrapper),
:deep(.n-spin-container),
:deep(.n-spin-content) {
	height: 100%;
}
</style>
