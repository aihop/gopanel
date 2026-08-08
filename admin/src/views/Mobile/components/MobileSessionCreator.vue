<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { createMobileSession, getMobileExecutors, getMobileProjects, getMobileWorktreeCapability } from "@/api/modules/mobile"
import type { AIProject, CodeApprovalPolicy, CodeExecutor, CodeSession, CodeWorktreeCapability } from "@/api/interface/code"
import { mobileMessages } from "@/i18n/locales/mobile"

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
	(event: "update:show", value: boolean): void
	(event: "created", session: CodeSession): void
}>()

const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const projects = ref<AIProject[]>([])
const executors = ref<CodeExecutor[]>([])
const projectId = ref<number | null>(null)
const executorId = ref("")
const title = ref("")
const approvalPolicy = ref<CodeApprovalPolicy>("safe_auto")
const loading = ref(false)
const submitting = ref(false)
const loadError = ref("")
const worktreeCapability = ref<CodeWorktreeCapability | null>(null)
const capabilityLoading = ref(false)

const projectOptions = computed(() => projects.value.map(project => ({ label: project.name, value: project.id })))
const availableExecutors = computed(() => executors.value.filter(executor => executor.available && executor.id !== "terminal"))
const executorOptions = computed(() => availableExecutors.value.map(executor => ({
	label: `${executor.name}${executor.version ? ` · ${executor.version}` : ""}`,
	value: executor.id
})))
const selectedExecutor = computed(() => executors.value.find(executor => executor.id === executorId.value))
const approvalPolicies = computed<CodeApprovalPolicy[]>(() =>
	selectedExecutor.value?.approvalPolicies.length ? selectedExecutor.value.approvalPolicies : ["full_auto"]
)
const supportsApproval = computed(() => approvalPolicies.value.length > 1)
const dirtyRepositories = computed(() => worktreeCapability.value?.dirtyRepositories || [])
const canCreate = computed(() => Boolean(projectId.value && executorId.value && worktreeCapability.value?.available))

async function loadWorktreeCapability() {
	worktreeCapability.value = null
	if (!projectId.value) return
	capabilityLoading.value = true
	try {
		worktreeCapability.value = await getMobileWorktreeCapability(projectId.value)
	} catch (error) {
	} finally {
		capabilityLoading.value = false
	}
}

async function loadOptions() {
	loading.value = true
	loadError.value = ""
	try {
		const [projectResult, executorResult] = await Promise.all([getMobileProjects(), getMobileExecutors()])
		projects.value = projectResult.items
		executors.value = executorResult || []
		projectId.value = null
		projectId.value = projects.value[0]?.id || null
		executorId.value = availableExecutors.value.find(item => item.id === "codex")?.id || availableExecutors.value[0]?.id || ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.loadFailed")
	} finally {
		loading.value = false
	}
}

async function submit() {
	if (!canCreate.value || submitting.value) return
	submitting.value = true
	try {
		const session = await createMobileSession(
			{
				title: title.value.trim(),
				projectId: projectId.value,
				executorId: executorId.value,
				approvalPolicy: approvalPolicy.value
			},
			{
				initializationFailed: t("mobile.sessionInitializationFailed"),
				initializationTimedOut: t("mobile.sessionInitializationTimedOut")
			}
		)
		message.success(t("mobile.sessionCreated"))
		emit("created", session)
		emit("update:show", false)
	} catch (error) {
	} finally {
		submitting.value = false
	}
}

watch(
	() => props.show,
	show => {
		if (!show) return
		title.value = ""
		approvalPolicy.value = "safe_auto"
		void loadOptions()
	}
)

watch(executorId, value => {
	if (value && !approvalPolicies.value.includes(approvalPolicy.value)) {
		approvalPolicy.value = approvalPolicies.value.includes("safe_auto")
			? "safe_auto"
			: approvalPolicies.value[0] || "full_auto"
	}
})

watch(projectId, () => void loadWorktreeCapability())
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="min(640px, 88dvh)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('mobile.newSession')" closable>
			<n-spin :show="loading">
				<div class="mx-auto max-w-xl space-y-4">
					<n-alert v-if="loadError" type="error" :title="t('mobile.loadFailed')">
						<div class="flex items-center justify-between gap-3">
							<span>{{ loadError }}</span>
							<n-button size="small" @click="loadOptions">{{ t("mobile.retry") }}</n-button>
						</div>
					</n-alert>
					<n-empty v-else-if="!loading && projects.length === 0" :description="t('mobile.noProjects')" />
					<template v-else>
						<n-form label-placement="top">
							<n-form-item :label="t('mobile.project')">
								<n-select v-model:value="projectId" :options="projectOptions" :loading="capabilityLoading" />
							</n-form-item>
							<n-alert v-if="dirtyRepositories.length" type="warning" :show-icon="false">
								{{ t("mobile.worktreeDirtyDesktopHint", { repositories: dirtyRepositories.join(", ") }) }}
							</n-alert>
							<n-alert v-else-if="worktreeCapability && !worktreeCapability.available" type="error" :show-icon="false">
								{{ t("mobile.worktreeUnavailable") }}
							</n-alert>
							<n-form-item :label="t('mobile.executor')">
								<n-select v-model:value="executorId" :options="executorOptions" :placeholder="t('mobile.noExecutor')" />
							</n-form-item>
							<n-form-item :label="t('mobile.sessionName')">
								<n-input v-model:value="title" :placeholder="t('mobile.sessionNamePlaceholder')" />
							</n-form-item>
							<n-form-item :label="t('mobile.approvalPolicy')">
								<n-radio-group v-model:value="approvalPolicy">
									<n-space vertical>
										<n-radio v-if="approvalPolicies.includes('manual')" value="manual">{{ t("mobile.approvalManual") }}</n-radio>
										<n-radio v-if="approvalPolicies.includes('safe_auto')" value="safe_auto">{{ t("mobile.approvalSafe") }}</n-radio>
										<n-radio v-if="approvalPolicies.includes('full_auto')" value="full_auto">{{ t("mobile.approvalFull") }}</n-radio>
									</n-space>
								</n-radio-group>
							</n-form-item>
							<n-alert v-if="!supportsApproval" type="warning" :show-icon="false">
								{{ t("mobile.executorFullAutoOnly") }}
							</n-alert>
						</n-form>
					</template>
				</div>
			</n-spin>
			<template #footer>
				<n-button type="primary" block size="large" :loading="submitting" :disabled="loading || capabilityLoading || !!loadError || !canCreate" @click="submit">
					{{ t("mobile.createAndOpen") }}
				</n-button>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>
