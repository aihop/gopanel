<template>
	<n-modal
		:show="showModal"
		@update:show="$emit('update:show-modal', $event)"
		preset="dialog"
		style="width: min(760px, 94vw)"
		:title="editingProjectId ? t('code.editProject') : t('code.createProject')"
	>
		<div class="flex flex-col gap-4 mt-4">
			<n-input
				v-model:value="form.name"
				:placeholder="t('code.projectName')"
				placeholder-class="text-[var(--n-text-color-3)]"
			/>
			<ProjectQualityGateSettings
				v-model="form.requireQualityGate"
				:checks="form.qualityChecks"
				:source-dirs="form.sourceDirs"
				:repositories="repositoryOptions"
				@update:checks="form.qualityChecks = $event"
			/>
			<n-input-number
				v-model:value="form.monthlyTokenBudget"
				:min="0"
				:step="100000"
				style="width: 100%"
				:placeholder="t('code.monthlyTokenBudget')"
			>
				<template #prefix>
					{{ t("code.monthlyTokenBudget") }}
				</template>
			</n-input-number>
			<div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
				{{ t("code.monthlyTokenBudgetHint") }}
			</div>
			<n-select
				v-model:value="form.primaryRepository"
				:options="primaryRepositoryOptions"
				:loading="repositoriesLoading"
				clearable
				:placeholder="t('code.primaryRepositoryPlaceholder')"
			/>
			<div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
				{{ t("code.primaryRepositoryHint") }}
			</div>
			<div v-if="repositoryOptions.length > 1">
				<div class="mb-2 text-sm font-medium text-[var(--n-text-color)]">
					{{ t("code.includedRepositories") }}
				</div>
				<div class="flex flex-col gap-2 rounded-xl bg-[var(--n-color-embedded)] p-3">
					<n-checkbox
						v-for="option in repositoryOptions"
						:key="option.value"
						:checked="!form.excludedRepositories.includes(option.value)"
						:disabled="option.value === form.primaryRepository"
						@update:checked="checked => toggleRepositoryIncluded(option.value, checked)"
					>
						<span class="text-xs">{{ option.label }}</span>
					</n-checkbox>
				</div>
				<div class="mt-2 text-xs text-[var(--n-text-color-3)]">
					{{ t("code.includedRepositoriesHint") }}
				</div>
			</div>
			<n-input
				v-model:value="form.deliveryBranch"
				:placeholder="t('code.deliveryBranchPlaceholder')"
			>
				<template #prefix>
					{{ t("code.deliveryBranch") }}
				</template>
			</n-input>
			<div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
				{{ t("code.deliveryBranchHint") }}
			</div>
			<n-select
				v-model:value="form.deliveryMode"
				:options="deliveryModeOptions"
				:placeholder="t('code.deliveryMode')"
			/>
			<div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
				{{ form.deliveryMode === "branch" ? t("code.deliveryModeBranchHint") : t("code.deliveryModeDirectHint") }}
			</div>
			<ProjectGitCredentialSelect v-model="form.gitCredentialId" />
			<n-input
				v-model:value="form.desc"
				type="textarea"
				:placeholder="t('code.projectDesc')"
			/>
			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<div class="text-sm font-medium text-[var(--n-text-color)]">
						{{ t("code.projectDirectories") }}
					</div>
					<n-button
						type="primary"
						secondary
						size="small"
						@click="$emit('update:show-directory-picker', true)"
					>
						{{ t("code.browseDirectory") }}
					</n-button>
				</div>
				<div
					v-if="form.sourceDirs.length"
					class="flex flex-wrap gap-2 rounded-xl bg-[var(--n-color-embedded)] p-3"
				>
					<n-tag
						v-for="sourceDir in form.sourceDirs"
						:key="sourceDir"
						closable
						:title="sourceDir"
						@close="$emit('remove-source-dir', sourceDir)"
					>
						{{ sourceDir }}
					</n-tag>
				</div>
				<n-empty
					v-else
					size="small"
					:description="t('code.projectDirectoryRequired')"
				/>
				<div class="mt-2 text-xs text-[var(--n-text-color-3)]">
					{{ t("code.projectDirectoriesHint") }}
				</div>
			</div>
		</div>
		<template #action>
			<n-button :disabled="creatingProject" @click="$emit('update:show-modal', false)">
				{{ $t('commons.button.cancel') }}
			</n-button>
			<n-button type="primary" :loading="creatingProject" @click="$emit('submit')">
				{{ editingProjectId ? t("code.saveChanges") : $t('commons.button.confirm') }}
			</n-button>
		</template>
	</n-modal>
	<ProjectDirectoryPicker
		:show="showDirectoryPicker"
		:initial-path="form.workDir || defaultWorkDir"
		:root-path="directoryRoot"
		:selected-paths="form.sourceDirs"
		@select="$emit('source-dirs-selected', $event)"
	/>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import ProjectDirectoryPicker from "./ProjectDirectoryPicker.vue"
import ProjectGitCredentialSelect from "./ProjectGitCredentialSelect.vue"
import ProjectQualityGateSettings from "./ProjectQualityGateSettings.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

defineProps<{
	showModal: boolean
	editingProjectId: number | null
	form: {
		name: string
		desc: string
		workDir: string
		sourceDirs: string[]
		excludedRepositories: string[]
		primaryRepository: string
		deliveryBranch: string
		deliveryMode: "direct" | "branch"
		gitCredentialId: number
		requireQualityGate: boolean
		qualityChecks: any[]
		monthlyTokenBudget: number
	}
	repositoryOptions: Array<{ label: string; value: string }>
	repositoriesLoading: boolean
	creatingProject: boolean
	defaultWorkDir: string
	directoryRoot: string
	showDirectoryPicker: boolean
}>()

defineEmits<{
	"update:show-modal": [value: boolean]
	"update:show-directory-picker": [value: boolean]
	submit: []
	"remove-source-dir": [sourceDir: string]
	"source-dirs-selected": [sourceDirs: string[]]
}>()

const { t } = useI18n({ messages: codeProjectMessages })

const primaryRepositoryOptions = computed(() => [
	{ label: t("code.primaryRepositoryAuto"), value: "" },
	...props.repositoryOptions,
])

const deliveryModeOptions = computed(() => [
	{ label: t("code.deliveryModeDirect"), value: "direct" },
	{ label: t("code.deliveryModeBranch"), value: "branch" },
])

const toggleRepositoryIncluded = (path: string, included: boolean) => {
	if (!included && path === props.form.primaryRepository) return
}
</script>