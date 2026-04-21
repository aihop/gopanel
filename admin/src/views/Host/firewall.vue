<template>
  <div>
    <div class="bg-white/86 rounded-[28px] border border-blue-100/80 p-8 shadow-[0_28px_80px_rgba(15,23,42,0.08)] backdrop-blur-xl sm:p-10">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-4">
          <div class="inline-flex rounded-full border border-blue-200 bg-blue-50 px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
            Firewall Center
          </div>
          <div class="text-4xl font-semibold leading-[1.08] fg-base-100 sm:text-5xl">防火墙</div>
          <div class="text-base leading-8 text-slate-500 sm:text-lg">
            开启防火墙后，系统将对所有入站和出站流量进行过滤，防止未授权访问
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-4 lg:min-w-[560px]">
          <div
            v-for="item in summaryCards"
            :key="item.label"
            class="rounded-[24px] border border-slate-200 bg-slate-50/80 p-5"
          >
            <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">{{ item.label }}</div>
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
            {{ base.name || "未检测" }} · {{ isRunning ? "已启动" : "未启动" }}
          </n-tag>
          <n-tag
            round
            :bordered="false"
            type="info"
          >版本：{{ base.version || "--" }}</n-tag>
          <n-tag
            round
            :bordered="false"
            type="default"
          >Ping：<span class="text-warning">待支持</span></n-tag>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <n-popconfirm
            v-if="!isRunning"
            @positive-click="changeOperation('start')"
          >
            <template #trigger>
              <n-button
                type="primary"
                class="!rounded-[18px] px-5 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
              >
                启动
              </n-button>
            </template>
            将会启动当前系统防火墙，是否继续？
          </n-popconfirm>
          <n-popconfirm
            v-else
            @positive-click="changeOperation('stop')"
          >
            <template #trigger>
              <n-button
                ghost
                type="warning"
                class="!rounded-[18px] px-5"
              >关闭</n-button>
            </template>
            系统防火墙关闭后，服务器将失去安全防护，是否继续？
          </n-popconfirm>
          <n-popconfirm @positive-click="changeOperation('restart')">
            <template #trigger>
              <n-button
                ghost
                class="!rounded-[18px] px-5"
              >重启</n-button>
            </template>
            将对当前防火墙执行重启操作，是否继续？
          </n-popconfirm>
          <n-button
            ghost
            class="!rounded-[18px] px-5"
            @click="refreshAll"
          >刷新</n-button>
        </div>
      </div>
    </div>

    <div class="mt-8 rounded-[28px] border border-blue-100/80 bg-base-100 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)] sm:p-8">
      <div class="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-3">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Rule Workspace</div>
          <div class="text-2xl font-semibold fg-base-100">规则工作台</div>
          <div class="text-sm leading-7 text-slate-500">
            支持端口规则、IP 规则和转发规则切换，按关键词与策略快速过滤，并对选中规则进行批量移除。
          </div>
        </div>
        <div class="flex flex-col gap-4 xl:min-w-[640px] xl:items-end">
          <div class="rounded-lg border border-slate-200 bg-slate-50/90 p-2 px-6">
            <n-tabs
              :value="ruleType"
              animated
              @update:value="handleTypeChange"
            >
              <n-tab name="port">端口规则</n-tab>
              <n-tab name="ip">IP 规则</n-tab>
              <n-tab name="forward">转发规则</n-tab>
            </n-tabs>
          </div>

          <div class="grid w-full gap-3 lg:grid-cols-[1.5fr_0.9fr_0.9fr_0.8fr_auto_auto]">
            <n-input
              :value="keyword"
              clearable
              placeholder="搜索端口、IP、协议或描述"
              class="filter-input"
              @update:value="handleKeywordChange"
              @keydown.enter="handleSearch"
            >
              <template #suffix>
                <Icon name="material-symbols:search" />
              </template>
            </n-input>
            <n-select
              v-if="ruleType !== 'forward'"
              :value="strategy"
              :options="strategyOptions"
              class="filter-select"
              @update:value="handleStrategyChange"
            />
            <div v-else></div>
            <n-select
              :value="status"
              :options="statusOptions"
              class="filter-select"
              @update:value="handleStatusChange"
            />
            <n-select
              :value="refreshSeconds"
              :options="refreshOptions"
              class="filter-select"
              @update:value="handleRefreshChange"
            />
            <n-button
              class="!rounded-[18px] px-6"
              @click="handleSearch"
            >筛选</n-button>
            <n-button
              type="primary"
              class="!rounded-[18px] px-6 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
              @click="addRule"
            >
              {{ createButtonText }}
            </n-button>
          </div>

          <div class="flex w-full flex-wrap items-center justify-between gap-3 rounded-[22px] border border-slate-200 bg-slate-50/90 px-5 py-4">
            <div class="text-sm leading-7 text-slate-500">
              当前视图：
              <span class="font-medium text-slate-700">{{ ruleTypeLabel }}</span>
              <span class="mx-2 text-slate-300">·</span>
              已选择
              <span class="font-semibold fg-base-100">{{ selectedRowKeys.length }}</span>
              条规则
            </div>
            <div class="flex flex-wrap items-center gap-3">
              <n-button
                ghost
                type="error"
                class="!rounded-[18px] px-5"
                :disabled="selectedRows.length === 0"
                @click="batchDelete"
              >
                批量删除
              </n-button>
              <n-tag
                round
                :bordered="false"
                type="warning"
              >Docker 映射端口可能不受 ufw 完全控制</n-tag>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-8 rounded-[26px] border border-slate-100 bg-slate-50/75 p-4 sm:p-6">
        <div class="mb-5 rounded-[20px] border border-amber-100 bg-amber-50/80 px-4 py-3 text-sm leading-7 text-amber-700">
          Linux 防火墙对 Docker
          端口映射存在天然限制，若端口来自容器编排，请优先在应用或已安装页面控制端口暴露策略。
        </div>
        <n-data-table
          :columns="columns"
          :data="rules"
          :loading="loading"
          :pagination="pagination"
          :remote="true"
          :row-key="getRuleRowKey"
          :checked-row-keys="selectedRowKeys"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
          @update:checked-row-keys="handleCheckedRowKeys"
        />
      </div>
    </div>

    <RulesDrawer
      ref="rulesDrawerRef"
      @save="handleDrawerSave"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref, type Ref } from "vue"
import { NButton, NTag, useDialog, useMessage } from "naive-ui"
import {
	hostsFirewallBaseAPI,
	hostsFirewallForwardAPI,
	hostsFirewallIPAPI,
	hostsFirewallOperateAPI,
	hostsFirewallPortAPI,
	hostsFirewallListAPI,
	hostsFirewallUpdateAddrAPI,
	hostsFirewallUpdatePortAPI
} from "@/api/host/firewall"
import { isSucc } from "@/utils/is"
import RulesDrawer from "./components/RulesDrawer.vue"

type RuleType = "port" | "ip" | "forward"

const message = useMessage()
const dialog = useDialog()
const rulesDrawerRef = ref()

const base = ref<any>({})
const loading = ref(false)
const rules = ref<any[]>([])
const ruleType = ref<RuleType>("port")
const keyword = ref("")
const strategy = ref("all")
const status = ref("all")
const refreshSeconds = ref<number>(0)
const selectedRowKeys = ref<string[]>([])
const refreshTimer = ref<number | null>(null)

const handleKeywordChange = (value: string) => {
	keyword.value = value
}

const handleStrategyChange = (value: string) => {
	strategy.value = value
}

const handleStatusChange = (value: string) => {
	status.value = value
}

const getRuleRowKey = (row: { id: string | number }) => row.id

const pagination = ref({
	page: 1,
	limit: 10,
	itemCount: 0,
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100]
})

const strategyOptions = [
	{ label: "全部策略", value: "all" },
	{ label: "允许", value: "accept" },
	{ label: "拒绝", value: "drop" }
]

const statusOptions = [
	{ label: "全部状态", value: "all" },
	{ label: "已使用", value: "used" },
	{ label: "未使用", value: "unused" },
	{ label: "未知", value: "unknown" }
]

const refreshOptions = [
	{ label: "不刷新", value: 0 },
	{ label: "5 秒 / 次", value: 5 },
	{ label: "10 秒 / 次", value: 10 },
	{ label: "30 秒 / 次", value: 30 },
	{ label: "60 秒 / 次", value: 60 },
	{ label: "120 秒 / 次", value: 120 },
	{ label: "300 秒 / 次", value: 300 }
]

const isRunning = computed(() => String(base.value?.status || "").toLowerCase() === "running")
const ruleTypeLabel = computed(() =>
	ruleType.value === "port" ? "端口规则" : ruleType.value === "ip" ? "IP 规则" : "转发规则"
)
const createButtonText = computed(() =>
	ruleType.value === "port" ? "创建端口规则" : ruleType.value === "ip" ? "创建 IP 规则" : "创建转发规则"
)

const selectedRows = computed(() => rules.value.filter(item => selectedRowKeys.value.includes(item.id)))

const summaryCards = computed(() => [
	{
		label: "Engine",
		value: base.value?.name || "未检测",
		desc: "当前系统正在使用的防火墙引擎"
	},
	{
		label: "Status",
		value: isRunning.value ? "运行中" : "未启动",
		desc: "实时反映当前防火墙服务状态"
	},
	{
		label: "Rules",
		value: `${pagination.value.itemCount}`,
		desc: `${ruleTypeLabel.value}总数`
	},
	{
		label: "Refresh",
		value: refreshSeconds.value === 0 ? "手动" : `${refreshSeconds.value}s`,
		desc: "当前列表自动刷新频率"
	}
])

const columns = computed<any[]>(() => {
	const commonActionColumn = {
		title: "操作",
		key: "actions",
		width: 160,
		render: (row: any) =>
			h("div", { class: "flex gap-2" }, [
				h(
					NButton,
					{
						size: "small",
						type: "primary",
						text: true,
						onClick: () => editRule(row)
					},
					{ default: () => "编辑" }
				),
				h(
					NButton,
					{
						size: "small",
						type: "error",
						text: true,
						onClick: () => confirmDelete(row)
					},
					{ default: () => "删除" }
				)
			])
	}

	if (ruleType.value === "ip") {
		return [
			{ type: "selection", width: 50 },
			{ title: "IP / 网段", key: "address", minWidth: 220 },
			{
				title: "策略",
				key: "strategy",
				width: 120,
				render: renderStrategy
			},
			{
				title: "协议族",
				key: "family",
				width: 120
			},
			{ title: "描述", key: "description", minWidth: 220 },
			commonActionColumn
		]
	}

	if (ruleType.value === "forward") {
		return [
			{ type: "selection", width: 50 },
			{ title: "协议", key: "protocol", width: 120 },
			{ title: "入口端口", key: "port", width: 140 },
			{ title: "目标 IP", key: "targetIP", minWidth: 180 },
			{ title: "目标端口", key: "targetPort", width: 140 },
			commonActionColumn
		]
	}

	return [
		{ type: "selection", width: 50 },
		{ title: "协议", key: "protocol", width: 120 },
		{ title: "端口", key: "port", width: 140 },
		{
			title: "策略",
			key: "strategy",
			width: 120,
			render: renderStrategy
		},
		{
			title: "状态",
			key: "usedStatus",
			width: 120,
			render: renderStatus
		},
		{ title: "协议族", key: "family", width: 120 },
		{ title: "描述", key: "description", minWidth: 220 },
		commonActionColumn
	]
})

function renderStrategy(row: any) {
	const isAccept = row.strategy === "accept"
	return h(
		NTag,
		{ type: isAccept ? "success" : "error", size: "small", bordered: false, round: true },
		{ default: () => (isAccept ? "允许" : "拒绝") }
	)
}

function renderStatus(row: any) {
	const map: Record<string, { text: string; type: "success" | "warning" | "default" }> = {
		inUsed: { text: "已使用", type: "warning" },
		unused: { text: "未使用", type: "success" },
		unknown: { text: "未知", type: "default" }
	}
	const statusItem = map[row.usedStatus] || map.unknown
	return h(
		NTag,
		{ type: statusItem.type, size: "small", bordered: false, round: true },
		{ default: () => statusItem.text }
	)
}

async function getBase() {
	try {
		const res: any = await hostsFirewallBaseAPI()
		if (isSucc(res.code)) {
			base.value = res.data || {}
			return
		}
	} catch (error: any) {}
}

async function getRulesList() {
	try {
		loading.value = true
		const res: any = await hostsFirewallListAPI({
			page: pagination.value.page,
			limit: pagination.value.limit,
			info: keyword.value,
			status: status.value,
			strategy: strategy.value,
			type: ruleType.value
		})
		if (isSucc(res.code)) {
			const items = (res.data?.items || []) as any[]
			rules.value = mergeDualStackRules(items)
			pagination.value.itemCount = res.data?.total || 0
			selectedRowKeys.value = []
			return
		}
		rules.value = []
		pagination.value.itemCount = 0
		selectedRowKeys.value = []
	} catch (error: any) {
		rules.value = []
		pagination.value.itemCount = 0
		selectedRowKeys.value = []
	} finally {
		loading.value = false
	}
}

function mergeDualStackRules(items: any[]) {
	if (ruleType.value !== "port" && ruleType.value !== "ip") return items
	const map = new Map<string, any>()
	for (const item of items) {
		const port = String(item?.port ?? "")
		const protocol = String(item?.protocol ?? "")
		const strategyValue = String(item?.strategy ?? "")
		const rawAddress = String(item?.address ?? "")
		const addressNormalized = rawAddress.trim().toLowerCase() === "anywhere" ? "" : rawAddress.trim()
		const description = String(item?.description ?? "")
		const usedStatusValue = String(item?.usedStatus ?? "")
		const key = `${ruleType.value}|${port}|${protocol}|${strategyValue}|${addressNormalized}|${description}|${usedStatusValue}`
		const family = String(item?.family ?? "").toLowerCase()
		const existing = map.get(key)
		if (!existing) {
			const cloned = { ...item, _families: new Set<string>() }
			if (family) cloned._families.add(family)
			if (addressNormalized === "" && rawAddress.trim().toLowerCase() === "anywhere") {
				cloned.address = "Anywhere"
			}
			map.set(key, cloned)
			continue
		}
		if (family) existing._families.add(family)
		if (!existing.address && rawAddress.trim().toLowerCase() === "anywhere") {
			existing.address = "Anywhere"
		}
	}
	return Array.from(map.values()).map(item => {
		const families: Set<string> | undefined = item._families
		if (families && families.size > 0) {
			const list = Array.from(families)
			const hasV4 = families.has("ipv4")
			const hasV6 = families.has("ipv6")
			item.family = hasV4 && hasV6 ? "ipv4/ipv6" : list[0]
		}
		delete item._families
		return item
	})
}

function addRule() {
	rulesDrawerRef.value?.open(ruleType.value)
}

function editRule(rule: any) {
	rulesDrawerRef.value?.open(ruleType.value, rule)
}

function handleSearch() {
	pagination.value.page = 1
	getRulesList()
}

function handleTypeChange(value: string) {
	ruleType.value = value as RuleType
	pagination.value.page = 1
	selectedRowKeys.value = []
	getRulesList()
}

function handlePageChange(page: number) {
	pagination.value.page = page
	getRulesList()
}

function handlePageSizeChange(limit: number) {
	pagination.value.limit = limit
	pagination.value.page = 1
	getRulesList()
}

function handleRefreshChange(value: number) {
	refreshSeconds.value = value
	setupRefreshTimer()
}

function setupRefreshTimer() {
	if (refreshTimer.value) {
		window.clearInterval(refreshTimer.value)
		refreshTimer.value = null
	}
	if (refreshSeconds.value > 0) {
		refreshTimer.value = window.setInterval(() => {
			getRulesList()
			getBase()
		}, refreshSeconds.value * 1000)
	}
}

async function changeOperation(operation: string) {
	try {
		const res: any = await hostsFirewallOperateAPI({ operation })
		if (isSucc(res.code)) {
		   message.success("操作成功")
		   await getBase()
		}
	} catch (error: any) {
	 
	}
}

async function performDelete(row: any) {
	if (ruleType.value === "ip") {
		return hostsFirewallIPAPI({
			operation: "remove",
			address: row.address,
			strategy: row.strategy,
			description: row.description || ""
		})
	}
	if (ruleType.value === "forward") {
		return hostsFirewallForwardAPI({
			forceDelete: false,
			rules: [
				{
					operation: "remove",
					num: row.num || "",
					protocol: row.protocol,
					port: row.port,
					targetIP: row.targetIP,
					targetPort: row.targetPort
				}
			]
		})
	}
	return hostsFirewallPortAPI({
		operation: "remove",
		port: row.port,
		protocol: row.protocol,
		strategy: row.strategy,
		address: row.address || "",
		description: row.description || ""
	})
}

function confirmDelete(row: any) {
	dialog.warning({
		title: "确认删除当前规则吗？",
		content: "删除后会立即作用到系统防火墙规则，请确认当前业务不再依赖该规则。",
		positiveText: "确认删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			const res: any = await performDelete(row)
			if (!isSucc(res.code)) {
				message.error(res.msg || "删除失败")
				return
			}
			message.success("删除成功")
			await getRulesList()
			await getBase()
		}
	})
}

function batchDelete() {
	if (selectedRows.value.length === 0) {
		message.warning("请先选择要删除的规则")
		return
	}
	dialog.warning({
		title: "确认批量删除吗？",
		content: `即将删除 ${selectedRows.value.length} 条规则，此操作会直接影响服务器防火墙。`,
		positiveText: "确认删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			for (const row of selectedRows.value) {
				const res: any = await performDelete(row)
				if (!isSucc(res.code)) {
					message.error(res.msg || "批量删除失败")
					return
				}
			}
			message.success("批量删除成功")
			await getRulesList()
			await getBase()
		}
	})
}

function handleCheckedRowKeys(value: Array<string | number>) {
	selectedRowKeys.value = value.map(item => String(item))
}

async function handleDrawerSave(payload: any, formLoading: Ref<boolean>) {
	formLoading.value = true
	try {
		let res: any
		if (payload.type === "ip") {
			if (payload.isEdit) {
				res = await hostsFirewallUpdateAddrAPI({
					oldRule: {
						operation: "remove",
						address: payload.oldData?.address || "",
						strategy: payload.oldData?.strategy || "accept",
						description: payload.oldData?.description || ""
					},
					newRule: {
						operation: "add",
						address: payload.data.address,
						strategy: payload.data.strategy,
						description: payload.data.description || ""
					}
				})
			} else {
				res = await hostsFirewallIPAPI({
					operation: "add",
					address: payload.data.address,
					strategy: payload.data.strategy,
					description: payload.data.description || ""
				})
			}
		} else if (payload.type === "forward") {
			res = await hostsFirewallForwardAPI({
				forceDelete: false,
				rules: [
					{
						operation: payload.isEdit ? "remove" : "add",
						num: payload.isEdit ? payload.oldData?.num || "" : "",
						protocol: payload.isEdit ? payload.oldData?.protocol || "tcp" : payload.data.protocol,
						port: payload.isEdit ? payload.oldData?.port || "" : payload.data.port,
						targetIP: payload.isEdit ? payload.oldData?.targetIP || "127.0.0.1" : payload.data.targetIP,
						targetPort: payload.isEdit ? payload.oldData?.targetPort || "" : payload.data.targetPort
					}
				]
			})
			if (isSucc(res.code) && payload.isEdit) {
				res = await hostsFirewallForwardAPI({
					forceDelete: false,
					rules: [
						{
							operation: "add",
							protocol: payload.data.protocol,
							port: payload.data.port,
							targetIP: payload.data.targetIP,
							targetPort: payload.data.targetPort
						}
					]
				})
			}
		} else if (payload.isEdit) {
			res = await hostsFirewallUpdatePortAPI({
				oldRule: {
					operation: "remove",
					port: payload.oldData?.port || "",
					protocol: payload.oldData?.protocol || "tcp",
					strategy: payload.oldData?.strategy || "accept",
					address: payload.oldData?.address || "",
					description: payload.oldData?.description || ""
				},
				newRule: {
					operation: "add",
					port: payload.data.port,
					protocol: payload.data.protocol,
					strategy: payload.data.strategy,
					address: payload.data.address || "",
					description: payload.data.description || ""
				}
			})
		} else {
			res = await hostsFirewallPortAPI({
				operation: "add",
				port: payload.data.port,
				protocol: payload.data.protocol,
				strategy: payload.data.strategy,
				address: payload.data.address || "",
				description: payload.data.description || ""
			})
		}

		if (!isSucc(res.code)) {
			message.error(res.msg || "保存失败")
			return
		}
		message.success(payload.isEdit ? "规则编辑成功" : "规则创建成功")
		rulesDrawerRef.value?.close()
		await getRulesList()
		await getBase()
	} finally {
		formLoading.value = false
	}
}

async function refreshAll() {
	await Promise.all([getBase(), getRulesList()])
}

onMounted(() => {
	void refreshAll()
	setupRefreshTimer()
})

onBeforeUnmount(() => {
	if (refreshTimer.value) {
		window.clearInterval(refreshTimer.value)
	}
})
</script>
