<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { preflightCodeProjectQualityChecks } from "@/api/modules/code"
import type {
	CodeProjectQualityCheck,
	CodeQualityKind,
	CodeQualityPreflight
} from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const props = defineProps<{
	modelValue: boolean
	checks: CodeProjectQualityCheck[]
	sourceDirs: string[]
	repositories: Array<{ label: string; value: string }>
}>()
const emit = defineEmits<{
	(event: "update:modelValue", value: boolean): void
	(event: "update:checks", value: CodeProjectQualityCheck[]): void
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const preflighting = ref(false)
const preflightError = ref("")
const preflight = ref<CodeQualityPreflight | null>(null)

const kindOptions = computed(() =>
	(["test", "lint", "typecheck", "build"] as CodeQualityKind[]).map(value => ({
		label: t(`code.qualityKind_${value}`),
		value
	}))
)

function updateCheck(index: number, patch: Partial<CodeProjectQualityCheck>) {
	const checks = props.checks.map((check, current) => current === index ? { ...check, ...patch } : check)
	emit("update:checks", checks)
}

function addCheck() {
	emit("update:checks", [
		...props.checks,
		{
			name: t("code.qualityCustomDefault"),
			kind: "test",
			repository: props.repositories[0]?.value || "",
			workDir: ".",
			command: ""
		}
	])
}

function removeCheck(index: number) {
	emit("update:checks", props.checks.filter((_, current) => current !== index))
}

async function runPreflight() {
	if (!props.sourceDirs.length || preflighting.value) return
	preflighting.value = true
	preflightError.value = ""
	try {
		preflight.value = (await preflightCodeProjectQualityChecks(props.sourceDirs, props.checks)).data
	} catch (error) {
		preflight.value = null
		preflightError.value = error instanceof Error ? error.message : t("code.qualityPreflightFailed")
	} finally {
		preflighting.value = false
	}
}

watch(
	() => [props.sourceDirs.join("\u0000"), JSON.stringify(props.checks)],
	() => {
		preflight.value = null
		preflightError.value = ""
	}
)
</script>

<template>
	<section class="rounded-xl bg-[var(--n-color-embedded)] p-3">
		<div class="flex items-start justify-between gap-3">
			<div>
				<div class="text-sm font-medium">{{ t("code.requireQualityGate") }}</div>
				<div class="mt-1 text-xs text-[var(--n-text-color-3)]">{{ t("code.requireQualityGateHint") }}</div>
			</div>
			<n-switch :value="modelValue" @update:value="emit('update:modelValue', $event)" />
		</div>

		<div v-if="modelValue" class="mt-4 space-y-3">
			<n-alert type="info" :show-icon="false">{{ t("code.qualityAutoDetectHint") }}</n-alert>
			<div
				v-for="(check, index) in checks"
				:key="index"
				class="space-y-2 rounded-xl border border-[var(--n-border-color)] bg-[var(--n-color)] p-3"
			>
				<div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
					<n-input
						:value="check.name"
						:placeholder="t('code.qualityCheckName')"
						@update:value="updateCheck(index, { name: $event })"
					/>
					<n-select
						:value="check.kind"
						:options="kindOptions"
						@update:value="updateCheck(index, { kind: $event })"
					/>
					<n-select
						:value="check.repository"
						:options="repositories"
						:placeholder="t('code.qualityRepository')"
						@update:value="updateCheck(index, { repository: $event })"
					/>
					<n-input
						:value="check.workDir"
						:placeholder="t('code.qualityWorkDir')"
						@update:value="updateCheck(index, { workDir: $event })"
					/>
				</div>
				<div class="flex gap-2">
					<n-input
						:value="check.command"
						class="min-w-0 flex-1 font-mono"
						:placeholder="t('code.qualityCommand')"
						@update:value="updateCheck(index, { command: $event })"
					/>
					<n-button quaternary type="error" @click="removeCheck(index)">{{ t("code.qualityRemove") }}</n-button>
				</div>
			</div>
			<div class="flex flex-wrap gap-2">
				<n-button secondary @click="addCheck">{{ t("code.qualityAdd") }}</n-button>
				<n-button
					type="primary"
					secondary
					:loading="preflighting"
					:disabled="!sourceDirs.length"
					@click="runPreflight"
				>
					{{ t("code.qualityPreflight") }}
				</n-button>
			</div>
			<n-alert v-if="preflightError" type="error" :show-icon="false">{{ preflightError }}</n-alert>
			<n-alert
				v-else-if="preflight"
				:type="preflight.ready ? 'success' : 'warning'"
				:show-icon="false"
			>
				<div>{{ t(preflight.ready ? "code.qualityPreflightReady" : "code.qualityPreflightBlocked") }}</div>
				<div v-for="item in preflight.items" :key="item.id" class="mt-2 flex items-start gap-2 text-xs">
					<n-tag size="small" :type="item.available ? 'success' : 'error'" :bordered="false">
						{{ item.available ? t("code.qualityAvailable") : t("code.qualityUnavailable") }}
					</n-tag>
					<span class="min-w-0 break-all font-mono">{{ item.workDir }} · {{ item.command }}</span>
				</div>
				<div v-if="!preflight.items.length" class="mt-2 text-xs">{{ t("code.qualityNoChecks") }}</div>
			</n-alert>
		</div>
	</section>
</template>
