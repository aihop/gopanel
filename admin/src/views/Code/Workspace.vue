<template>
	<div
		class="ai-workspace-root relative flex h-full min-h-[calc(100vh-130px)] w-full flex-col overflow-hidden rounded-[28px] border border-slate-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.92))] shadow-[0_18px_45px_rgba(15,23,42,0.08)]"
	>
		<n-layout has-sider class="h-full flex-1 !bg-transparent" style="width: 100%">
			<!-- 左侧边栏：该项目内的历史任务 -->
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
						<div
							class="ai-workspace-sider-card rounded-[24px] border border-slate-200/80 bg-white/90 p-4 shadow-sm"
						>
							<div class="flex flex-col gap-4">
								<div
									class="flex cursor-pointer items-center gap-3 rounded-2xl px-1 py-1 transition-opacity hover:opacity-80"
									@click="backToLobby"
								>
									<n-button quaternary circle size="small" class="!bg-slate-100">
										<template #icon>←</template>
									</n-button>
									<div class="min-w-0">
										<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
											{{ $t("code.workspace") }}
										</div>
										<div class="truncate text-sm font-semibold text-[var(--n-text-color)]">
											{{ groupInfo ? groupInfo.name : t("code.projectFallback") }}
										</div>
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
					</div>

					<div class="mt-3 flex min-h-0 flex-1 flex-col overflow-hidden">
						<div class="flex items-center justify-between px-5 pb-2 pt-1">
							<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">历史任务</div>
							<div class="text-xs text-slate-400">{{ tasks.length }} 条</div>
						</div>

						<n-scrollbar class="ai-workspace-task-scrollbar min-h-0 flex-1">
							<div class="px-3 pb-4 pr-5">
							<div
								v-if="tasks.length === 0"
								class="flex min-h-[240px] items-center justify-center"
							>
								<n-empty :description="t('code.noProjectHistory')" />
							</div>

							<div v-else class="space-y-1">
								<div
									v-for="task in tasks"
									:key="task.id"
									class="ai-workspace-task-row group/task relative flex cursor-pointer items-start justify-between gap-3 rounded-xl px-3 py-2.5 transition-colors duration-200 hover:bg-slate-200/45"
									:class="currentTaskId === task.id ? 'ai-workspace-task-row--active bg-blue-50/80' : ''"
									@click="selectTask(task)"
								>
									<div class="min-w-0 flex-1">
										<div class="truncate text-sm font-semibold text-slate-800" :title="task.title">
											{{ task.title }}
										</div>
										<div class="mt-1.5 flex items-center gap-2">
											<n-tag size="small" type="success" round :bordered="false">
												{{ task.agentName || "terminal" }}
											</n-tag>
											<span class="text-xs text-slate-400">
												{{ new Date(task.createdAt).toLocaleDateString() }}
											</span>
										</div>
									</div>

									<div
										class="opacity-100 transition-opacity md:opacity-0 md:group-hover/task:opacity-100"
										@click.stop
									>
										<n-dropdown
											trigger="click"
											:options="taskActionOptions"
											@select="key => handleTaskAction(key, task)"
										>
											<n-button
												quaternary
												circle
												size="small"
												class="ai-workspace-task-btn !bg-transparent"
											>
												<template #icon>
													<MoreIcon />
												</template>
											</n-button>
										</n-dropdown>
									</div>
								</div>
							</div>
							</div>
						</n-scrollbar>
					</div>
				</div>
			</n-layout-sider>

			<!-- 右侧：终端工作区 -->
			<n-layout-content content-style="padding: 0; display: flex; flex-direction: column; height: 100%;">
				<div
					class="ai-workspace-content-panel flex h-full min-h-0 flex-1 flex-col bg-[radial-gradient(circle_at_top_right,rgba(59,130,246,0.08),transparent_28%)] p-4 md:p-5"
				>
					<div
						class="ai-workspace-session-bar mb-3 flex items-center justify-between px-2 py-1"
					>
						<div class="min-w-0">
							<div class="truncate text-sm font-semibold text-slate-800">
								{{ sessionLabel }}
							</div>
						</div>
						<div class="flex flex-wrap items-center justify-end gap-2">
							<CodeApprovalCenter :session-id="currentSessionId" @take-terminal="takeOverTerminal" />
							<SessionApprovalPolicy v-if="currentSessionId !== null" :session-id="currentSessionId" />
							<n-button-group v-if="currentSessionId !== null">
								<n-button size="small" :type="workMode === 'conversation' ? 'primary' : 'default'" @click="workMode = 'conversation'">
									{{ $t("code.conversationMode") }}
								</n-button>
								<n-button size="small" :type="workMode === 'terminal' ? 'primary' : 'default'" @click="workMode = 'terminal'">
									{{ $t("code.advancedTerminal") }}
								</n-button>
							</n-button-group>
							<n-button
								v-if="currentSessionId !== null || currentTaskId !== null"
								secondary
								class="!rounded-[14px]"
								@click="showHistoryDrawer = true"
							>
								{{ $t("code.conversationHistory") }}
							</n-button>
							<n-button type="primary" secondary class="!rounded-[14px]" @click="createNewTask">
								新对话
							</n-button>
						</div>
					</div>

					<div class="ai-workspace-terminal-wrap min-h-0 flex-1 overflow-hidden rounded-[26px] border border-slate-200/80 shadow-[0_24px_50px_rgba(15,23,42,0.18)]">
						<ConversationPanel
							v-if="currentSessionId !== null && workMode === 'conversation'"
							:key="`conversation-${currentSessionId}`"
							:session-id="currentSessionId"
							@task-created="handleTaskCreated"
							@show-history="showHistoryDrawer = true"
						/>
						<CodeTerminal
							v-else-if="currentTaskId !== null || currentSessionId !== null"
							:key="terminalKey"
							:task-id="currentTaskId"
							:session-id="currentSessionId"
							:auto-take-control="terminalTakeoverRequested"
							@task-created="handleTaskCreated"
						/>
						<div
							v-else
							class="ai-workspace-empty-bg flex h-full flex-1 items-center justify-center bg-[linear-gradient(180deg,#ffffff,#f8fafc)]"
						>
							<n-empty description="请在左侧选择一个历史任务，或发起新对话" size="large">
								<template #extra>
									<n-button
										type="primary"
										class="!rounded-[16px] shadow-[0_12px_28px_rgba(37,99,235,0.18)]"
										@click="createNewTask"
									>
										发起新对话
									</n-button>
								</template>
							</n-empty>
						</div>
					</div>
				</div>
			</n-layout-content>
		</n-layout>

		<!-- 弹窗：重命名任务 -->
		<n-modal v-model:show="showRenameModal" preset="dialog" title="重命名任务">
			<n-input
				v-model:value="editingTaskTitle"
				placeholder="请输入新的任务名称"
				@keyup.enter="submitRename"
				class="mt-4"
			/>
			<template #action>
				<n-button @click="showRenameModal = false">取消</n-button>
				<n-button type="primary" @click="submitRename" :loading="renaming">确定</n-button>
			</template>
		</n-modal>

		<NewSessionModal
			v-model:show="showNewSessionModal"
			:project-id="currentGroupId"
			@created="handleSessionCreated"
		/>
		<SessionHistoryDrawer
			v-model:show="showHistoryDrawer"
			:session-id="currentSessionId"
			:task-id="currentTaskId"
		/>
	</div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { useMessage, useDialog } from "naive-ui"
import CodeTerminal from "./components/CodeTerminal.vue"
import ConversationPanel from "./components/ConversationPanel.vue"
import NewSessionModal from "./components/NewSessionModal.vue"
import SessionApprovalPolicy from "./components/SessionApprovalPolicy.vue"
import CodeApprovalCenter from "./components/CodeApprovalCenter.vue"
import SessionHistoryDrawer from "./components/SessionHistoryDrawer.vue"
import { getAITasks, updateAITask, deleteAITask, getAIGroups } from "@/api/modules/code"
import type { AITask, AIGroup, CodeSession } from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n({ messages: codeProjectMessages })

const AddIcon = () => "+"
const MoreIcon = () => "..."

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
		console.error("获取组信息失败:", error)
	}
}

const backToLobby = () => {
	router.push("/code/index")
}

// === 任务与终端逻辑 ===
const tasks = ref<AITask[]>([])
const currentTaskId = ref<number | null>(null)
const currentSessionId = ref<number | null>(null)
const showNewSessionModal = ref(false)
const showHistoryDrawer = ref(false)
const terminalKey = ref(0)
const terminalTakeoverRequested = ref(false)
const workMode = ref<"conversation" | "terminal">("conversation")
const currentTask = computed(() => tasks.value.find(task => task.id === currentTaskId.value) || null)
const sessionLabel = computed(() => {
	if (currentTask.value?.title) return currentTask.value.title
	if (currentSessionId.value !== null) return t("code.newSession")
	return t("code.selectTaskToStart")
})

const fetchTasks = async () => {
	if (!currentGroupId.value) return
	try {
		const res = await getAITasks({ page: 1, limit: 50, projectId: currentGroupId.value })
		if (res.code === 0) {
			tasks.value = res.data.items || []
		}
	} catch (error) {
		console.error("获取历史任务失败:", error)
	}
}

onMounted(() => {
	fetchGroupInfo()
	fetchTasks()
})

watch(
	() => route.params.id,
	newId => {
		if (newId && route.name === "Code-Group") {
			currentTaskId.value = null
			currentSessionId.value = null
			showNewSessionModal.value = false
			showHistoryDrawer.value = false
			fetchGroupInfo()
			fetchTasks()
		}
	}
)

const createNewTask = () => {
	terminalTakeoverRequested.value = false
	showNewSessionModal.value = true
}

const takeOverTerminal = () => {
	if (currentSessionId.value === null) return
	terminalTakeoverRequested.value = true
	workMode.value = "terminal"
	terminalKey.value++
}

const handleSessionCreated = (session: CodeSession) => {
	terminalTakeoverRequested.value = false
	currentTaskId.value = null
	currentSessionId.value = session.id
	workMode.value = session.agentName === "codex" || session.agentName === "terminal" ? "terminal" : "conversation"
	terminalKey.value++
	void fetchTasks()
}

const selectTask = (task: AITask) => {
	if (currentTaskId.value === task.id && currentSessionId.value === null) return
	currentTaskId.value = task.id
	currentSessionId.value = task.sessionId || null
	terminalTakeoverRequested.value = false
	workMode.value = task.sessionId && !["terminal", "codex"].includes(task.agentName) ? "conversation" : "terminal"
	terminalKey.value++

	// 可以考虑在这里把 task_id 同步到 URL query 中以便分享更深的一层，
	// 例如：router.replace({ query: { task_id: task.id } })
}

const handleTaskCreated = (taskId: number) => {
	currentTaskId.value = taskId
	fetchTasks()
}

const taskActionOptions = [
	{ label: "重命名", key: "rename" },
	{ label: "删除", key: "delete", style: "color: red;" }
]

const showRenameModal = ref(false)
const editingTaskId = ref<number | null>(null)
const editingTaskTitle = ref("")
const renaming = ref(false)

const handleTaskAction = (key: string, task: AITask) => {
	if (key === "rename") {
		editingTaskId.value = task.id
		editingTaskTitle.value = task.title
		showRenameModal.value = true
	} else if (key === "delete") {
		dialog.warning({
			title: "删除任务",
			content: `确定要删除任务 "${task.title}" 吗？此操作将同时删除所有历史对话记录且无法恢复。`,
			positiveText: "确定删除",
			negativeText: "取消",
			onPositiveClick: async () => {
				try {
					const res = await deleteAITask(task.id)
					if (res.code === 0) {
						message.success("删除成功")
						if (currentTaskId.value === task.id) {
							currentTaskId.value = null
							currentSessionId.value = null
						}
						fetchTasks()
					}
				} catch (error) {
					message.error("删除失败")
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
			message.success("重命名成功")
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
	background: linear-gradient(
		180deg,
		color-mix(in srgb, var(--bg-default-color) 98%, white),
		color-mix(in srgb, var(--bg-secondary-color) 92%, transparent)
	);
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

.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) {
	right: 4px !important;
	width: 3px !important;
}

.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail__scrollbar) {
	width: 3px !important;
	background-color: rgba(148, 163, 184, 0.24) !important;
}

.ai-workspace-task-scrollbar :deep(.n-scrollbar-rail__scrollbar:hover) {
	background-color: rgba(148, 163, 184, 0.4) !important;
}

.theme-dark .ai-workspace-task-row:hover {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 10%, transparent) !important;
}

.theme-dark .ai-workspace-task-row--active {
	background-color: color-mix(in srgb, var(--primary-color) 14%, transparent) !important;
}

.theme-dark .ai-workspace-task-btn {
	background-color: color-mix(in srgb, var(--fg-secondary-color) 15%, transparent) !important;
}

.theme-dark .ai-workspace-content-panel {
	background: radial-gradient(
		circle at top right,
		color-mix(in srgb, var(--primary-color) 8%, transparent),
		transparent 28%
	);
}

.theme-dark .ai-workspace-terminal-wrap {
	border-color: color-mix(in srgb, var(--border-color) 80%, transparent);
}

.theme-dark .ai-workspace-empty-bg {
	background: linear-gradient(180deg, var(--bg-default-color), var(--bg-secondary-color));
}
</style>
