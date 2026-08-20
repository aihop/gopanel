<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { codeTaskTimestamp } from "../codeDashboardBuckets"
import TaskApprovalAction from "./TaskApprovalAction.vue"

const props = defineProps<{
	task: CodeTaskListItem
	projectName: string
	showProject?: boolean
	selected?: boolean
	archived?: boolean
	archiving?: boolean
}>()
const emit = defineEmits<{
	open: [task: CodeTaskListItem]
	archive: [task: CodeTaskListItem]
	openWorkspace: [task: CodeTaskListItem]
	refresh: []
}>()
const { t } = useI18n({ messages: codeProjectMessages })

const timeLabel = computed(() =>
	new Date(codeTaskTimestamp(props.task)).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" }),
)
const deliveryNeedsAttention = computed(() =>
	["failed", "partial", "conflict"].includes(props.task.summary.deliveryStatus || ""),
)
</script>

<template>
  <div
    class="dashboard-task-row group/row relative mb-0.5 flex cursor-pointer items-center gap-2 rounded-md py-1.5 pl-5 pr-2.5 transition-colors"
    :class="selected ? 'dashboard-task-row--selected' : ''"
    role="button"
    tabindex="0"
    @click="emit('open', task)"
    @keydown.enter.self="emit('open', task)"
    @keydown.space.self.prevent="emit('open', task)"
  >
    <span
      class="dashboard-task-row__marker absolute left-1.5 top-1.5 bottom-1.5 w-0.5 rounded-full"
      :class="selected ? 'bg-[var(--n-text-color-3)]' : 'bg-transparent'"
    />
    <div class="min-w-0 flex-1">
      <div class="flex min-w-0 items-center gap-1.5">
        <span
          v-if="showProject !== false"
          class="max-w-[88px] shrink-0 truncate text-[11px] text-[var(--n-text-color-3)]"
          :title="projectName"
        >
          {{ projectName }}
        </span>
        <span
          class="min-w-0 truncate text-[13px] tracking-[0.01em]"
          :class="selected ? 'font-medium text-[var(--n-text-color)]' : 'font-normal text-[var(--n-text-color-2)]'"
          :title="task.title"
        >
          {{ task.title }}
        </span>
      </div>
    </div>
    <TaskApprovalAction
      class="shrink-0"
      :task="task"
      compact
      @click.stop
      @approved="emit('refresh')"
    />
    <Icon
      v-if="deliveryNeedsAttention"
      name="mdi:alert-circle-outline"
      :size="14"
      class="shrink-0 text-amber-500"
      :title="t('code.dashboardDeliveryAttention')"
    />
    <n-button
      quaternary
      circle
      size="tiny"
      :loading="archiving"
      :title="archived ? t('code.taskUnarchive') : t('code.taskArchive')"
      class="shrink-0 opacity-0 transition-opacity group-hover/row:opacity-100"
      :class="archiving ? '!opacity-100' : ''"
      @click.stop="emit('archive', task)"
    >
      <template #icon>
        <Icon
          :name="archived ? 'mdi:archive-arrow-up-outline' : 'mdi:archive-arrow-down-outline'"
          :size="14"
        />
      </template>
    </n-button>
    <span class="w-10 shrink-0 text-right text-[11px] tabular-nums text-[var(--n-text-color-3)]">{{ timeLabel }}</span>
  </div>
</template>

<style scoped>
.dashboard-task-row--selected {
	background: color-mix(in srgb, var(--n-text-color) 4%, transparent);
}

.dashboard-task-row:not(.dashboard-task-row--selected):hover {
	background: color-mix(in srgb, var(--n-text-color) 3%, transparent);
}
</style>
