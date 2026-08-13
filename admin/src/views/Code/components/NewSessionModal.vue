<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import {
	commitCodeProjectChanges,
	createCodeSession,
	getAIProviderAccounts,
	getCodeExecutors,
	getCodeWorktreeCapability
} from "@/api/modules/code"
import type {
	CodeApprovalPolicy,
	CodeExecutor,
	CodeExecutorConfig,
	CodeSession,
	CodeWorktreeCapability
} from "@/api/interface/code"
import type { AIProviderAccount } from "@/api/interface/aiAccounts"
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
const providerSource = ref<"account" | "manual">("account")
const providerAccounts = ref<AIProviderAccount[]>([])
const selectedProviderAccountId = ref<number | null>(null)
const providerConfig = ref<CodeExecutorConfig>({ baseUrl: "", apiKey: "", model: "" })
const isolated = ref(false)
const worktreeCapability = ref<CodeWorktreeCapability | null>(null)
const title = ref("")
const loading = ref(false)
const submitting = ref(false)
const loadError = ref("")
const providerAccountLoading = ref(false)
const providerAccountLoadError = ref("")
const submitError = ref("")

const aiExecutors = computed(() => executors.value.filter(executor => executor.id !== "terminal"))
const availableExecutors = computed(() => aiExecutors.value.filter(executor => executor.available))
const selectedExecutor = computed(() => executors.value.find(executor => executor.id === selectedExecutorId.value))
const supportsApproval = computed(() => (selectedExecutor.value?.approvalPolicies.length || 0) > 1)
const approvalPolicies = computed<CodeApprovalPolicy[]>(() =>
	selectedExecutor.value?.approvalPolicies.length ? selectedExecutor.value.approvalPolicies : ["full_auto"]
)
const providerFields = computed(() => selectedExecutor.value?.configSchema?.fields || [])
const showProviderConfig = computed(() => providerFields.value.length > 0)
const availableProviderAccounts = computed(() =>
	providerAccounts.value.filter(account => account.enabled && account.hasApiKey)
)
const providerAccountOptions = computed(() =>
	availableProviderAccounts.value.map(account => ({
		label: `${account.name} · ${account.model}`,
		value: account.id
	}))
)
const providerFieldLabel = (key: keyof CodeExecutorConfig) => t(`code.providerField_${key}`)
const providerFieldPlaceholder = (key: keyof CodeExecutorConfig) => t(`code.providerPlaceholder_${key}`)
const dirtyRepositories = computed(() => worktreeCapability.value?.dirtyRepositories || [])
const dirtyStrategy = ref<"commit" | "snapshot">("commit")
const commitMessage = ref("")
const committing = ref(false)
const dirtyRepositoriesBlocked = computed(
	() => isolated.value && dirtyRepositories.value.length > 0 && dirtyStrategy.value === "commit"
)

async function commitDirtyRepositories() {
	if (!props.projectId || !commitMessage.value.trim()) return
	committing.value = true
	try {
		const response = await commitCodeProjectChanges(props.projectId, commitMessage.value.trim())
		if (response.code !== 0) throw new Error(response.message)
		const failed = response.data.filter(item => item.status === "failed")
		if (failed.length) {
			message.error(failed[0].errorMessage || t("code.dirtyCommitFailed"))
			await loadWorktreeCapability()
			return
		}
		const committed = response.data.filter(item => item.status === "committed").length
		message.success(t("code.dirtyCommitted", { count: committed }))
		commitMessage.value = ""
		await loadWorktreeCapability()
	} catch (error) {
		message.error(error instanceof Error && error.message ? error.message : t("code.dirtyCommitFailed"))
	} finally {
		committing.value = false
	}
}
const isolationAvailable = computed(() => Boolean(worktreeCapability.value?.available))

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
		void 0
	}
}

const loadProviderAccounts = async () => {
	providerAccountLoading.value = true
	providerAccountLoadError.value = ""
	try {
		const response = await getAIProviderAccounts()
		providerAccounts.value = response.data || []
		if (!availableProviderAccounts.value.some(account => account.id === selectedProviderAccountId.value)) {
			selectedProviderAccountId.value = availableProviderAccounts.value[0]?.id || null
		}
	} catch (error) {
		providerAccounts.value = []
		providerAccountLoadError.value = error instanceof Error ? error.message : t("code.providerAccountLoadFailed")
	} finally {
		providerAccountLoading.value = false
	}
}

watch(
	() => props.show,
	show => {
		if (show) {
			title.value = ""
			submitError.value = ""
			approvalPolicy.value = "safe_auto"
			providerMode.value = "default"
			providerSource.value = "account"
			selectedProviderAccountId.value = null
			providerConfig.value = { baseUrl: "", apiKey: "", model: "" }
			dirtyStrategy.value = "commit"
			commitMessage.value = ""
			void Promise.all([loadExecutors(), loadWorktreeCapability(), loadProviderAccounts()])
		}
	}
)

watch(selectedExecutorId, value => {
	const executor = executors.value.find(item => item.id === value)
	providerMode.value = executor?.configured || executor?.id === "claude" ? "default" : "custom"
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
	if (dirtyRepositoriesBlocked.value) {
		message.warning(t("code.dirtyCommitRequired"))
		return
	}
	if (showProviderConfig.value && providerMode.value === "custom") {
		if (providerSource.value === "account" && !selectedProviderAccountId.value) {
			message.warning(t("code.providerAccountRequired"))
			return
		}
		if (providerSource.value === "manual") {
			const missingField = providerFields.value.find(
				field => field.required && !providerConfig.value[field.key].trim()
			)
			if (missingField) {
				message.warning(t("code.providerFieldRequired", { field: providerFieldLabel(missingField.key) }))
				return
			}
		}
	}
	submitting.value = true
	submitError.value = ""
	try {
		const sessionRequest = {
			title: title.value.trim(),
			workDir: "",
			projectId: props.projectId,
			executorId: selectedExecutorId.value,
			approvalPolicy: approvalPolicy.value,
			isolated: isolated.value,
			includeUncommitted: dirtyStrategy.value === "snapshot",
			providerAccountId:
				showProviderConfig.value && providerMode.value === "custom" && providerSource.value === "account"
					? selectedProviderAccountId.value || undefined
					: undefined,
			provider:
				showProviderConfig.value && providerMode.value === "custom" && providerSource.value === "manual"
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
		submitError.value = error instanceof Error ? error.message : t("code.sessionInitializationFailed")
		message.error(submitError.value)
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
				<n-alert v-if="submitError" type="error" :title="t('code.sessionInitializationFailed')">
					{{ submitError }}
				</n-alert>
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
								<div v-if="executor.available && !executor.configured" class="mt-2 text-xs text-amber-600">
									{{
										t(executor.id === "claude" ? "code.executorConnectionUndetected" : "code.executorNeedsProvider")
									}}
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
										<n-radio
											value="default"
											:disabled="!selectedExecutor?.configured && selectedExecutor?.id !== 'claude'"
										>
											{{ t("code.providerDefault") }}
										</n-radio>
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
								<n-radio-group v-if="providerMode === 'custom'" v-model:value="providerSource">
									<n-space>
										<n-radio value="account">
											{{ t("code.providerSavedAccount") }}
										</n-radio>
										<n-radio value="manual">{{ t("code.providerManual") }}</n-radio>
									</n-space>
								</n-radio-group>
								<div v-if="providerMode === 'custom' && providerSource === 'account'" class="space-y-2">
									<n-form-item :label="t('code.providerAccount')" :show-feedback="false">
										<n-select
											v-model:value="selectedProviderAccountId"
											:options="providerAccountOptions"
											:loading="providerAccountLoading"
											:placeholder="t('code.providerAccountPlaceholder')"
										/>
									</n-form-item>
									<n-alert v-if="providerAccountLoadError" type="error" :show-icon="false">
										<div class="flex items-center justify-between gap-3">
											<span>{{ providerAccountLoadError }}</span>
											<n-button size="tiny" @click="loadProviderAccounts">{{ t("code.retry") }}</n-button>
										</div>
									</n-alert>
									<n-empty
										v-else-if="!providerAccountLoading && !availableProviderAccounts.length"
										:description="t('code.providerAccountEmpty')"
										size="small"
									/>
								</div>
								<div
									v-if="providerMode === 'custom' && providerSource === 'manual'"
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
												? t(isolated ? "code.worktreeIsolationDesc" : "code.directWorkspaceDesc")
													: t(
															`code.worktreeReason_${worktreeCapability?.reason || "loading"}`
														)
											}}
										</div>
									</div>
									<n-switch v-model:value="isolated" :disabled="!isolationAvailable" />
								</div>
								<n-alert v-if="isolated && dirtyRepositories.length" type="warning" :show-icon="false">
									<div class="text-xs font-medium leading-5">
										{{ t("code.dirtyRepositoriesFound", { repositories: dirtyRepositories.join(", ") }) }}
									</div>
									<!-- 把两条路的后果都写出来。原先默认「复制进隔离区」且只说
									     「会自动安全复制」——「安全」只保证不破坏源目录，
									     不保证之后不出事，那份安心感是错的。 -->
									<n-radio-group v-model:value="dirtyStrategy" size="small" class="mt-2 flex flex-col gap-2">
										<n-radio value="commit">
											<div class="text-xs font-medium">{{ t("code.dirtyCommitFirst") }}</div>
											<div class="text-[11px] leading-5 opacity-70">{{ t("code.dirtyCommitFirstHint") }}</div>
										</n-radio>
										<n-radio value="snapshot">
											<div class="text-xs font-medium">{{ t("code.dirtySnapshot") }}</div>
											<div class="text-[11px] leading-5 opacity-70">{{ t("code.dirtySnapshotHint") }}</div>
										</n-radio>
									</n-radio-group>
									<div v-if="dirtyStrategy === 'commit'" class="mt-2 flex items-center gap-2">
										<n-input
											v-model:value="commitMessage"
											size="small"
											:placeholder="t('code.dirtyCommitMessage')"
											:disabled="committing"
										/>
										<n-button
											size="small"
											type="primary"
											:loading="committing"
											:disabled="!commitMessage.trim()"
											@click="commitDirtyRepositories"
										>
											{{ t("code.dirtyCommitAction") }}
										</n-button>
									</div>
								</n-alert>
								<n-alert v-if="!isolated && dirtyRepositories.length" type="warning" :show-icon="false">
									<div class="text-xs leading-5">
										{{ t("code.directWorkspaceDirtyWarning", { repositories: dirtyRepositories.join(", ") }) }}
									</div>
								</n-alert>
							</div>
						</n-form-item>
						<n-alert type="info" :show-icon="false">
							{{ t(isolated ? "code.sessionUsesIsolatedDirectory" : "code.sessionUsesDirectDirectory") }}
						</n-alert>
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
					:disabled="loading || committing || !!loadError || !selectedExecutorId || dirtyRepositoriesBlocked"
					@click="submit"
				>
					{{ t("code.createAndOpen") }}
				</n-button>
			</div>
		</template>
	</n-modal>
</template>
