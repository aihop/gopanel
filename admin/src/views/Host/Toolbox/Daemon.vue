<template>
  <div class="mt-2 space-y-8">
    <div class="bg-white/86 rounded-[28px] border border-blue-100/80 p-8 shadow-[0_28px_80px_rgba(15,23,42,0.08)] backdrop-blur-xl sm:p-10">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-4">
          <div class="inline-flex rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
            Daemon Center
          </div>
          <div class="text-4xl font-semibold leading-[1.08] fg-base-100 sm:text-5xl">守护进程</div>
          <div class="text-base leading-8 text-slate-500 sm:text-lg">
            统一管理常驻进程、配置文件与运行状态，支持批量启动、停止、重载与日志查看
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-4 lg:min-w-[560px]">
          <div
            v-for="item in summaryCards"
            :key="item.label"
            class="rounded-[24px] border border-slate-200 bg-slate-50/80 p-5"
          >
            <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
              {{ item.label }}
            </div>
            <div class="mt-3 text-xl font-semibold fg-base-100">{{ item.value }}</div>
            <div class="mt-2 text-sm leading-6 text-slate-500">{{ item.desc }}</div>
          </div>
        </div>
      </div>

      <div class="mt-8 flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div class="flex flex-wrap items-center gap-3">
          <n-tag
            round
            :bordered="false"
            :type="isRunning ? 'success' : 'warning'"
            class="!px-4 !py-2"
          >
            服务状态 · {{ isRunning ? "已启动" : "未启动" }}
          </n-tag>
          <n-tag
            round
            :bordered="false"
            type="info"
          >
            当前视图 · {{ activeTab === "list" ? "进程列表" : "配置文件" }}
          </n-tag>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <n-button
            type="primary"
            ghost
            class="!rounded-[18px] px-5"
            @click="handleDaemonStart"
          >
            全部启动
          </n-button>
          <n-button
            type="error"
            ghost
            class="!rounded-[18px] px-5"
            @click="handleDaemonStop"
          >
            全部停止
          </n-button>
          <n-button
            ghost
            class="!rounded-[18px] px-5"
            @click="refreshAll"
          >刷新</n-button>
          <n-button
            type="primary"
            class="!rounded-[18px] px-5 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
            @click="openPost()"
          >
            创建守护进程
          </n-button>
        </div>
      </div>

      <n-alert
        v-if="!agentStatus.online"
        class="mt-6"
        type="warning"
        :show-icon="true"
        title="Agent 未初始化"
      >
        <div class="text-sm leading-6">
          <div>gp-agent 未启动或未安装，守护进程功能暂不可用。</div>
          <div
            v-if="agentStatus.error"
            class="mt-1 text-slate-500"
          >{{ agentStatus.error }}</div>
          <n-space class="mt-3">
            <n-button
              size="small"
              type="primary"
              :loading="ensuringAgent"
              @click="ensureAgent"
            >一键初始化</n-button>
          </n-space>
        </div>
      </n-alert>
    </div>

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
      <OpDialog
        ref="opDialogRef"
        @search="handleEnsureFinished"
      />
    </div>
  </div>
</template>
<script setup lang="ts">
import { useTable } from "@/composables/useTable"
import DaemonPost from "./components/DaemonPost.vue"
import {
	NButton,
	NTag,
	NDrawer,
	NDrawerContent,
	NInput,
	useMessage,
	useDialog,
	NTabs,
	NTabPane,
	NSpace,
	NAlert
} from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import type { Ref } from "vue"
import { computed, h, onMounted, reactive, ref, watch } from "vue"
import DaemonConfigFile from "./components/DaemonConfigFile.vue"
import DaemonProcessLog from "./components/DaemonProcessLog.vue"
import { copyText } from "@/utils/util"
import { AgentEnsureAPI, AgentStatusAPI } from "@/api/modules/agent"
import { useAuthStore } from "@/store/auth"
import OpDialog from "@/components/OpDialog.vue"
import {
	DaemonStatus,
	DaemonStart,
	DaemonStop,
	DaemonReload,
	daemonProcessListAPI,
	DaemonProcessStart,
	DaemonProcessStop,
	DaemonProcessReload,
	DaemonConfigDelete,
	DaemonConfigAdd,
	DaemonConfigUpdate,
} from "@/api/modules/daemon"
const dialog = useDialog()
const stopConfirmInput = ref("")
const deleteConfirmInput = ref("")

const params = reactive({
	listAPI: daemonProcessListAPI,
	params: {
		wheres: []
	}
})
const { list, pages, curPage, pageSize, getList, loading, getData, onPageSizeChange, onPageChange, pageSizeOptions } =
	useTable(params)

const renderTagWithCopy = (value?: string) => {
	const content = value || "-"
	return h("div", { class: "flex items-center gap-2" }, [
		h(
			NTag,
			{
				size: "small",
				type: value ? "info" : "default",
				bordered: false,
				class: "max-w-[380px] truncate"
			},
			{ default: () => content }
		),
		h(
			NButton,
			{
				text: true,
				size: "tiny",
				type: "primary",
				disabled: !value,
				onClick: (e: MouseEvent) => {
					e?.stopPropagation?.()
					if (!value) return
					copyText(value)
				}
			},
			"复制"
		)
	])
}
const columns: DataTableColumns<any> = [
	{ title: "名称", key: "name" },
	{ title: "pid", key: "pid" },

	{ title: "启动用户", key: "config.user" },
	{
		title: "运行目录",
		key: "config.directory",
		render: (row: any) => renderTagWithCopy(row?.config?.directory)
	},
	{
		title: "启动命令",
		key: "config.command",
		render: (row: any) => renderTagWithCopy(row?.config?.command)
	},
	{
		title: "进程数量",
		key: "config.numprocs",
		render: (row: any) => {
			if (row.config.numprocs) {
				return row.config.numprocs
			} else {
				return 1
			}
		}
	},
	{
		title: "状态",
		key: "statename",
		render: (row: any) => h(NTag, { text: true }, row.statename)
		// statename = Stopped,Running,Exited
	},
	{
		title: "操作",
		key: "actions",
		align: "center" as const,
		fixed:"right",
		render: (row: any) =>
			h("div", { class: "flex items-center gap-2 justify-center" }, [
				h(
					NButton,
					{
						text: true,
						type: "primary",
						onClick: () => {
							openPost(row)
						}
					},
					"编辑"
				),
				h(
					NButton,
					{ text: true, type: "primary", onClick: () => DaemonProcessLogRef.value?.open(row) },
					"日志"
				),
				h(
					NButton,
					{
						text: true,
						type: "primary",
						disabled: row.statename == "Running",
						onClick: () => {
							handleProcessStart(row.name)
						}
					},
					"启动"
				),
				h(
					NButton,
					{
						text: true,
						type: "primary",
						disabled: row.statename == "Stopped" || row.statename == "Exited",
						onClick: () => {
							handleProcessStop(row.name)
						}
					},
					"停止"
				),
				h(
					NButton,
					{
						text: true,
						type: "primary",
						disabled: row.statename == "Stopped" || row.statename == "Exited",
						onClick: () => {
							handleProcessReload(row.name)
						}
					},
					"重启"
				),
				h(
					NButton,
					{
						text: true,
						type: "primary",
						onClick: () => {
							handleProcessDelete(row.name)
						}
					},
					"删除"
				)
			])
	}
]

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

const handleDaemonStart = async () => {
	loading.value = true
	await DaemonReload()
	await DaemonStart()
	getData()
}

const handleDaemonStop = async () => {
	stopConfirmInput.value = ""
	dialog.warning({
		title: "确认全部停止",
		content: () =>
			h("div", [
				h("div", { class: "mb-4 text-gray-500" }, "此操作将停止所有进程，输入“全部停止”以确认"),
				h(NInput, {
					value: stopConfirmInput.value,
					placeholder: "请输入：全部停止",
					"onUpdate:value": v => (stopConfirmInput.value = v)
				})
			]),
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			if (stopConfirmInput.value !== "全部停止") return false
			loading.value = true
			await DaemonStop()
			getData()
		}
	})
}

const handleProcessStart = async (name: string) => {
	loading.value = true
	await DaemonProcessStart(name)
	getData()
}

const handleProcessStop = async (name: string) => {
	loading.value = true
	await DaemonProcessStop(name)
	getData()
}

const handleProcessReload = async (name: string) => {
	loading.value = true
	await DaemonProcessReload(name)
	getData()
}

const handleProcessDelete = async (name: string) => {
	deleteConfirmInput.value = ""
	dialog.info({
		title: "确认删除",
		content: () =>
			h("div", [
				h(
					"div",
					{
						class: "mb-4"
					},
					"此操作将立即删除该进程，输入“立即删除”以确认"
				),
				h(NInput, {
					value: deleteConfirmInput.value,
					placeholder: "请输入：立即删除",
					"onUpdate:value": v => (deleteConfirmInput.value = v)
				})
			]),
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			if (deleteConfirmInput.value !== "立即删除") return false
			loading.value = true
			await DaemonProcessStop(name)
			await DaemonConfigDelete({ names: [name] })
			await DaemonReload()
			getData()
		}
	})
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

const authStore = useAuthStore()
const opDialogRef = ref<InstanceType<typeof OpDialog> | null>(null)
const ensuringAgent = ref(false)
const agentStatus = ref<{ online: boolean; error?: string }>({ online: true })

const fetchAgentStatus = async () => {
	try {
		const res = await AgentStatusAPI()
		agentStatus.value = {
			online: !!res?.data?.online,
			error: res?.data?.error
		}
	} catch (e: any) {
		agentStatus.value = { online: false, error: e?.message || "获取 Agent 状态失败" }
	}
}

const ensureAgent = async () => {
	if (ensuringAgent.value) return
	ensuringAgent.value = true
	try {
		const res = await AgentEnsureAPI()
		const log = res?.data?.log
		const token = authStore.getAuth() || authStore.auth || ""
		if (log) {
			opDialogRef.value?.acceptParams({
				title: "初始化 Agent",
				sseUrl: `/api/agent/ensure/logs?log=${encodeURIComponent(log)}&token=${encodeURIComponent(token)}`
			})
		}
	} catch (e) {
	} finally {
		ensuringAgent.value = false
	}
}

const handleEnsureFinished = () => {
	fetchAgentStatus()
	refreshAll()
}

const postConfirm = async (data: any, loading: Ref<boolean>) => {
	loading.value = true
	if (data.config) {
		await DaemonConfigUpdate(data)	
	} else {
		await DaemonConfigAdd(data)
	}
	await DaemonReload()
	getData()
	DaemonPostModel.value.close()
}

onMounted(() => {
	fetchAgentStatus()
	refreshAll()
})
</script>
