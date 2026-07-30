<template>
	<n-modal
		:show="show"
		preset="card"
		style="width: min(680px, calc(100vw - 32px))"
		:title="t('controlPlane.diagnostics')"
		@update:show="emit('update:show', $event)"
	>
		<n-empty v-if="!status" :description="t('controlPlane.empty')" />
		<div v-else class="space-y-5">
			<div class="grid gap-3 sm:grid-cols-2">
				<div v-for="component in components" :key="component.name" class="rounded-xl border border-gray-200 p-4">
					<div class="flex items-center justify-between gap-3">
						<span class="font-semibold">{{ component.name }}</span>
						<n-tag :type="component.healthy ? 'success' : 'warning'">{{ stateText(component.state) }}</n-tag>
					</div>
					<div class="mt-3 space-y-2 break-all text-xs text-gray-500">
						<div>{{ t("controlPlane.status") }} · {{ component.reachable ? t("controlPlane.reachable") : t("controlPlane.unreachable") }}</div>
						<div>{{ t("controlPlane.socket") }} · {{ component.socketPath || "-" }}</div>
						<div v-if="component.version">{{ t("controlPlane.version") }} · {{ component.version }}</div>
					</div>
					<n-collapse v-if="component.error" class="mt-3">
						<n-collapse-item :title="t('controlPlane.rawError')" name="error">
							<div class="break-all text-xs text-red-500">{{ component.error }}</div>
						</n-collapse-item>
					</n-collapse>
				</div>
			</div>

			<n-alert v-if="status.healthy" type="success" :show-icon="true">
				{{ t("controlPlane.healthySummary") }}
			</n-alert>
			<n-alert v-else-if="status.autoRepairable" type="warning" :show-icon="true">
				{{ t("controlPlane.autoRepairDesc") }}
			</n-alert>
			<template v-else>
				<n-alert type="warning" :show-icon="true">{{ t("controlPlane.gpcRepairDesc") }}</n-alert>
				<div v-for="(command, index) in status.gpc.commands || []" :key="index" class="space-y-2">
					<div class="text-sm font-medium">{{ t("controlPlane.recoveryCommand") }}</div>
					<pre class="max-h-44 overflow-auto whitespace-pre-wrap break-all rounded-xl bg-gray-950 p-4 text-xs text-gray-100">{{ command }}</pre>
					<n-button size="small" secondary @click="copyCommand(command)">{{ t("controlPlane.copy") }}</n-button>
				</div>
			</template>

			<div class="flex flex-wrap justify-end gap-2">
				<n-button @click="emit('update:show', false)">{{ t("controlPlane.close") }}</n-button>
				<n-button v-if="!status.healthy && !status.autoRepairable" type="primary" :loading="loading" @click="emit('recheck')">
					{{ t("controlPlane.recheck") }}
				</n-button>
				<n-button v-if="status.autoRepairable" type="primary" :loading="repairing" @click="emit('repair')">
					{{ t("controlPlane.autoRepair") }}
				</n-button>
			</div>
		</div>
	</n-modal>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { ControlPlaneStatus } from "@/api/modules/agent"
import { controlPlaneMessages } from "@/i18n/locales/controlPlane"

const props = defineProps<{
	show: boolean
	status: ControlPlaneStatus | null
	loading: boolean
	repairing: boolean
}>()
const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "recheck"): void
	(e: "repair"): void
}>()
const { t } = useI18n({ messages: controlPlaneMessages })
const message = useMessage()
const components = computed(() => props.status ? [props.status.gpc, props.status.agent] : [])
const stateText = (state: string) => t(`controlPlane.state_${state}`)
const copyCommand = async (command: string) => {
	try {
		await navigator.clipboard.writeText(command)
		message.success(t("controlPlane.copied"))
	} catch {
		message.error(t("controlPlane.copyFailed"))
	}
}
</script>
