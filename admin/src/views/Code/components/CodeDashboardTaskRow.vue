<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { codeTaskTimestamp } from "../codeDashboardBuckets"
import CodeProjectIdentity from "./CodeProjectIdentity.vue"
import CodeTaskAgentSnippet from "./CodeTaskAgentSnippet.vue"
import CodeTaskFocusMarker from "./CodeTaskFocusMarker.vue"
import CodeTaskMetaLine from "./CodeTaskMetaLine.vue"
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

// 选中行要当详情头用，所以多给两条终端上面看不到的信息。
const branchLabel = computed(() => props.task.summary.branch || "")
const executorLabel = computed(() =>
	[props.task.summary.executor, props.task.summary.model].filter(Boolean).join(" · "),
)
</script>

<template>
  <div
    class="dashboard-task-row group/row relative mb-1 flex cursor-pointer items-stretch gap-3 rounded-lg py-3 ml-5 pr-3.5 transition-colors"
    :class="selected ? 'dashboard-task-row--selected' : ''"
    @click="emit('open', task)"
  >
    <CodeTaskFocusMarker :active="selected" />
    <div class="min-w-0 flex-1">
      <div class="flex min-w-0 items-center justify-between gap-2">
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <Icon
            v-if="task.agentName === 'terminal'"
            name="mdi:console-line"
            :size="14"
            class="shrink-0 text-slate-500"
          />
          <CodeProjectIdentity
            v-if="showProject !== false"
            class="max-w-[140px] shrink-0 text-[11px] text-[var(--n-text-color-3)]"
            :project-id="task.projectId"
            :name="projectName || t('code.projectFallback')"
          />
          <span
            class="min-w-0 truncate text-sm font-semibold text-[var(--n-text-color)]"
            :title="task.title"
          >
            {{ task.title }}
          </span>
          <TaskApprovalAction
            class="shrink-0"
            :task="task"
            @approved="emit('refresh')"
          />
        </div>
        <!-- self-start：行是 items-stretch（竖线要通高），这一簇仍然跟标题对齐 -->
        <div class="flex shrink-0 self-start items-center gap-1 pt-0.5 text-[11px] text-[var(--n-text-color-3)]">
          <!-- 操作只在悬停时出现：每行都常驻两个按钮会把列表搞得很吵 -->
          <n-button
            quaternary
            circle
            size="tiny"
            :title="t('code.detailOpenWorkspace')"
            class="opacity-0 transition-opacity group-hover/row:opacity-100"
            @click.stop="emit('openWorkspace', task)"
          >
            <template #icon>
              <Icon
                name="mdi:open-in-new"
                :size="13"
              />
            </template>
          </n-button>
          <n-button
            quaternary
            circle
            size="tiny"
            :loading="archiving"
            :title="archived ? t('code.taskUnarchive') : t('code.taskArchive')"
            class="opacity-0 transition-opacity group-hover/row:opacity-100"
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
          <span>{{ timeLabel }}</span>
        </div>
      </div>
      <CodeTaskMetaLine
        :task="task"
        class="mt-1.5"
      />

      <!--
        选中行就是这条任务的详情头 —— 终端上面不再重复一遍。
        分支/执行器只在选中时露出来，未选中的行保持两行，列表不会变松散。
        两个都没有就整行不渲染，避免选中时多出一条空白。
      -->
      <div
        v-if="selected && (branchLabel || executorLabel)"
        class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-[var(--n-text-color-3)]"
      >
        <span
          v-if="branchLabel"
          class="flex min-w-0 items-center gap-1"
          :title="branchLabel"
        >
          <Icon
            name="mdi:source-branch"
            :size="12"
            class="shrink-0"
          />
          <span class="truncate">{{ branchLabel }}</span>
        </span>
        <span
          v-if="executorLabel"
          class="truncate"
        >
          {{ executorLabel }}
        </span>
      </div>

      <CodeTaskAgentSnippet
        v-if="selected"
        :task="task"
      />
    </div>
  </div>
</template>

<style scoped>
/*
	选中态跟工作台侧栏（ProjectTaskSidebar）同一套语言：左侧竖线 + 淡底。
	终端上面已经没有头部了，「现在看的是哪条」全靠这一行说清楚，
	所以竖线是主信号，底色只是陪衬 —— 不加内描边，免得跟竖线抢注意力。
*/
/* 悬停时先给一条淡的，提示「点这里会选中」 */
.dashboard-task-row:hover :deep(.code-task-focus-marker:not(.code-task-focus-marker--on)) {
	background: color-mix(in srgb, var(--primary-color) 28%, transparent);
}

.dashboard-task-row--selected {
	background: color-mix(in srgb, var(--primary-color) 10%, transparent);
}

/*
	:not(--selected) 是必要的：`.row:hover` 的优先级(0,2,0)高于 `.row--selected`(0,1,0)，
	不排除的话鼠标划过选中行反而会把它洗淡，正好和「锁住焦点」相反。
*/
.dashboard-task-row:not(.dashboard-task-row--selected):hover {
	background: color-mix(in srgb, var(--primary-color) 7%, transparent);
}

</style>
