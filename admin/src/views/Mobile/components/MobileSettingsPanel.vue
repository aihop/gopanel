<script setup lang="ts">
import type { MobileNode } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"
import { useLocalesStore, type I18nLangCode } from "@/store/i18n"
import Icon from "@/components/common/Icon.vue"
import MobileSystemUpdate from "./MobileSystemUpdate.vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"

defineProps<{
	node: MobileNode | null
	nodeOnline: boolean
}>()
const emit = defineEmits<{ selectNode: []; logout: [] }>()
const { t } = useI18n({ messages: mobileMessages })
const localesStore = useLocalesStore()

const locale = computed({
	get: () => localesStore.locale,
	set: value => localesStore.setLocale(value as I18nLangCode)
})

const languageOptions = computed(() => [
	{ label: t("lang.chinese"), value: "zh" },
	{ label: t("lang.english"), value: "en" }
])
</script>

<template>
	<div class="space-y-4">
		<section class="overflow-hidden rounded-2xl bg-white shadow-sm">
			<button
				type="button"
				class="flex w-full items-center gap-3 border-0 bg-transparent p-4 text-left"
				@click="emit('selectNode')"
			>
				<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-blue-600">
					<Icon name="mdi:server-network" :size="22" />
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-sm font-medium text-slate-900">{{ t("mobile.selectNode") }}</div>
					<div class="mt-1 flex items-center gap-2 text-xs text-slate-500">
						<span :class="nodeOnline ? 'bg-emerald-500' : 'bg-slate-400'" class="h-2 w-2 rounded-full" />
						<span class="truncate">{{ node?.name || t("mobile.selectNode") }}</span>
					</div>
				</div>
				<Icon name="mdi:chevron-right" class="text-slate-400" />
			</button>
		</section>

		<section class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="mb-3 flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-50 text-violet-600">
					<Icon name="mdi:translate" :size="22" />
				</div>
				<div class="text-sm font-medium text-slate-900">{{ t("setting.language") }}</div>
			</div>
			<n-radio-group v-model:value="locale" class="grid w-full grid-cols-2 gap-2">
				<n-radio-button v-for="option in languageOptions" :key="option.value" :value="option.value">
					{{ option.label }}
				</n-radio-button>
			</n-radio-group>
		</section>

		<MobileSystemUpdate show-current-version />

		<section class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="flex items-start gap-3">
				<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-50 text-rose-600">
					<Icon name="mdi:cellphone-key" :size="22" />
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-sm font-medium text-slate-900">{{ t("mobile.logout") }}</div>
					<div class="mt-1 text-xs leading-5 text-slate-500">{{ t("mobile.logoutConfirm") }}</div>
				</div>
			</div>
			<n-button class="mt-4 w-full" type="error" secondary @click="emit('logout')">
				{{ t("mobile.logout") }}
			</n-button>
		</section>
	</div>
</template>
