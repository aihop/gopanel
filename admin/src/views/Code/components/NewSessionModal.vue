<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createCodeSession, getCodeExecutors } from "@/api/modules/code"
import type { CodeExecutor, CodeSession } from "@/api/interface/code"

const props = defineProps<{
	show: boolean
	projectId: number
}>()

const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "created", session: CodeSession): void
}>()

const { t } = useI18n()
const message = useMessage()
const executors = ref<CodeExecutor[]>([])
const selectedExecutorId = ref("")
const title = ref("")
const workDir = ref("")
const loading = ref(false)
const submitting = ref(false)
const loadError = ref("")

const availableExecutors = computed(() => executors.value.filter(executor => executor.available))

const loadExecutors = async () => {
	loading.value = true
	loadError.value = ""
	try {
		const response = await getCodeExecutors()
		executors.value = response.data || []
		const preferredExecutor =
			executors.value.find(executor => executor.id === "codex" && executor.available) ||
			availableExecutors.value[0]
		selectedExecutorId.value = preferredExecutor?.id || ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.executorLoadFailed")
		message.error(t("code.executorLoadFailed"))
	} finally {
		loading.value = false
	}
}

watch(
	() => props.show,
	show => {
		if (show) {
			title.value = ""
			workDir.value = ""
			void loadExecutors()
		}
	}
)

const close = () => emit("update:show", false)

const submit = async () => {
	if (!selectedExecutorId.value) {
		message.warning(t("code.selectExecutorRequired"))
		return
	}
	submitting.value = true
	try {
		const response = await createCodeSession({
			title: title.value.trim(),
			workDir: workDir.value.trim(),
			projectId: props.projectId,
			executorId: selectedExecutorId.value
		})
		emit("created", response.data)
		close()
		message.success(t("code.sessionCreated"))
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.sessionCreateFailed"))
	} finally {
		submitting.value = false
	}
}
</script>

<template>
	<n-modal
		:show="show"
		preset="card"
		style="width: 720px"
		:title="t('code.newSession')"
		:mask-closable="!submitting"
		@update:show="emit('update:show', $event)"
	>
		<n-spin :show="loading">
			<div class="space-y-5">
				<n-alert v-if="loadError" type="error" :title="t('code.executorLoadFailed')">
					<div class="flex items-center justify-between gap-4">
						<span>{{ loadError }}</span>
						<n-button size="small" @click="loadExecutors">{{ t("code.retry") }}</n-button>
					</div>
				</n-alert>

				<n-empty v-else-if="!loading && executors.length === 0" :description="t('code.noExecutors')" />

				<template v-else>
					<div>
						<div class="mb-3 text-sm font-semibold text-[var(--n-text-color)]">
							{{ t("code.chooseExecutor") }}
						</div>
						<div class="grid gap-3 sm:grid-cols-2">
							<button
								v-for="executor in executors"
								:key="executor.id"
								type="button"
								class="rounded-2xl border p-4 text-left transition-all"
								:class="[
									selectedExecutorId === executor.id
										? 'border-blue-500 bg-blue-50 shadow-sm'
										: 'border-slate-200 bg-white hover:border-blue-200',
									!executor.available ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'
								]"
								:disabled="!executor.available"
								@click="selectedExecutorId = executor.id"
							>
								<div class="flex items-start justify-between gap-3">
									<div class="font-semibold text-slate-800">{{ executor.name }}</div>
									<n-tag
										size="small"
										:type="executor.available ? 'success' : 'default'"
										:bordered="false"
									>
										{{ executor.available ? t("code.available") : t("code.unavailable") }}
									</n-tag>
								</div>
								<div class="mt-2 text-xs leading-5 text-slate-500">
									{{ t(`code.executorDesc_${executor.id}`) }}
								</div>
								<div v-if="executor.version" class="mt-2 truncate text-xs text-slate-400">
									{{ executor.version }}
								</div>
								<div v-if="!executor.available" class="mt-2 text-xs text-amber-600">
									{{ t(`code.executorReason_${executor.reasonCode || "unavailable"}`) }}
								</div>
							</button>
						</div>
					</div>

					<n-form label-placement="top">
						<n-form-item :label="t('code.sessionTitle')">
							<n-input v-model:value="title" :placeholder="t('code.sessionTitlePlaceholder')" />
						</n-form-item>
						<n-form-item :label="t('code.workDir')">
							<n-input v-model:value="workDir" :placeholder="t('code.workDirPlaceholder')" />
						</n-form-item>
					</n-form>
				</template>
			</div>
		</n-spin>

		<template #footer>
			<div class="flex justify-end gap-3">
				<n-button :disabled="submitting" @click="close">{{ t("code.cancel") }}</n-button>
				<n-button
					type="primary"
					:loading="submitting"
					:disabled="loading || !!loadError || !selectedExecutorId"
					@click="submit"
				>
					{{ t("code.createAndOpen") }}
				</n-button>
			</div>
		</template>
	</n-modal>
</template>
