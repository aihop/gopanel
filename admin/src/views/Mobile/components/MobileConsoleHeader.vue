<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { mobileMessages } from "@/i18n/locales/mobile"
import Logo from "@/layouts/common/Logo.vue"

defineProps<{
	activeTab: "overview" | "resources" | "code" | "settings"
	nodeName: string
	nodeOnline: boolean
	canCreateSession: boolean
}>()
const emit = defineEmits<{ selectNode: []; newSession: [] }>()
const { t } = useI18n({ messages: mobileMessages })
</script>

<template>
	<header class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur">
		<div class="mx-auto flex max-w-2xl items-center justify-between">
			<div class="flex min-w-0 items-center gap-3">
				<Logo :dark="false" class="shrink-0" />
				<button
					v-if="activeTab !== 'code' || !canCreateSession"
					type="button"
					class="mt-0.5 flex max-w-[65vw] items-center gap-1 border-0 bg-transparent p-0 text-xs text-slate-500"
					@click="emit('selectNode')"
				>
					<span
						class="h-2 w-2 shrink-0 rounded-full"
						:class="nodeOnline ? 'bg-emerald-500' : 'bg-slate-400'"
					/>
					<span class="truncate">{{ nodeName || t("mobile.selectNode") }}</span>
					<span>⌄</span>
				</button>
			</div>
			<div class="flex shrink-0 items-center gap-1">
				<n-button
					v-if="canCreateSession && (activeTab === 'overview' || activeTab === 'code')"
					size="small"
					type="primary"
					secondary
					@click="emit('newSession')"
				>
					{{ t("mobile.newSession") }}
				</n-button>
			</div>
		</div>
	</header>
</template>
