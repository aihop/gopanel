<template>
  <div class="ai-workspace-root relative flex h-full min-h-[calc(100vh-130px)] w-full flex-col overflow-hidden rounded-[28px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]">
    <n-layout
      has-sider
      class="h-full flex-1 !bg-transparent"
      style="width: 100%"
    >
      <!-- 左侧边栏：该组内的历史任务 -->
      <n-layout-sider
        collapse-mode="width"
        :collapsed-width="0"
        :width="320"
        show-trigger="bar"
        class="ai-workspace-sider !bg-[rgba(248,250,252,0.75)] backdrop-blur-sm"
        style="height: 100%"
      >
        <div class="ai-workspace-sider-inner flex h-full flex-col border-r border-slate-200/80">
          <div class="ai-workspace-sider-header border-b border-slate-200/80 p-5">
            <div class="ai-workspace-sider-card rounded-[24px] border border-slate-200/80 bg-white/90 p-4 shadow-sm">
              <div class="flex flex-col gap-4">
                <div
                  class="flex cursor-pointer items-center gap-3 rounded-2xl px-1 py-1 transition-opacity hover:opacity-80"
                  @click="backToLobby"
                >
                  <n-button
                    quaternary
                    circle
                    size="small"
                    class="!bg-slate-100"
                  >
                    <template #icon>←</template>
                  </n-button>
                  <div class="min-w-0">
                    <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">AI Workspace</div>
                    <div class="truncate text-sm font-semibold text-[var(--n-text-color)]">{{ groupInfo ? groupInfo.name : '项目组' }}</div>
                  </div>
                </div>
                <n-button
                  type="primary"
                  block
                  @click="createNewTask"
                  class="!h-11 !rounded-[16px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
                >
                  <template #icon>
                    <AddIcon />
                  </template>
                  发起新对话
                </n-button>
              </div>
            </div>

            <div class="flex items-center justify-between px-5 pt-5">
              <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">历史任务</div>
              <div class="text-xs text-slate-400">{{ tasks.length }} 条</div>
            </div>

            <n-scrollbar class="flex-1 px-4 pb-4 pt-4">
              <div
                v-if="tasks.length === 0"
                class="ai-workspace-task-empty flex min-h-[240px] items-center justify-center rounded-[22px] border border-dashed border-slate-200 bg-white/70"
              >
                <n-empty description="该组暂无对话历史" />
              </div>

              <div
                v-else
                class="space-y-3"
              >
                <div
                  v-for="task in tasks"
                  :key="task.id"
                  class="ai-workspace-task-card group/task relative flex cursor-pointer items-start justify-between gap-3 rounded-[22px] border border-slate-200/80 bg-white/90 p-4 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-200 hover:shadow-[0_16px_30px_rgba(15,23,42,0.08)]"
                  :class="
                    currentTaskId === task.id
                      ? 'ai-workspace-task-card--active !border-blue-200 !bg-[linear-gradient(180deg,rgba(239,246,255,0.95),rgba(255,255,255,0.96))] shadow-[0_18px_34px_rgba(37,99,235,0.12)]'
                      : ''
                  "
                  @click="selectTask(task)"
                >
                  <div class="min-w-0 flex-1">
                    <div
                      class="truncate text-sm font-semibold text-slate-800"
                      :title="task.title"
                    >
                      {{ task.title }}
                    </div>
                    <div class="mt-3 flex items-center gap-2">
                      <n-tag
                        size="small"
                        type="success"
                        round
                        :bordered="false"
                      >
                        {{ task.agentName || 'trae' }}
                      </n-tag>
                      <span class="text-xs text-slate-400">{{ new Date(task.createdAt).toLocaleDateString() }}</span>
                    </div>
                  </div>

                  <div
                    class="opacity-100 transition-opacity md:opacity-0 md:group-hover/task:opacity-100"
                    @click.stop
                  >
                    <n-dropdown
                      trigger="click"
                      :options="taskActionOptions"
                      @select="(key) => handleTaskAction(key, task)"
                    >
                      <n-button
                        quaternary
                        circle
                        size="small"
                        class="ai-workspace-task-btn !bg-slate-100"
                      >
                        <template #icon>
                          <MoreIcon />
                        </template>
                      </n-button>
                    </n-dropdown>
                  </div>
                </div>
              </div>
            </n-scrollbar>
          </div>
        </div>
      </n-layout-sider>

      <!-- 右侧：终端工作区 -->
      <n-layout-content content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
        <div class="ai-workspace-content-panel flex h-full min-h-0 flex-1 flex-col bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.08),transparent_28%)] p-4 md:p-5">
          <div class="ai-workspace-session-bar mb-4 flex items-center justify-between rounded-[22px] border border-slate-200/80 bg-white/85 px-4 py-3 shadow-sm">
            <div class="min-w-0">
              <div class="text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">Current Session</div>
              <div class="truncate text-sm font-semibold text-slate-800">
                {{ isNewTask ? '新的 AI 对话' : currentTaskId !== null ? '正在查看历史任务' : '请选择一个任务开始协作' }}
              </div>
            </div>
            <n-button
              type="primary"
              secondary
              class="!rounded-[14px]"
              @click="createNewTask"
            >
              新对话
            </n-button>
          </div>

          <div class="ai-workspace-terminal-wrap min-h-0 flex-1 overflow-hidden rounded-[26px] border border-slate-200/80 bg-[#0f172a] shadow-[0_24px_50px_rgba(15,23,42,0.18)]">
            <AgentTerminal
              v-if="currentTaskId !== null || isNewTask"
              :key="terminalKey"
              :task-id="currentTaskId"
              :group-id="currentGroupId"
              :default-agent="isNewTask ? 'trae' : ''"
              @task-created="handleTaskCreated"
            />
            <div
              v-else
              class="ai-workspace-empty-bg flex h-full flex-1 items-center justify-center bg-[linear-gradient(180deg,#ffffff,#f8fafc)]"
            >
              <n-empty
                description="请在左侧选择一个历史任务，或发起新对话"
                size="large"
              >
                <template #extra>
                  <n-button
                    type="primary"
                    class="!rounded-[16px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
                    @click="createNewTask"
                  >发起新对话</n-button>
                </template>
              </n-empty>
            </div>
          </div>
        </div>
      </n-layout-content>
    </n-layout>

    <!-- 弹窗：重命名任务 -->
    <n-modal
      v-model:show="showRenameModal"
      preset="dialog"
      title="重命名任务"
    >
      <n-input
        v-model:value="editingTaskTitle"
        placeholder="请输入新的任务名称"
        @keyup.enter="submitRename"
        class="mt-4"
      />
      <template #action>
        <n-button @click="showRenameModal = false">取消</n-button>
        <n-button
          type="primary"
          @click="submitRename"
          :loading="renaming"
        >确定</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, useDialog } from 'naive-ui'
import AgentTerminal from './components/AgentTerminal.vue'
import { getAITasks, updateAITask, deleteAITask, getAIGroups } from '@/api/modules/ai_agent'
import type { AITask, AIGroup } from '@/api/interface/ai_agent'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const AddIcon = () => '+'
const MoreIcon = () => '...'

const currentGroupId = computed(() => Number(route.params.id))
const groupInfo = ref<AIGroup | null>(null)

// 拉取当前组的信息
const fetchGroupInfo = async () => {
  try {
    const res = await getAIGroups({ page: 1, limit: 50 })
    if (res.code === 0) {
      groupInfo.value = res.data.items.find(g => g.id === currentGroupId.value) || null
    }
  } catch (error) {
    console.error('获取组信息失败:', error)
  }
}

const backToLobby = () => {
  router.push('/ai/index')
}

// === 任务与终端逻辑 ===
const tasks = ref<AITask[]>([])
const currentTaskId = ref<number | null>(null)
const isNewTask = ref(false)
const terminalKey = ref(0)

const fetchTasks = async () => {
  if (!currentGroupId.value) return
  try {
    const res = await getAITasks({ page: 1, limit: 50, projectId: currentGroupId.value })
    if (res.code === 0) {
      tasks.value = res.data.items || []
    }
  } catch (error) {
    console.error('获取历史任务失败:', error)
  }
}

onMounted(() => {
  fetchGroupInfo()
  fetchTasks()
})

watch(
  () => route.params.id,
  (newId) => {
    if (newId && route.name === 'AIAgent-Group') {
      currentTaskId.value = null
      isNewTask.value = false
      fetchGroupInfo()
      fetchTasks()
    }
  }
)

const createNewTask = () => {
  currentTaskId.value = null
  isNewTask.value = true
  terminalKey.value++ 
}

const selectTask = (task: AITask) => {
  if (currentTaskId.value === task.id) return
  currentTaskId.value = task.id
  isNewTask.value = false
  terminalKey.value++ 
  
  // 可以考虑在这里把 task_id 同步到 URL query 中以便分享更深的一层，
  // 例如：router.replace({ query: { task_id: task.id } })
}

const handleTaskCreated = (taskId: number) => {
  currentTaskId.value = taskId
  isNewTask.value = false
  fetchTasks() 
}

const taskActionOptions = [
  { label: '重命名', key: 'rename' },
  { label: '删除', key: 'delete', style: 'color: red;' }
]

const showRenameModal = ref(false)
const editingTaskId = ref<number | null>(null)
const editingTaskTitle = ref('')
const renaming = ref(false)

const handleTaskAction = (key: string, task: AITask) => {
  if (key === 'rename') {
    editingTaskId.value = task.id
    editingTaskTitle.value = task.title
    showRenameModal.value = true
  } else if (key === 'delete') {
    dialog.warning({
      title: '删除任务',
      content: `确定要删除任务 "${task.title}" 吗？此操作将同时删除所有历史对话记录且无法恢复。`,
      positiveText: '确定删除',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          const res = await deleteAITask(task.id)
          if (res.code === 0) {
            message.success('删除成功')
            if (currentTaskId.value === task.id) {
              isNewTask.value = false
              currentTaskId.value = null
            }
            fetchTasks()
          }
        } catch (error) {
          message.error('删除失败')
        }
      }
    })
  }
}

const submitRename = async () => {
  if (!editingTaskTitle.value.trim() || !editingTaskId.value) return
  renaming.value = true
  try {
    const res = await updateAITask(editingTaskId.value, editingTaskTitle.value)
    if (res.code === 0) {
      message.success('重命名成功')
      showRenameModal.value = false
      fetchTasks()
    }
  } finally {
    renaming.value = false
  }
}
</script>

<style scoped>
.theme-dark .ai-workspace-root {
  border-color: color-mix(in srgb, var(--border-color) 70%, transparent);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--bg-default-color) 98%, white), color-mix(in srgb, var(--bg-secondary-color) 92%, transparent));
  box-shadow: 0 18px 45px rgba(0, 0, 0, 0.25);
}

.theme-dark .ai-workspace-sider {
  background: color-mix(in srgb, var(--bg-secondary-color) 75%, transparent) !important;
}

.theme-dark .ai-workspace-sider-inner {
  border-right-color: color-mix(in srgb, var(--border-color) 80%, transparent);
}

.theme-dark .ai-workspace-sider-header {
  border-bottom-color: color-mix(in srgb, var(--border-color) 80%, transparent);
}

.theme-dark .ai-workspace-sider-card {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent);
  background-color: color-mix(in srgb, var(--bg-default-color) 90%, transparent);
}

.theme-dark .ai-workspace-task-empty {
  border-color: color-mix(in srgb, var(--border-color) 60%, transparent) !important;
  background-color: color-mix(in srgb, var(--bg-default-color) 70%, transparent) !important;
}

.theme-dark .ai-workspace-task-card {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
  background-color: color-mix(in srgb, var(--bg-default-color) 90%, transparent) !important;
}

.theme-dark .ai-workspace-task-card--active {
  border-color: color-mix(in srgb, var(--primary-color) 50%, transparent) !important;
  background: linear-gradient(180deg, color-mix(in srgb, var(--primary-color) 18%, var(--bg-default-color)), color-mix(in srgb, var(--bg-default-color) 96%, transparent)) !important;
}

.theme-dark .ai-workspace-task-btn {
  background-color: color-mix(in srgb, var(--fg-secondary-color) 15%, transparent) !important;
}

.theme-dark .ai-workspace-content-panel {
  background: radial-gradient(circle at top right, color-mix(in srgb, var(--primary-color) 8%, transparent), transparent 28%);
}

.theme-dark .ai-workspace-session-bar {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent);
  background-color: color-mix(in srgb, var(--bg-default-color) 85%, transparent);
}

.theme-dark .ai-workspace-terminal-wrap {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent);
}

.theme-dark .ai-workspace-empty-bg {
  background: linear-gradient(180deg, var(--bg-default-color), var(--bg-secondary-color));
}
</style>
