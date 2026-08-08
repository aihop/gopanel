<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createCodeSession, getCodeExecutors, getCodeWorktreeCapability } from "@/api/modules/code"
import type {
	CodeApprovalPolicy,
	CodeExecutor,
	CodeExecutorConfig,
	CodeSession,
	CodeWorktreeCapability
} from "@/api/interface/code"
import { newCodeSessionMessages } from "../newCodeSessionMessages"

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
const providerMode = ref<"default" | "custom">("default")
const providerConfig = ref<CodeExecutorConfig>({ baseUrl: "", apiKey: "", model: "" })
const isolated = ref(false)
const worktreeCapability = ref<CodeWorktreeCapability | null>(null)
const title = ref("")
const loading = ref(false)
const submitting = ref(false)
const loadError = ref("")

const aiExecutors = computed(() => executors.value.filter(executor => executor.id !== "terminal"))
const availableExecutors = computed(() => aiExecutors.value.filter(executor => executor.available))
const selectedExecutor = computed(() => executors.value.find(executor => executor.id === selectedExecutorId.value))
const supportsApproval = computed(() => (selectedExecutor.value?.approvalPolicies.length || 0) > 1)
const approvalPolicies = computed<CodeApprovalPolicy[]>(() =>
	selectedExecutor.value?.approvalPolicies.length ? selectedExecutor.value.approvalPolicies : ["full_auto"]
)
const providerFields = computed(() => selectedExecutor.value?.configSchema?.fields || [])
const showProviderConfig = computed(() => providerFields.value.length > 0)
const providerFieldLabel = (key: keyof CodeExecutorConfig) => t(`code.providerField_${key}`)
const providerFieldPlaceholder = (key: keyof CodeExecutorConfig) => t(`code.providerPlaceholder_${key}`)
const dirtyRepositories = computed(() => worktreeCapability.value?.dirtyRepositories || [])

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
		// 错误提示由请求拦截器统一处理
	}
}

watch(
	() => props.show,
	show => {
		if (show) {
			title.value = ""
			approvalPolicy.value = "safe_auto"
			providerMode.value = "default"
			providerConfig.value = { baseUrl: "", apiKey: "", model: "" }
			void Promise.all([loadExecutors(), loadWorktreeCapability()])
		}
	}
)

watch(selectedExecutorId, value => {
	if (value && !approvalPolicies.value.includes(approvalPolicy.value)) {
		approvalPolicy.value = approvalPolicies.value.includes("safe_auto")
			? "safe_auto"
			: approvalPolicies.value[0] || "full_auto"
	}
})

const close = () => emit("update:show", false)

const submit = async () => {
	if (!selectedExecutorId.value) {
		message.warning(t("code.selectExecutorRequired"))
		return
	}
	if (showProviderConfig.value && providerMode.value === "custom") {
		const missingField = providerFields.value.find(
			field => field.required && !providerConfig.value[field.key].trim()
		)
		if (missingField) {
			message.warning(t("code.providerFieldRequired", { field: providerFieldLabel(missingField.key) }))
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
			includeUncommitted: true,
			provider:
				showProviderConfig.value && providerMode.value === "custom"
					? (Object.fromEntries(
							providerFields.value.map(field => [field.key, providerConfig.value[field.key].trim()])
						) as unknown as CodeExecutorConfig)
					: undefined
		}
		const response = await createCodeSession(sessionRequest, {
			initializationFailed: t("code.sessionInitializationFailed"),
			initializationTimedOut: t("code.sessionInitializationTimedOut")
		})
		emit("created", response.data)
		close()
		message.success(t("code.sessionCreated"))
	} catch (error) {
		// 错误提示由请求拦截器统一处理
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
		:title="t('code.newAiTask')"
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

				<n-empty v-else-if="!loading && aiExecutors.length === 0" :description="t('code.noExecutors')" />

				<template v-else>
					<div>
						<div class="mb-3 text-sm font-semibold text-[var(--n-text-color)]">
							{{ t("code.chooseExecutor") }}
						</div>
						<div class="grid gap-3 sm:grid-cols-2">
							<button
								v-for="executor in aiExecutors"
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
						<n-form-item v-if="showProviderConfig" :label="t('code.provider')">
							<div class="w-full space-y-3">
								<n-radio-group v-model:value="providerMode">
									<n-space>
										<n-radio value="default">{{ t("code.providerDefault") }}</n-radio>
										<n-radio value="custom">{{ t("code.providerCustom") }}</n-radio>
									</n-space>
								</n-radio-group>
								<n-alert type="info" :show-icon="false">
									{{
										providerMode === "default"
											? t("code.providerDefaultHint", { executor: selectedExecutor?.name || "" })
											: t("code.providerCustomHint")
									}}
								</n-alert>
								<div
									v-if="providerMode === 'custom'"
									class="grid gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 sm:grid-cols-2"
								>
									<n-form-item
										v-for="field in providerFields"
										:key="field.key"
										:class="field.key === 'apiKey' ? 'sm:col-span-2' : ''"
										:label="providerFieldLabel(field.key)"
										:show-feedback="false"
									>
										<n-input
											v-model:value="providerConfig[field.key]"
											:type="field.type === 'password' ? 'password' : 'text'"
											:show-password-on="field.type === 'password' ? 'click' : undefined"
											:placeholder="providerFieldPlaceholder(field.key)"
										/>
									</n-form-item>
								</div>
							</div>
						</n-form-item>
						<n-form-item :label="t('code.approvalPolicy')">
							<div class="grid w-full gap-3 sm:grid-cols-3">
								<button
									v-for="policy in approvalPolicies"
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
						<n-alert v-if="!supportsApproval" type="warning" :show-icon="false">
							{{ t("code.executorFullAutoOnly") }}
						</n-alert>
						<n-alert v-if="approvalPolicy === 'full_auto'" type="warning">
							{{ t("code.fullAutoWarning") }}
						</n-alert>
						<n-form-item :label="t('code.worktreeIsolation')">
							<div class="w-full space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-3">
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
									<n-switch :value="isolated" disabled />
								</div>
								<n-alert v-if="dirtyRepositories.length" type="warning" :show-icon="false">
									<div class="text-xs leading-5">
										{{ t("code.worktreeDirtyRepositories", { repositories: dirtyRepositories.join(", ") }) }}
									</div>
								</n-alert>
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
