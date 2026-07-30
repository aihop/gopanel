<template>
  <div class="dmn-root mt-2 space-y-8">
    <DaemonOverviewPanel
      :summary-cards="summaryCards"
      :is-running="isRunning"
      :active-tab="activeTab"
      :agent-status="agentStatus"
      :agent-update="agentUpdate"
      :updating-agent="updatingAgent"
      @daemon-start="handleDaemonStart"
      @daemon-stop="handleDaemonStop"
      @refresh="refreshAll"
      @create="openPost()"
      @update-agent="confirmUpdateAgent"
    />

    <ControlPlaneStatusCard
      :status="controlPlaneStatus"
      :loading="controlPlaneLoading"
      :error="controlPlaneError"
      :repairing="ensuringAgent"
      @refresh="fetchControlPlaneStatus(true)"
      @details="controlPlaneModalVisible = true"
      @repair="ensureAgent"
    />

    <div class="rounded-[28px] border border-blue-100/80 bg-base-100 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)] sm:p-8">
      <div class="flex flex-col gap-5 border-b border-slate-100 pb-7 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-3">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Daemon Workspace</div>
          <div class="text-2xl font-semibold fg-base-100">进程与配置工作台</div>
          <div class="text-sm leading-7 text-slate-500">
            在同一工作区内切换查看进程列表与配置文件，执行启动、停止、重启、日志与删除等操作。
          </div>
        </div>
        <div class="rounded-full bg-slate-100 p-1">
          <div class="flex flex-wrap gap-1">
            <button
              type="button"
              class="rounded-full px-5 py-2 text-sm font-medium transition-all"
              :class="
								activeTab === 'list'
									? 'bg-base-100 text-blue-600 shadow-[0_10px_25px_rgba(15,23,42,0.08)]'
									: 'text-slate-500 hover:text-slate-700'
							"
              @click="activeTab = 'list'"
            >
              进程列表
            </button>
            <button
              type="button"
              class="rounded-full px-5 py-2 text-sm font-medium transition-all"
              :class="
								activeTab === 'config'
									? 'bg-base-100 text-blue-600 shadow-[0_10px_25px_rgba(15,23,42,0.08)]'
									: 'text-slate-500 hover:text-slate-700'
							"
              @click="activeTab = 'config'"
            >
              配置文件
            </button>
          </div>
        </div>
      </div>

      <div
        v-if="activeTab === 'list'"
        class="mt-8 rounded-[26px] border border-slate-100 bg-slate-50/75 p-4 sm:p-6"
      >
        <n-data-table
          :loading="loading"
          :columns="columns"
          :data="list"
          :pagination="false"
          :bordered="false"
          :scroll-x="900"
        />
        <div class="mt-5 flex justify-end">
          <n-pagination
            v-model:page="curPage"
            v-model:page-size="pageSize"
            :page-count="pages"
            :item-count="list.length < pageSize && curPage === 1 ? list.length : undefined"
            :page-sizes="pageSizeOptions"
            show-size-picker
            show-quick-jumper
            @update:page="onPageChange"
            @update:page-size="onPageSizeChange"
          >
            <template #prefix>共 {{ pages }} 页</template>
          </n-pagination>
        </div>
      </div>

      <div
        v-else
        class="mt-8 overflow-hidden rounded-[26px] border border-slate-100 bg-slate-50/75 p-2"
      >
        <DaemonConfigFile />
      </div>

      <DaemonPost
        ref="DaemonPostModel"
        @confirm="postConfirm"
      />
      <DaemonProcessLog ref="DaemonProcessLogRef"></DaemonProcessLog>
      <!-- search = 日志流结束(EOF)，cancel = 用户手动关掉弹窗；
           两种都要解锁按钮，否则手动关闭后按钮会一直转圈 -->
      <OpDialog
        ref="opDialogRef"
        @search="handleEnsureFinished"
        @cancel="handleEnsureFinished"
      />
    </div>

    <ControlPlaneRepairModal
      v-model:show="controlPlaneModalVisible"
      :status="controlPlaneStatus"
      :loading="controlPlaneLoading"
      :repairing="ensuringAgent"
      @recheck="handleControlPlaneRecheck"
      @repair="ensureAgent"
    />
  </div>
</template>
<script setup lang="ts">
import { useTable } from "@/composables/useTable"
import DaemonOverviewPanel from "./DaemonOverviewPanel.vue"
import ControlPlaneStatusCard from "./ControlPlaneStatusCard.vue"
import ControlPlaneRepairModal from "./ControlPlaneRepairModal.vue"
import DaemonPost from "./components/DaemonPost.vue"
import {
	useMessage,
	useDialog,
} from "naive-ui"
import type { Ref } from "vue"
import { computed, onMounted, reactive, ref, watch } from "vue"
import DaemonConfigFile from "./components/DaemonConfigFile.vue"
import DaemonProcessLog from "./components/DaemonProcessLog.vue"
import OpDialog from "@/components/OpDialog.vue"
import {
	DaemonStatus,
	daemonProcessListAPI,
} from "@/api/modules/daemon"
import { createDaemonColumns } from "./daemonTableColumns"
import { useDaemonActions } from "./useDaemonActions"
import { useDaemonAgentStatus } from "./useDaemonAgentStatus"
import { useControlPlaneStatus } from "./useControlPlaneStatus"
const dialog = useDialog()
void useMessage()

const params = reactive({
	listAPI: daemonProcessListAPI,
	params: {
		wheres: []
	}
})
const { list, pages, curPage, pageSize, getList, loading, getData, onPageSizeChange, onPageChange, pageSizeOptions } =
	useTable(params)

const daemonStatus = ref({ Statecode: 0, Statename: "" })

const isRunning = computed(() => daemonStatus.value?.Statename === "RUNNING")
const summaryCards = computed(() => {
	const items = Array.isArray(list.value) ? list.value : []
	const running = items.filter((item: any) => item?.statename === "Running").length
	const stopped = items.filter((item: any) => item?.statename === "Stopped" || item?.statename === "Exited").length
	return [
		{ label: "SERVICE", value: isRunning.value ? "在线" : "离线", desc: "守护进程主服务运行状态" },
		{ label: "TOTAL", value: items.length, desc: "当前登记的守护进程数量" },
		{ label: "RUNNING", value: running, desc: "正在运行中的守护进程" },
		{ label: "STOPPED", value: stopped, desc: "已停止或退出的守护进程" }
	]
})

const fetchDaemonStatus = async () => {
	try {
		const res = await DaemonStatus()
		if (res.data) {
			daemonStatus.value = res.data
		}
	} catch {}
}

const activeTab = ref("list")

const refreshAll = async () => {
	loading.value = true
	await fetchDaemonStatus()
	getData()
}

watch(activeTab, newVal => {
	if (newVal === "list") {
		refreshAll()
	}
})

const DaemonProcessLogRef = ref()

const DaemonPostModel = ref()
const openPost = (record?: any) => {
	DaemonPostModel.value.open(record)
}

const opDialogRef = ref<InstanceType<typeof OpDialog> | null>(null)
const controlPlaneModalVisible = ref(false)
const {
	controlPlaneStatus,
	controlPlaneLoading,
	controlPlaneError,
	fetchControlPlaneStatus
} = useControlPlaneStatus()
const {
	handleDaemonStart,
	handleDaemonStop,
	handleProcessStart,
	handleProcessStop,
	handleProcessReload,
	handleProcessDelete,
	postConfirm
} = useDaemonActions(dialog, loading, refreshAll, getData, DaemonPostModel)

const {
	ensuringAgent,
	updatingAgent,
	agentStatus,
	agentUpdate,
	fetchAgentStatus,
	checkAgentUpdate,
	ensureAgent,
	updateAgent,
	handleEnsureFinished: handleAgentOperationFinished
} = useDaemonAgentStatus(opDialogRef, () => {
	refreshAll()
	fetchControlPlaneStatus()
})

const handleEnsureFinished = () => {
	handleAgentOperationFinished()
}

const handleControlPlaneRecheck = async () => {
	await fetchControlPlaneStatus(true)
	if (controlPlaneStatus.value?.autoRepairable) ensureAgent()
}

// 更新 gp-agent 会替换二进制并重启 agent，先让用户确认再执行
function confirmUpdateAgent() {
	dialog.warning({
		title: "更新 gp-agent",
		content: `将把 gp-agent 从 v${agentUpdate.value.currentVersion || "?"} 更新到 v${agentUpdate.value.latestVersion || "?"}，更新过程中 agent 会重启，是否继续？`,
		positiveText: "开始更新",
		negativeText: "取消",
		onPositiveClick: () => updateAgent()
	})
}

const columns = createDaemonColumns({
	openPost,
	openLogs: (row: any) => DaemonProcessLogRef.value?.open(row),
	handleProcessStart,
	handleProcessStop,
	handleProcessReload,
	handleProcessDelete
})

onMounted(() => {
	fetchAgentStatus()
	fetchControlPlaneStatus()
	checkAgentUpdate()
	refreshAll()
})
</script>

<style>
.theme-dark .dmn-root .text-slate-500,
.theme-dark .dmn-root .text-slate-400 {
  color: var(--fg-secondary-color) !important;
}
.theme-dark .dmn-root .text-slate-600,
.theme-dark .dmn-root .text-slate-700,
.theme-dark .dmn-root .text-slate-900 {
  color: var(--fg-default-color) !important;
}
.theme-dark .dmn-root .border-slate-100 {
  border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .dmn-root .border-blue-100\/80 {
  border-color: color-mix(in srgb, var(--primary-color) 20%, transparent) !important;
}
.theme-dark .dmn-root .bg-slate-50\/75 {
  background-color: color-mix(in srgb, var(--bg-default-color) 75%, transparent) !important;
}
.theme-dark .dmn-root .bg-slate-100 {
  background-color: color-mix(in srgb, var(--bg-default-color) 92%, transparent) !important;
}
.theme-dark .dmn-root .hover\:text-slate-700:hover {
  color: var(--fg-default-color) !important;
}
</style>
