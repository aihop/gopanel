<template>
	<div
		class="flex flex-col gap-4 rounded-[22px] px-5 py-4 shadow-[0_1px_2px_rgba(37,99,235,0.05)] xl:flex-row xl:items-center xl:justify-between"
	>
		<div class="flex min-w-0 items-center gap-4">
			<div class="text-xs font-bold uppercase tracking-[0.08em] text-blue-600">{{ t("home.baseInfo") }}</div>
			<div v-if="editingHostname" class="flex min-w-0 flex-1 flex-col gap-1.5">
				<div class="flex items-center gap-2">
					<n-input
						ref="hostnameInputRef"
						:value="hostnameDraft"
						:placeholder="t('dashboardHostname.placeholder')"
						:status="hostnameError ? 'error' : undefined"
						:maxlength="253"
						:disabled="hostnameSaving"
						@update:value="handleHostnameInput"
						@keydown.enter.prevent="saveHostname"
						@keydown.esc.prevent="cancelHostnameEdit"
					/>
					<n-button size="small" type="primary" :loading="hostnameSaving" @click="saveHostname">
						{{ t("dashboardHostname.save") }}
					</n-button>
					<n-button size="small" :disabled="hostnameSaving" @click="cancelHostnameEdit">
						{{ t("dashboardHostname.cancel") }}
					</n-button>
				</div>
				<span class="text-xs" :class="hostnameError ? 'text-red-500' : 'text-slate-500'">
					{{ hostnameError || t("dashboardHostname.rule") }}
				</span>
			</div>
			<div v-else class="flex min-w-0 items-center gap-2">
				<span class="min-w-0 truncate text-xl font-bold leading-[1.1] text-slate-900">
					{{ baseInfo.hostname || "--" }}
				</span>
				<n-button size="tiny" quaternary type="primary" @click="startHostnameEdit">
					{{ t("dashboardHostname.edit") }}
				</n-button>
			</div>
			<div class="hidden truncate text-[13px] text-slate-500 md:block">
				{{ [baseInfo.os, baseInfo.platformVersion, baseInfo.kernelArch].filter(Boolean).join(" · ") || "--" }}
			</div>
		</div>

		<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-slate-500 xl:justify-end">
			<span>
				<strong class="mr-1 font-semibold text-slate-900">IPv4</strong>
				{{ baseInfo.ipv4Addr || "--" }}
			</span>
			<span>
				<strong class="mr-1 font-semibold text-slate-900">{{ t("home.runningTime") }}</strong>
				{{ formatUptime(currentInfo.uptime) }}
			</span>
			<span>
				<strong class="mr-1 font-semibold text-slate-900">{{ t("menu.process") }}</strong>
				{{ currentInfo.procs }}
			</span>
			<span>
				<strong class="mr-1 font-semibold text-slate-900">{{ t("home.kernelVersion") }}</strong>
				{{ shortText(baseInfo.kernelVersion || "--", 16) }}
			</span>
			<n-tag size="small" :bordered="false" :type="lowPowerMode ? 'warning' : 'info'" round>
				{{ lowPowerMode ? t("dashboardHostname.lowPowerMode") : t("dashboardHostname.standardMode") }}
			</n-tag>
			<n-switch size="small" :value="lowPowerMode" @update:value="emit('set-low-power-mode', $event)" />
		</div>
	</div>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { updateHostname } from "@/api/modules/dashboard"
import { dashboardHostnameMessages } from "@/i18n/locales/dashboardHostname"
import { nextTick, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { formatUptime, shortText } from "./dashboardStatusHelpers"

const props = defineProps<{
	baseInfo: Dashboard.BaseInfo
	currentInfo: Dashboard.CurrentInfo
	lowPowerMode: boolean
	memoryCleaning: boolean
	cpuRelieving: boolean
}>()

const emit = defineEmits<{
	(e: "set-low-power-mode", value: boolean): void
	(e: "memory-clean"): void
	(e: "cpu-relieve"): void
}>()

const { t } = useI18n({ messages: dashboardHostnameMessages })
const message = useMessage()
const editingHostname = ref(false)
const hostnameDraft = ref("")
const hostnameError = ref("")
const hostnameSaving = ref(false)
const hostnameInputRef = ref<{ focus: () => void } | null>(null)

function startHostnameEdit() {
	hostnameDraft.value = props.baseInfo.hostname
	hostnameError.value = ""
	editingHostname.value = true
	nextTick(() => hostnameInputRef.value?.focus())
}

function cancelHostnameEdit() {
	if (hostnameSaving.value) return
	editingHostname.value = false
	hostnameDraft.value = ""
	hostnameError.value = ""
}

function handleHostnameInput(value: string) {
	hostnameDraft.value = value.replace(/[^A-Za-z0-9.-]/g, "")
	hostnameError.value = value === hostnameDraft.value ? "" : t("dashboardHostname.ErrHostnameInvalid")
}

async function saveHostname() {
	if (hostnameSaving.value) return
	const hostname = hostnameDraft.value.trim()
	if (!isValidHostname(hostname)) {
		hostnameError.value = t("dashboardHostname.ErrHostnameInvalid")
		return
	}
	if (hostname === props.baseInfo.hostname) {
		cancelHostnameEdit()
		return
	}

	hostnameSaving.value = true
	try {
		const res = await updateHostname(hostname)
		if (res.code !== 0 || !res.data?.hostname) {
			throw new Error(res.msg || res.message || "ErrHostnameUpdateFailed")
		}
		props.baseInfo.hostname = res.data.hostname
		editingHostname.value = false
		message.success(t("dashboardHostname.updated"))
	} catch (error: any) {
		const errorCode = error?.message
		const knownError = ["ErrHostnameInvalid", "ErrHostnameToolUnavailable", "ErrHostnameUpdateFailed"].includes(
			errorCode
		)
		message.error(t(knownError ? `dashboardHostname.${errorCode}` : "dashboardHostname.updateFailed"))
	} finally {
		hostnameSaving.value = false
	}
}

function isValidHostname(hostname: string) {
	if (!hostname || hostname.length > 253 || !/^[A-Za-z0-9.-]+$/.test(hostname)) return false
	return hostname
		.split(".")
		.every(label => label.length > 0 && label.length <= 63 && !label.startsWith("-") && !label.endsWith("-"))
}
</script>
