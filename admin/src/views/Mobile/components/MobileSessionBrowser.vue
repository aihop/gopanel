<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIProject, CodeSession } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

const props = defineProps<{
	projects: AIProject[]
	sessions: CodeSession[]
	selectedProjectId: number | null
	selectedSessionId: number
	loading: boolean
}>()
const emit = defineEmits<{
	"update:selectedProjectId": [projectId: number]
	"new-session": []
	"project-terminal": []
	"select-session": [session: CodeSession]
}>()
const { t } = useI18n({ messages: mobileMessages })

const projectOptions = computed(() => props.projects.map(project => ({ label: project.name, value: project.id })))
</script>

<template>
	<section class="space-y-4">
		<div class="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm">
			<div class="mb-2 text-xs font-medium text-slate-500">{{ t("mobile.projectSessions") }}</div>
			<n-select
				:value="selectedProjectId"
				:options="projectOptions"
				:placeholder="t('mobile.selectProject')"
				:disabled="loading || projects.length === 0"
				@update:value="emit('update:selectedProjectId', $event)"
			/>
		</div>
		<div class="grid grid-cols-2 gap-2">
			<n-button type="primary" secondary class="!h-11 !rounded-xl" @click="emit('new-session')">
				<template #icon><Icon name="mdi:robot-outline" /></template>
				{{ t("mobile.newSession") }}
			</n-button>
			<n-button secondary class="!h-11 !rounded-xl" @click="emit('project-terminal')">
				<template #icon><Icon name="mdi:console-line" /></template>
				{{ t("mobile.projectTerminal") }}
			</n-button>
		</div>
		<div v-if="loading" class="flex justify-center rounded-2xl bg-white py-16 shadow-sm">
			<n-spin size="small" />
		</div>
		<n-empty v-else-if="projects.length === 0" :description="t('mobile.noProjects')" class="rounded-2xl bg-white py-16" />
		<n-empty v-else-if="!loading && sessions.length === 0" :description="t('mobile.noProjectSessions')" class="rounded-2xl bg-white py-16">
			<template #extra>
				<n-button type="primary" @click="emit('new-session')">{{ t("mobile.newSession") }}</n-button>
			</template>
		</n-empty>
		<div v-else class="space-y-2">
			<button
				v-for="session in sessions"
				:key="session.id"
				type="button"
				class="flex w-full min-w-0 items-center gap-3 rounded-2xl border border-slate-200 bg-white p-3 text-left shadow-sm transition active:scale-[0.99]"
				:class="selectedSessionId === session.id ? 'border-blue-300 bg-blue-50' : ''"
				@click="emit('select-session', session)"
			>
				<span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-blue-600">
					<Icon name="mdi:robot-outline" :size="20" />
				</span>
				<span class="min-w-0 flex-1">
					<span class="block truncate text-sm font-semibold text-slate-900">{{ session.title }}</span>
					<span class="mt-0.5 block truncate text-xs text-slate-500">{{ session.agentName }}</span>
				</span>
				<Icon name="mdi:chevron-right" class="shrink-0 text-slate-400" />
			</button>
		</div>
	</section>
</template>
