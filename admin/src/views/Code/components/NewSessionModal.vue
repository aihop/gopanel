<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createCodeSession, getCodeExecutors, getCodeWorktreeCapability } from "@/api/modules/code"
import type { CodeApprovalPolicy, CodeExecutor, CodeSession, CodeWorktreeCapability } from "@/api/interface/code"
import { newCodeSessionMessages } from "../newCodeSessionMessages"

type CodexWireAPI = "responses"

const props = defineProps<{
	show: boolean
	projectId: number
}>()

const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "created", session: CodeSession): void
}>()

const { t } = useI18n({ messages: newCodeSessionMessages })
const message = useMessage()
const executors = ref<CodeExecutor[]>([])
const selectedExecutorId = ref("")
const approvalPolicy = ref<CodeApprovalPolicy>("safe_auto")
const codexProviderMode = ref<"default" | "custom">("default")
const codexBaseUrl = ref("")
const codexApiKey = ref("")
const codexWireApi = ref<CodexWireAPI>("responses")
const isolated = ref(false)
const worktreeCapability = ref<CodeWorktreeCapability | null>(null)
const title = ref("")
const loading = ref(false)
const submitting = ref(false)
const loadError = ref("")

const availableExecutors = computed(() => executors.value.filter(executor => executor.available))
const showCodexProvider = computed(() => selectedExecutorId.value === "codex")
const codexWireApiOptions = computed(() => [
	{ label: t("code.codexWireApiResponses"), value: "responses" },
	{ label: t("code.codexWireApiChatUnsupported"), value: "chat", disabled: true }
])

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

const loadWorktreeCapability = async () => {
	worktreeCapability.value = null
	isolated.value = false
	try {
		const response = await getCodeWorktreeCapability(props.projectId)
		worktreeCapability.value = response.data
		isolated.value = response.data.available
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("code.worktreeCapabilityFailed"))
	}
}

watch(
	() => props.show,
	show => {
		if (show) {
			title.value = ""
			approvalPolicy.value = "safe_auto"
			codexProviderMode.value = "default"
			codexBaseUrl.value = ""
			codexApiKey.value = ""
			codexWireApi.value = "responses"
			void Promise.all([loadExecutors(), loadWorktreeCapability()])
		}
	}
)

const close = () => emit("update:show", false)

const submit = async () => {
	if (!selectedExecutorId.value) {
		message.warning(t("code.selectExecutorRequired"))
		return
	}
	if (showCodexProvider.value && codexProviderMode.value === "custom") {
		if (!codexBaseUrl.value.trim()) {
			message.warning(t("code.codexBaseUrlRequired"))
			return
		}
		if (!codexApiKey.value.trim()) {
			message.warning(t("code.codexApiKeyRequired"))
			return
		}
	}
	submitting.value = true
	try {
		const sessionRequest = {
			title: title.value.trim(),
			workDir: "",
			projectId: props.projectId,
			executorId: selectedExecutorId.value,
			approvalPolicy: approvalPolicy.value,
			isolated: isolated.value,
			codexProvider:
				showCodexProvider.value && codexProviderMode.value === "custom"
					? {
							baseUrl: codexBaseUrl.value.trim(),
							apiKey: codexApiKey.value.trim(),
							wireApi: codexWireApi.value
						}
					: undefined
		}
		const response = await createCodeSession(sessionRequest)
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
						<n-form-item v-if="showCodexProvider" :label="t('code.codexProvider')">
							<div class="w-full space-y-3">
								<n-radio-group v-model:value="codexProviderMode">
									<n-space>
										<n-radio value="default">{{ t("code.codexProviderDefault") }}</n-radio>
										<n-radio value="custom">{{ t("code.codexProviderCustom") }}</n-radio>
									</n-space>
								</n-radio-group>
								<n-alert type="info" :show-icon="false">
									{{
										codexProviderMode === "default"
											? t("code.codexProviderDefaultHint")
											: t("code.codexProviderCustomHint")
									}}
								</n-alert>
								<div
									v-if="codexProviderMode === 'custom'"
									class="grid gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 sm:grid-cols-2"
								>
									<n-form-item :label="t('code.codexBaseUrl')" :show-feedback="false">
										<n-input
											v-model:value="codexBaseUrl"
											:placeholder="t('code.codexBaseUrlPlaceholder')"
										/>
									</n-form-item>
									<n-form-item :label="t('code.codexWireApi')" :show-feedback="false">
										<n-select v-model:value="codexWireApi" :options="codexWireApiOptions" />
									</n-form-item>
									<n-form-item
										class="sm:col-span-2"
										:label="t('code.codexApiKey')"
										:show-feedback="false"
									>
										<n-input
											v-model:value="codexApiKey"
											type="password"
											show-password-on="click"
											:placeholder="t('code.codexApiKeyPlaceholder')"
										/>
									</n-form-item>
								</div>
							</div>
						</n-form-item>
						<n-form-item :label="t('code.approvalPolicy')">
							<div class="grid w-full gap-3 sm:grid-cols-3">
								<button
									v-for="policy in ['manual', 'safe_auto', 'full_auto'] as CodeApprovalPolicy[]"
									:key="policy"
									type="button"
									class="rounded-xl border p-3 text-left transition-all"
									:class="
										approvalPolicy === policy
											? 'border-blue-500 bg-blue-50 shadow-sm'
											: 'border-slate-200 bg-white hover:border-blue-200'
									"
									@click="approvalPolicy = policy"
								>
									<div class="text-sm font-semibold text-slate-800">
										{{ t(`code.approvalPolicy_${policy}`) }}
									</div>
									<div class="mt-1 text-xs leading-5 text-slate-500">
										{{ t(`code.approvalPolicyDesc_${policy}`) }}
									</div>
								</button>
							</div>
						</n-form-item>
						<n-alert v-if="approvalPolicy === 'full_auto'" type="warning">
							{{ t("code.fullAutoWarning") }}
						</n-alert>
						<n-form-item :label="t('code.worktreeIsolation')">
							<div class="w-full rounded-xl border border-slate-200 bg-slate-50 p-3">
								<div class="flex items-center justify-between gap-4">
									<div>
										<div class="text-sm font-semibold text-slate-800">
											{{ t("code.worktreeIsolationTitle") }}
										</div>
										<div class="mt-1 text-xs leading-5 text-slate-500">
											{{
												worktreeCapability?.available
													? t("code.worktreeIsolationDesc")
													: t(
															`code.worktreeReason_${worktreeCapability?.reason || "loading"}`
														)
											}}
										</div>
									</div>
									<n-switch v-model:value="isolated" :disabled="!worktreeCapability?.available" />
								</div>
							</div>
						</n-form-item>
						<n-alert type="info" :show-icon="false">{{ t("code.sessionUsesProjectDirectory") }}</n-alert>
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
