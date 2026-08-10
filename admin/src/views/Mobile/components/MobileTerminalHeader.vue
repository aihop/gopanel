<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { mobileTerminalMessages } from "../mobileTerminalMessages"

const props = defineProps<{
	taskName: string
	projectName: string
	mode: "ai" | "native"
	connected: boolean
	reconnecting: boolean
	hasSelection: boolean
}>()
const emit = defineEmits<{
	back: []
	openFiles: []
	openStatus: []
	openRename: []
	copySelection: []
	copyOutput: []
}>()
const terminalHeaderMessages = {
	zh: { mobile: { ...mobileMessages.zh.mobile, ...mobileTerminalMessages.zh.mobile } },
	en: { mobile: { ...mobileMessages.en.mobile, ...mobileTerminalMessages.en.mobile } }
}
const { t } = useI18n({ messages: terminalHeaderMessages })
const copyOptions = computed(() => [
	{
		label: t("mobile.copyTerminalSelection"),
		key: "selection",
		disabled: !props.hasSelection
	},
	{ label: t("mobile.copyTerminalOutput"), key: "output" }
])

function copyTerminal(key: string | number) {
	if (key === "selection") emit("copySelection")
	if (key === "output") emit("copyOutput")
}
</script>

<template>
	<header
		class="flex shrink-0 items-center gap-2 border-b border-white/10 bg-slate-950/80 px-2 pb-2 pt-[max(8px,env(safe-area-inset-top))] backdrop-blur"
	>
		<button
			class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-200 transition-colors active:bg-white/10"
			type="button"
			:title="t('commons.button.back')"
			:aria-label="t('commons.button.back')"
			@click="emit('back')"
		>
			<svg
				viewBox="0 0 24 24"
				aria-hidden="true"
				class="h-6 w-6 fill-none stroke-current"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path d="M19 12H5" />
				<path d="m12 19-7-7 7-7" />
			</svg>
		</button>
		<div class="min-w-0 flex-1">
			<div class="truncate text-sm font-semibold">{{ taskName }}</div>
		</div>
		<div class="flex shrink-0 items-center gap-1.5">
			<span class="max-w-[30vw] truncate text-right text-xs text-slate-400" :title="projectName">
				{{ projectName }}
			</span>
			<span
				class="h-2 w-2 rounded-full"
				:class="connected ? 'bg-emerald-400' : reconnecting ? 'bg-amber-400' : 'bg-slate-500'"
				:title="
					connected
						? t('mobile.connected')
						: reconnecting
							? t('mobile.reconnecting')
							: t('mobile.disconnected')
				"
			/>
			<n-dropdown trigger="click" :options="copyOptions" @select="copyTerminal">
				<n-button
					size="small"
					quaternary
					circle
					:title="t('mobile.copyTerminal')"
					:aria-label="t('mobile.copyTerminal')"
				>
					<template #icon><Icon name="mdi:content-copy" :size="18" color="#cbd5e1" /></template>
				</n-button>
			</n-dropdown>
			<n-button
				v-if="mode === 'ai'"
				size="small"
				quaternary
				circle
				:title="t('mobile.renameSession')"
				:aria-label="t('mobile.renameSession')"
				@click="emit('openRename')"
			>
				<template #icon><Icon name="mdi:pencil-outline" :size="18" color="#cbd5e1" /></template>
			</n-button>
			<n-button
				v-if="mode === 'ai'"
				size="small"
				quaternary
				circle
				:title="t('mobile.taskStatus')"
				:aria-label="t('mobile.taskStatus')"
				@click="emit('openStatus')"
			>
				<template #icon><Icon name="mdi:timeline-clock-outline" :size="19" color="#cbd5e1" /></template>
			</n-button>
			<n-button
				v-if="mode === 'ai'"
				size="small"
				quaternary
				circle
				:title="t('mobile.files')"
				:aria-label="t('mobile.files')"
				@click="emit('openFiles')"
			>
				<template #icon><Icon name="mdi:folder-outline" :size="19" color="#cbd5e1" /></template>
			</n-button>
		</div>
	</header>
</template>
