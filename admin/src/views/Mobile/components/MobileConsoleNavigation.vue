<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

defineProps<{ activeTab: "overview" | "resources" | "code" | "settings" }>()
const emit = defineEmits<{ select: [tab: "overview" | "resources" | "code" | "settings"] }>()
const { t } = useI18n({ messages: mobileMessages })
</script>

<template>
	<nav
		class="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white/95 pb-[max(8px,env(safe-area-inset-bottom))] pt-1 backdrop-blur"
	>
		<div class="mx-auto grid max-w-2xl grid-cols-4">
			<button
				class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors"
				:class="activeTab === 'overview' ? 'text-blue-600' : 'text-slate-400'"
				@click="emit('select', 'overview')"
			>
				<Icon
					:name="activeTab === 'overview' ? 'mdi:view-dashboard' : 'mdi:view-dashboard-outline'"
					:size="23"
				/>
				<span>{{ t("mobile.overview") }}</span>
			</button>
			<button
				class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors"
				:class="activeTab === 'code' ? 'text-blue-600' : 'text-slate-400'"
				@click="emit('select', 'code')"
			>
				<Icon name="mdi:console-line" :size="23" />
				<span>{{ t("mobile.code") }}</span>
			</button>
			<button
				class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors"
				:class="activeTab === 'resources' ? 'text-blue-600' : 'text-slate-400'"
				@click="emit('select', 'resources')"
			>
				<Icon :name="activeTab === 'resources' ? 'mdi:widgets' : 'mdi:widgets-outline'" :size="23" />
				<span>{{ t("mobile.resources") }}</span>
			</button>
			<button
				class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs transition-colors"
				:class="activeTab === 'settings' ? 'text-blue-600' : 'text-slate-400'"
				@click="emit('select', 'settings')"
			>
				<Icon :name="activeTab === 'settings' ? 'mdi:cog' : 'mdi:cog-outline'" :size="23" />
				<span>{{ t("menu.setting") }}</span>
			</button>
		</div>
	</nav>
</template>
