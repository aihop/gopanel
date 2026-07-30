<template>
	<n-modal
		:show="show"
		preset="dialog"
		:title="t('containerRuntime.runtimeRepair')"
		:positive-text="t('containerRuntime.close')"
		:show-icon="false"
		@update:show="emit('update:show', $event)"
		@positive-click="emit('update:show', false)"
	>
		<div class="space-y-4">
			<template v-if="!validate?.summary?.runtimeInstalled">
				<div>
					<div class="font-medium">{{ t("containerRuntime.installTitle") }}</div>
					<div class="mt-1 text-xs text-gray-500">{{ t("containerRuntime.installDesc") }}</div>
				</div>

				<n-alert v-if="!validate?.summary?.installSupported" type="warning" :show-icon="false">
					{{ t("containerRuntime.unsupported") }}
				</n-alert>
				<n-alert v-else-if="!validate?.gpc?.reachable" type="warning" :show-icon="false">
					{{ t("containerRuntime.gpcRequired") }}
				</n-alert>

				<n-radio-group v-model:value="selectedRuntime" :disabled="installLoading">
					<div class="space-y-2">
						<label
							v-for="runtimeOption in runtimeOptions"
							:key="runtimeOption"
							class="flex cursor-pointer items-start gap-3 rounded-xl border border-gray-200 p-3"
						>
							<n-radio :value="runtimeOption" />
							<span>
								<span class="flex items-center gap-2 font-medium">
									{{ runtimeName(runtimeOption) }}
									<n-tag
										v-if="validate?.summary?.recommended === runtimeOption"
										size="small"
										type="success"
									>
										{{ t("containerRuntime.recommended") }}
									</n-tag>
								</span>
								<span class="mt-1 block text-xs text-gray-500">
									{{ t(`containerRuntime.${runtimeOption}Desc`) }}
								</span>
							</span>
						</label>
					</div>
				</n-radio-group>

				<n-popconfirm
					:positive-text="t('containerRuntime.install', { runtime: runtimeName(selectedRuntime) })"
					@positive-click="emit('install', selectedRuntime)"
				>
					<template #trigger>
						<n-button
							type="primary"
							:loading="installLoading"
							:disabled="
								!validate?.summary?.installSupported || !validate?.gpc?.reachable || installLoading
							"
						>
							{{ t("containerRuntime.install", { runtime: runtimeName(selectedRuntime) }) }}
						</n-button>
					</template>
					{{ t("containerRuntime.installConfirm", { runtime: runtimeName(selectedRuntime) }) }}
				</n-popconfirm>
			</template>

			<template v-else>
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div class="text-sm text-gray-600">{{ runtimeDetailText }}</div>
					<n-button
						v-if="canAutoRepair"
						:loading="autoRepairLoading"
						:disabled="!validate?.gpc?.reachable"
						type="primary"
						@click="emit('auto-repair')"
					>
						{{ t("containerRuntime.autoRepair") }}
					</n-button>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<n-tag :type="validate?.runtime?.serviceActive ? 'success' : 'warning'">
						{{ t("containerRuntime.service") }}:
						{{
							t(
								validate?.runtime?.serviceActive
									? "containerRuntime.serviceActive"
									: "containerRuntime.serviceInactive"
							)
						}}
					</n-tag>
					<n-tag :type="validate?.runtime?.apiReady ? 'success' : 'warning'">
						{{ t("containerRuntime.api") }}:
						{{
							t(
								validate?.runtime?.apiReady
									? "containerRuntime.apiReady"
									: "containerRuntime.apiNotReady"
							)
						}}
					</n-tag>
				</div>
				<div class="flex flex-wrap gap-2">
					<n-button
						v-if="canAutoRepair"
						:loading="repairSocketLoading"
						:disabled="!validate?.gpc?.reachable || autoRepairLoading"
						type="warning"
						@click="emit('repair-socket')"
					>
						{{ t("containerRuntime.repairSocket") }}
					</n-button>
					<n-button
						v-if="canAutoRepair"
						:loading="repairLingerLoading"
						:disabled="!validate?.gpc?.reachable || autoRepairLoading"
						@click="emit('repair-linger')"
					>
						{{ t("containerRuntime.repairLinger") }}
					</n-button>
				</div>
				<div v-if="canAutoRepair && !validate?.gpc?.reachable" class="text-xs text-gray-500">
					{{ t("containerRuntime.repairNeedsGpc") }}
				</div>
			</template>

			<n-alert v-if="installTask?.status === 'running'" type="info" :show-icon="false">
				{{ t("containerRuntime.installing", { runtime: runtimeName(installTask.runtime) }) }}
			</n-alert>
			<n-alert
				v-else-if="installTask?.status === 'failed'"
				type="error"
				:title="t('containerRuntime.installFailed')"
			>
				{{ installTask.message }}
			</n-alert>
			<n-alert v-else-if="installTask?.needsAction" type="warning" :show-icon="false">
				{{
					t(
						installTask.needsAction === "composeMissing"
							? "containerRuntime.composeMissingAction"
							: `containerRuntime.${installTask.needsAction}`
					)
				}}
			</n-alert>

			<n-collapse v-if="installTask?.output">
				<n-collapse-item :title="t('containerRuntime.installationLog')" name="install-log">
					<pre class="max-h-52 overflow-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs">{{
						installTask.output
					}}</pre>
				</n-collapse-item>
			</n-collapse>
		</div>
	</n-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { containerRuntimeMessages } from "../../../i18n/locales/containerRuntime"

const props = defineProps<{
	show: boolean
	validate: any
	runtimeDetailText: string
	canAutoRepair: boolean
	autoRepairLoading: boolean
	repairSocketLoading: boolean
	repairLingerLoading: boolean
	installLoading: boolean
	installTask: any
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "auto-repair"): void
	(e: "repair-socket"): void
	(e: "repair-linger"): void
	(e: "install", runtime: "docker" | "podman"): void
}>()

const { t } = useI18n({ messages: containerRuntimeMessages })
const selectedRuntime = ref<"docker" | "podman">("podman")
const runtimeOptions = computed<Array<"docker" | "podman">>(() => props.validate?.summary?.installOptions || [])
const runtimeName = (runtime: string) => (runtime === "docker" ? "Docker" : "Podman")

watch(
	() => props.validate?.summary?.recommended,
	recommended => {
		if (recommended === "docker" || recommended === "podman") selectedRuntime.value = recommended
	},
	{ immediate: true }
)
</script>
