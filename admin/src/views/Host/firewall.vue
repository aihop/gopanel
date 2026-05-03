<template>
  <div>
    <FirewallOverview
      :summary-cards="summaryCards"
      :base="base"
      :is-running="isRunning"
      @operate="changeOperation"
      @refresh="refreshAll"
    />

    <FirewallRuleWorkspace
      :rule-type="ruleType"
      :keyword="keyword"
      :strategy="strategy"
      :status="status"
      :refresh-seconds="refreshSeconds"
      :strategy-options="strategyOptions"
      :status-options="statusOptions"
      :refresh-options="refreshOptions"
      :create-button-text="createButtonText"
      :rule-type-label="ruleTypeLabel"
      :selected-count="selectedRowKeys.length"
      :selected-disabled="selectedRows.length === 0"
      :columns="columns"
      :rules="rules"
      :loading="loading"
      :pagination="pagination"
      :selected-row-keys="selectedRowKeys"
      @update:rule-type="handleTypeChange"
      @update:keyword="handleKeywordChange"
      @update:strategy="handleStrategyChange"
      @update:status="handleStatusChange"
      @update:refresh-seconds="handleRefreshChange"
      @search="handleSearch"
      @create="addRule"
      @batch-delete="batchDelete"
      @page-change="handlePageChange"
      @page-size-change="handlePageSizeChange"
      @checked-row-keys="handleCheckedRowKeys"
    />

    <RulesDrawer
      ref="rulesDrawerRef"
      @save="handleDrawerSave"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import {
	hostsFirewallBaseAPI,
	hostsFirewallOperateAPI,
	hostsFirewallListAPI
} from "@/api/host/firewall"
import { isSucc } from "@/utils/is"
import RulesDrawer from "./components/RulesDrawer.vue"
import FirewallOverview from "./components/FirewallOverview.vue"
import FirewallRuleWorkspace from "./components/FirewallRuleWorkspace.vue"
import { createFirewallColumns } from "./components/firewallColumns"
import { deleteFirewallRule, mergeDualStackRules, saveFirewallRule } from "./components/firewallRuleService"

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

const columns = computed<any[]>(() => createFirewallColumns(ruleType.value, editRule, confirmDelete))

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
			rules.value = mergeDualStackRules(ruleType.value, items)
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

function confirmDelete(row: any) {
	dialog.warning({
		title: "确认删除当前规则吗？",
		content: "删除后会立即作用到系统防火墙规则，请确认当前业务不再依赖该规则。",
		positiveText: "确认删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			const res: any = await deleteFirewallRule(ruleType.value, row)
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
				const res: any = await deleteFirewallRule(ruleType.value, row)
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
		const res: any = await saveFirewallRule(payload)

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
