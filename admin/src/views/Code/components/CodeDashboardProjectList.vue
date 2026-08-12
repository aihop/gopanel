<script setup lang="ts">
import { computed, watch } from "vue"
import { useStorage } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import CodeDashboardTaskRow from "./CodeDashboardTaskRow.vue"

const props = defineProps<{
	projects: AIProject[]
	tasks: CodeTaskListItem[]
	selectedTaskId: number | null
	archived: boolean
	archivingTaskId: number | null
}>()
const emit = defineEmits<{
	open: [task: CodeTaskListItem]
	archive: [task: CodeTaskListItem]
	openWorkspace: [task: CodeTaskListItem]
	createTask: [projectId: number]
	projectAction: [action: string, projectId: number]
	refresh: []
}>()
const { t } = useI18n({ messages: codeProjectMessages })
const collapsedProjects = useStorage<Record<string, boolean>>("code-dashboard-collapsed-projects", {})

const groups = computed(() => {
	const tasksByProject = new Map<number, CodeTaskListItem[]>()
	for (const task of props.tasks) {
		const projectTasks = tasksByProject.get(task.projectId) || []
		projectTasks.push(task)
		tasksByProject.set(task.projectId, projectTasks)
	}
	const knownProjectIds = new Set(props.projects.map(project => project.id))
	const knownGroups = props.projects.map(project => ({
		id: project.id,
		name: project.name,
		available: true,
		tasks: tasksByProject.get(project.id) || [],
	}))
	const unavailableGroups = [...tasksByProject.entries()]
		.filter(([projectId]) => !knownProjectIds.has(projectId))
		.map(([projectId, tasks]) => ({
			id: projectId,
			name: t("code.dashboardUnavailableProject", { id: projectId }),
			available: false,
			tasks,
		}))
	return [...knownGroups, ...unavailableGroups]
})

const selectedProjectId = computed(() => props.tasks.find(task => task.id === props.selectedTaskId)?.projectId || null)
const isCollapsed = (projectId: number) => collapsedProjects.value[String(projectId)] === true
const toggleProject = (projectId: number) => {
	collapsedProjects.value = {
		...collapsedProjects.value,
		[String(projectId)]: !isCollapsed(projectId),
	}
}

const projectActionOptions = computed(() => [
	{ label: t("code.enterProject"), key: "enter" },
	{ label: t("code.quickPanel"), key: "panel" },
	{ label: t("code.editProject"), key: "edit" },
])

watch(() => props.selectedTaskId, () => {
	const projectId = selectedProjectId.value
	if (!projectId || collapsedProjects.value[String(projectId)] !== true) return
	const next = { ...collapsedProjects.value }
	delete next[String(projectId)]
	collapsedProjects.value = next
})
</script>

<template>
  <div class="space-y-2 p-2">
    <section
      v-for="group in groups"
      :key="group.id"
      class="overflow-hidden rounded-2xl"
    >
      <div class="group/project flex h-10 items-center gap-1 px-1">
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-2 rounded-xl px-2 py-1.5 text-left text-sm text-[var(--n-text-color-2)] transition-colors hover:bg-[var(--n-color-embedded)] hover:text-[var(--n-text-color)]"
          :aria-expanded="!isCollapsed(group.id)"
          @click="toggleProject(group.id)"
        >
          <Icon
            :name="isCollapsed(group.id) ? 'mdi:chevron-right' : 'mdi:chevron-down'"
            :size="16"
            class="shrink-0 text-[var(--n-text-color-3)]"
          />
          <span
            class="min-w-0 flex-1 truncate font-medium"
            :title="group.name"
          >{{ group.name }}</span>
          <span class="shrink-0 text-xs text-[var(--n-text-color-3)]">{{ group.tasks.length }}</span>
        </button>
        <n-tooltip v-if="group.available && !archived">
          <template #trigger>
            <n-button
              quaternary
              circle
              size="tiny"
              @click.stop="emit('createTask', group.id)"
            >
              <template #icon>
                <Icon
                  name="mdi:plus"
                  :size="18"
                />
              </template>
            </n-button>
          </template>
          {{ t("code.dashboardCreateProjectTask", { name: group.name }) }}
        </n-tooltip>
        <n-dropdown
          v-if="group.available"
          trigger="click"
          :options="projectActionOptions"
          @select="action => emit('projectAction', String(action), group.id)"
        >
          <n-button
            quaternary
            circle
            size="tiny"
            :title="t('code.project')"
          >
            <template #icon>
              <Icon
                name="mdi:dots-horizontal"
                :size="17"
              />
            </template>
          </n-button>
        </n-dropdown>
      </div>

      <div v-if="!isCollapsed(group.id)">
        <div
          v-if="group.tasks.length === 0"
          class="px-10 py-2 text-xs text-[var(--n-text-color-3)]"
        >
          {{ t("code.dashboardNoProjectTasks") }}
        </div>
        <template v-else>
          <CodeDashboardTaskRow
            v-for="task in group.tasks"
            :key="task.id"
            :task="task"
            :project-name="group.name"
            :show-project="false"
            :selected="task.id === selectedTaskId"
            :archived="archived"
            :archiving="archivingTaskId === task.id"
            @open="emit('open', $event)"
            @archive="emit('archive', $event)"
            @open-workspace="emit('openWorkspace', $event)"
            @refresh="emit('refresh')"
          />
        </template>
      </div>
    </section>
  </div>
</template>
