<template>
	<div v-if="validate" class="bg-base-100 mt-3 rounded-[20px] p-4 px-6 shadow">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<n-tag :type="summaryTagType">{{ summaryText }}</n-tag>
				<span v-if="validate.summary?.runtimeInstalled" class="text-sm text-gray-500">
					{{ runtimeBadgeText }}
				</span>
				<span v-else class="text-sm text-gray-500">{{ t("containerRuntime.notInstalledDesc") }}</span>
			</div>
			<n-button quaternary size="small" @click="showDiagnostics = !showDiagnostics">
				{{ t("containerRuntime.diagnostics") }}
			</n-button>
		</div>

		<n-collapse-transition :show="showDiagnostics">
			<div class="mt-4 space-y-3 border-t border-gray-100 pt-4 text-xs text-gray-500">
				<div class="flex flex-wrap gap-x-5 gap-y-2">
					<span>
						{{
							validate.hostPinned ? t("containerRuntime.pinnedSocket") : t("containerRuntime.autoDetect")
						}}
					</span>
					<span>{{ t("containerRuntime.current") }}: {{ currentRuntimeHost }}</span>
					<span>{{ t("containerRuntime.configured") }}: {{ validate.configuredHost || "-" }}</span>
					<span>OS: {{ validate.os }}/{{ validate.arch }}</span>
					<span>
						{{ t("containerRuntime.compose") }}:
						{{ validate.compose?.ok ? t("containerRuntime.available") : t("containerRuntime.unavailable") }}
					</span>
					<span>
						GPC:
						{{
							validate.gpc?.reachable
								? t("containerRuntime.connected")
								: t("containerRuntime.disconnected")
						}}
					</span>
				</div>
				<div v-if="validate.notes?.length" class="space-y-1 text-orange-600">
					<div v-for="(note, index) in validate.notes" :key="index">- {{ diagnosticNote(note) }}</div>
				</div>
				<div v-if="dockerOnly && validate.summary?.runtimeInstalled" class="text-orange-600">
					{{ t("containerRuntime.podmanSettingsNotice", { runtime: validate.runtimeKind }) }}
				</div>
			</div>
		</n-collapse-transition>
	</div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import { containerRuntimeMessages } from "../../../i18n/locales/containerRuntime"

const props = defineProps<{
	validate: any
	runtimeBadgeText: string
	currentRuntimeHost: string
	dockerOnly: boolean
}>()

const { t } = useI18n({ messages: containerRuntimeMessages })
const showDiagnostics = ref(false)
const diagnosticNote = (note: string) => t(`containerRuntime.note_${note}`)
const summaryText = computed(() => {
	const state = props.validate?.summary?.state
	if (state === "notInstalled") return t("containerRuntime.notInstalled")
	if (state === "notReady") return t("containerRuntime.notReady")
	if (state === "composeMissing") return t("containerRuntime.composeMissing")
	return t("containerRuntime.ready")
})
const summaryTagType = computed(() => {
	const state = props.validate?.summary?.state
	if (state === "ready") return "success"
	if (state === "notInstalled") return "error"
	return "warning"
})
</script>
