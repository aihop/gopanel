<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import type { AIProject } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { codeDashboardFocusStatus } from "../codeDashboardBuckets"
import Icon from "@/components/common/Icon.vue"
import CodeProjectIdentity from "./CodeProjectIdentity.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"

const props = defineProps<{ projects: AIProject[]; tasks: CodeTaskListItem[]; selectedTaskId: number | null }>()
const emit = defineEmits<{ select: [task: CodeTaskListItem] }>()
const { t } = useI18n({ messages: codeProjectMessages })
const expanded = ref(false)
const visibleTasks = computed(() => (expanded.value ? props.tasks : props.tasks.slice(0, 5)))
const hiddenCount = computed(() => Math.max(0, props.tasks.length - visibleTasks.value.length))
const projectNames = computed(() => new Map(props.projects.map(project => [project.id, project.name])))
</script>

<template>
  <section class="border-b border-slate-200/70 px-3 py-3 dark:border-white/10">
    <div class="mb-2 flex items-center justify-between gap-2 px-1">
      <div class="flex min-w-0 items-center gap-2 text-xs font-semibold text-[var(--n-text-color-2)]">
        <Icon
          name="mdi:progress-clock"
          :size="15"
          class="shrink-0 text-blue-500"
        />
        <span class="truncate">{{ t("code.dashboardFocusTitle") }}</span>
        <span class="rounded-full bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-500">
          {{ tasks.length }}
        </span>
      </div>
      <n-button
        v-if="tasks.length > 5"
        text
        size="tiny"
        @click="expanded = !expanded"
      >
        {{
          expanded
            ? t("code.dashboardFocusCollapse")
            : t("code.dashboardFocusViewAll", { count: hiddenCount })
        }}
      </n-button>
    </div>
    <div class="max-h-60 space-y-1 overflow-y-auto">
      <button
        v-for="task in visibleTasks"
        :key="task.id"
        type="button"
        class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-[var(--n-color-embedded)]"
        :class="task.id === selectedTaskId ? 'bg-[var(--n-color-embedded)]' : ''"
        @click="emit('select', task)"
      >
        <CodeProjectIdentity
          class="max-w-24 shrink-0 text-[10px] text-[var(--n-text-color-3)]"
          :project-id="task.projectId"
          :name="projectNames.get(task.projectId) || t('code.projectFallback')"
        />
        <span
          class="min-w-0 flex-1 truncate text-xs font-medium text-[var(--n-text-color)]"
          :title="task.title"
        >
          {{ task.title }}
        </span>
        <TaskStatusBadge
          class="shrink-0"
          :status="codeDashboardFocusStatus(task)"
        />
      </button>
    </div>
  </section>
</template>
