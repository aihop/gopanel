<script setup lang="ts">
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import CodeTaskRepositoryPopover from "./CodeTaskRepositoryPopover.vue"
import { useCodeTaskMeta } from "../useCodeTaskMeta"

defineProps<{ task: CodeTaskListItem }>()
const {
  t,
  formatTaskDuration,
  formatTaskTokens,
  taskTokenStatus,
  taskGitMeta,
  taskTooltip,
  taskError,
  taskStage,
  taskDeliveryProgress,
} = useCodeTaskMeta()
</script>

<template>
  <!--
		任务行的第二行元信息。lead 插槽留给状态徽标/审批按钮：
		调用方自己决定放 TaskApprovalAction 还是纯 TaskStatusBadge，
		后面的 `·` 分隔符默认 lead 一定有内容。
	-->
  <div
    class="flex min-w-0 items-center gap-1.5 overflow-hidden whitespace-nowrap text-[10px] text-slate-400"
    :title="taskTooltip(task)"
  >
    <slot name="lead" />
    <!--
      出错和阶段排在最前面：这一行是 overflow-hidden 的，
      排在后面的内容窄屏会被截掉 —— 而「挂了」正是最不能被截掉的那条。
    -->
    <template v-if="taskError(task)">
      <span class="text-slate-300">·</span>
      <span
        class="min-w-0 truncate font-medium text-red-500"
        :title="taskError(task)"
      >{{ taskError(task) }}</span>
    </template>
    <template v-else-if="taskDeliveryProgress(task)">
      <span class="text-slate-300">·</span>
      <span class="whitespace-nowrap text-blue-500">{{ taskDeliveryProgress(task) }}</span>
    </template>
    <template v-else-if="taskStage(task)">
      <span class="text-slate-300">·</span>
      <span class="whitespace-nowrap text-[var(--n-text-color-2)]">{{ taskStage(task) }}</span>
    </template>
    <template v-if="taskGitMeta(task)">
      <span class="text-slate-300">·</span>
      <Icon
        :name="taskGitMeta(task)!.icon"
        :size="13"
        class="shrink-0"
        :class="taskGitMeta(task)!.color"
      />
    </template>
    <template v-if="task.summary.hasDiff || task.summary.repositories?.length">
      <span class="text-slate-300">·</span>
      <CodeTaskRepositoryPopover
			:project-id="task.projectId"
			:session-id="task.sessionId"
			:repositories="task.summary.repositories || []"
			:branch="task.summary.branch"
			:additions="task.summary.additions"
			:deletions="task.summary.deletions"
			:changed-files="task.summary.changedFiles"
		/>
    </template>
    <template v-if="task.summary.durationMs > 0">
      <span class="text-slate-300">·</span>
      <span class="whitespace-nowrap">{{ formatTaskDuration(task.summary.durationMs) }}</span>
    </template>
    <template v-if="task.summary.totalTokens > 0">
      <span class="text-slate-300">·</span>
      <span class="whitespace-nowrap text-blue-500">
        {{ t("code.taskTokens", { count: formatTaskTokens(task.summary.totalTokens) }) }}
      </span>
    </template>
    <template v-if="taskTokenStatus(task)">
      <span class="text-slate-300">·</span>
      <span
        class="whitespace-nowrap"
        :class="taskTokenStatus(task)!.color"
      >
        {{ taskTokenStatus(task)!.label }}
      </span>
    </template>
    <slot name="trail" />
  </div>
</template>
