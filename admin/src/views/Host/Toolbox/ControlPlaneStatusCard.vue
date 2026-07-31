<template>
	<div class="rounded-[28px] border border-blue-100/80 bg-base-100 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)] sm:p-8">
		<div class="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
			<div>
				<div class="text-lg font-semibold fg-base-100">{{ t("controlPlane.title") }}</div>
				<div class="mt-2 text-sm text-slate-500">{{ t("controlPlane.description") }}</div>
			</div>
			<n-button :loading="loading" secondary @click="emit('refresh')">
				{{ t("controlPlane.refresh") }}
			</n-button>
		</div>

		<div v-if="loading && !status" class="mt-6">
			<n-skeleton text :repeat="2" />
		</div>
		<n-alert v-else-if="error && !status" class="mt-6" type="error" :title="t('controlPlane.loadFailed')">
			<div>{{ error }}</div>
			<n-button class="mt-3" size="small" @click="emit('refresh')">{{ t("controlPlane.retry") }}</n-button>
		</n-alert>
		<n-empty v-else-if="!status" class="mt-6" :description="t('controlPlane.empty')" />
		<template v-else>
			<n-alert v-if="error" class="mt-6" type="error" :show-icon="true">{{ error }}</n-alert>
			<div class="mt-6 grid gap-4 md:grid-cols-2">
				<div v-for="component in components" :key="component.name" class="rounded-2xl border border-slate-100 bg-slate-50/75 p-5">
					<div class="flex items-center justify-between gap-3">
						<div class="font-semibold fg-base-100">{{ component.name }}</div>
						<n-tag :type="component.healthy ? 'success' : 'warning'" round>
							{{ stateText(component.state) }}
						</n-tag>
					</div>
					<div class="mt-3 truncate text-xs text-slate-500" :title="component.socketPath">
						{{ t("controlPlane.socket") }} · {{ component.socketPath || "-" }}
					</div>
					<div v-if="component.version" class="mt-2 text-xs text-slate-500">
						{{ t("controlPlane.version") }} · {{ component.version }}
					</div>
				</div>
			</div>

			<div class="mt-5 flex flex-col gap-3 rounded-2xl bg-slate-50/75 p-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="text-sm font-medium" :class="status.healthy ? 'text-green-600' : 'text-orange-600'">
					{{ summaryText }}
				</div>
				<div class="flex flex-wrap gap-2">
					<n-button secondary @click="emit('details')">{{ t("controlPlane.diagnostics") }}</n-button>
					<n-button v-if="status.autoRepairable" type="primary" :loading="repairing" @click="emit('repair')">
						{{ t("controlPlane.autoRepair") }}
					</n-button>
				</div>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { ControlPlaneStatus } from "@/api/modules/agent"
import { controlPlaneMessages } from "@/i18n/locales/controlPlane"

const props = defineProps<{
	status: ControlPlaneStatus | null
	loading: boolean
	error: string
	repairing: boolean
}>()
const emit = defineEmits<{
	(e: "refresh"): void
	(e: "details"): void
	(e: "repair"): void
}>()
const { t } = useI18n({ messages: controlPlaneMessages })
const components = computed(() => props.status ? [props.status.gpc, props.status.agent] : [])
const stateText = (state: string) => t(`controlPlane.state_${state}`)
const summaryText = computed(() => {
	if (props.status?.healthy) return t("controlPlane.healthySummary")
	if (props.status?.autoRepairable) return t("controlPlane.repairableSummary")
	return t("controlPlane.gpcSummary")
})
</script>
