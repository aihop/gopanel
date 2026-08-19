<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { openMobileProjectTerminal } from "@/api/modules/mobile"
import type { AIProject } from "@/api/interface/code"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

defineProps<{ show: boolean; projects: AIProject[] }>()
const emit = defineEmits<{
	"update:show": [show: boolean]
	opened: [session: HostTerminalSession, project: AIProject]
}>()
const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const openingProjectId = ref(0)

async function openTerminal(project: AIProject) {
	if (openingProjectId.value) return
	openingProjectId.value = project.id
	try {
		const session = await openMobileProjectTerminal(project.id)
		emit("update:show", false)
		emit("opened", session, project)
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.projectTerminalOpenFailed"))
	} finally {
		openingProjectId.value = 0
	}
}
</script>

<template>
	<n-modal
		:show="show"
		preset="card"
		style="width: min(440px, calc(100vw - 24px))"
		:title="t('mobile.projectTerminals')"
		@update:show="emit('update:show', $event)"
	>
		<div class="mb-4 text-sm text-slate-500">{{ t("mobile.projectTerminalHint") }}</div>
		<n-empty v-if="projects.length === 0" size="small" :description="t('mobile.noProjects')" />
		<div v-else class="grid grid-cols-1 gap-2">
			<button
				v-for="project in projects"
				:key="project.id"
				type="button"
				class="flex min-w-0 items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 text-left transition active:scale-[0.99] active:bg-slate-100"
				:disabled="openingProjectId !== 0"
				@click="openTerminal(project)"
			>
				<span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white">
					<Icon
						:name="openingProjectId === project.id ? 'mdi:loading' : 'mdi:console'"
						:size="19"
						:class="openingProjectId === project.id ? 'animate-spin' : ''"
					/>
				</span>
				<span class="min-w-0 flex-1">
					<span class="block truncate text-sm font-semibold text-slate-800">{{ project.name }}</span>
					<span class="mt-0.5 block truncate text-xs text-slate-500">
						{{ t("mobile.openProjectTerminal") }}
					</span>
				</span>
			</button>
		</div>
	</n-modal>
</template>
