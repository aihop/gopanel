<template>
	<div class="bg-base-100 mt-3 rounded-[20px] p-4 px-6 shadow">
		<div class="flex items-center justify-between">
			<n-space v-if="daemon.status" align="center">
				<n-tag type="success" class="uppercase">{{ daemon.containerType }}</n-tag>
				<n-tag v-if="daemon.status" type="warning">
					{{ dockerStatusText[daemon.status] }}
				</n-tag>
				<span class="text-sm text-gray-500">{{ t("containerRuntime.version") }}: {{ daemon.version }}</span>
			</n-space>
			<n-space>
				<n-button
					v-if="daemon.status === dockerStatus.Stopped"
					:loading="statusLoading"
					type="primary"
					@click="emit('update-status', 'start')"
				>
					{{ $t("container.start") }}
				</n-button>
				<n-popconfirm v-else-if="daemon.status" @positive-click="emit('update-status', 'stop')">
					<template #trigger>
						<n-button :loading="statusLoading" type="warning">{{ t("containerRuntime.stop") }}</n-button>
					</template>
					{{ t("containerRuntime.stopConfirm") }}
				</n-popconfirm>
				<n-popconfirm v-if="daemon.status" @positive-click="emit('update-status', 'restart')">
					<template #trigger>
						<n-button
							:loading="reloadLoading"
							:disabled="daemon.status === dockerStatus.Stopped"
							type="error"
						>
							{{ t("containerRuntime.restart") }}
						</n-button>
					</template>
					{{ t("containerRuntime.restartConfirm") }}
				</n-popconfirm>
				<n-button :disabled="!validate" :type="repairHintType" @click="emit('open-repair')">
					{{ t("containerRuntime.problemRepair") }}
				</n-button>
			</n-space>
		</div>
	</div>
</template>

<script setup lang="ts">
import { dockerStatus, dockerStatusText } from "../../../enums/dockerStatus.enum"
import { containerRuntimeMessages } from "../../../i18n/locales/containerRuntime"
import { useI18n } from "vue-i18n"

const { t } = useI18n({ messages: containerRuntimeMessages })

defineProps<{
	daemon: any
	validate: any
	statusLoading: boolean
	reloadLoading: boolean
	repairHintType: "default" | "primary" | "info" | "success" | "warning" | "error"
}>()

const emit = defineEmits<{
	(e: "update-status", operation: string): void
	(e: "open-repair"): void
}>()
</script>
