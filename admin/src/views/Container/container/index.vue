<template>
  <div v-loading="loading">
    <div v-if="dockerStatus !== 'Running'">
      <n-alert
        title="Tips"
        type="warning"
      >
        <span class="mr-3">{{ $t("container.serviceUnavailable") }}</span>
        <n-button
          type="primary"
          class="bt"
          ghost
          @click="goSetting"
        >{{ $t("container.setting") }}</n-button>
        <span class="ml-3">{{ $t("container.startIn") }}</span>
      </n-alert>
    </div>
    <LayoutContent
      :title="$t('container.container', 2)"
      :class="{ mask: dockerStatus !== 'Running' }"
    >
      <template #rightButton>
        <n-space class="flex flex-col justify-end sm:flex-row">
          <div class="w-[200px]">
            <n-select
              v-model:value="searchState"
              @update:value="search"
              clearable
              :options="stateOptions"
            >
              <template #header>{{ $t("commons.table.status") }}</template>
            </n-select>
          </div>
          <TableColumnSelect
            :columns="columnSettings"
            storage-key="containerColumn"
            size="medium"
            button-label="列设置"
            @update:columns="handleColumnSettingsChange"
          />
        </n-space>
      </template>
      <template #toolbar>
        <div class="flex w-full flex-col gap-4 md:flex-row md:justify-between py-3">
          <div class="flex flex-wrap gap-4">
            <n-button
              type="primary"
              @click="onOpenDialog('create')"
            >
              {{ $t("container.create") }}
            </n-button>
            <n-button
              type="primary"
              ghost
              @click="onClean()"
            >
              {{ $t("container.containerPrune") }}
            </n-button>

            <n-button-group>
              <n-button
                :disabled="checkStatus('start', null)"
                @click="onOperate('start', null)"
              >
                {{ $t("container.start") }}
              </n-button>
              <n-button
                :disabled="checkStatus('stop', null)"
                @click="onOperate('stop', null)"
              >
                {{ $t("container.stop") }}
              </n-button>
              <n-button
                :disabled="checkStatus('restart', null)"
                @click="onOperate('restart', null)"
              >
                {{ $t("container.restart") }}
              </n-button>
              <n-button
                :disabled="checkStatus('kill', null)"
                @click="onOperate('kill', null)"
              >
                {{ $t("container.kill") }}
              </n-button>
              <n-button
                :disabled="checkStatus('pause', null)"
                @click="onOperate('pause', null)"
              >
                {{ $t("container.pause") }}
              </n-button>
              <n-button
                :disabled="checkStatus('unpause', null)"
                @click="onOperate('unpause', null)"
              >
                {{ $t("container.unpause") }}
              </n-button>
              <n-button
                :disabled="checkStatus('remove', null)"
                @click="onOperate('remove', null)"
              >
                {{ $t("container.remove") }}
              </n-button>
            </n-button-group>
          </div>
          <div class="flex flex-row gap-2 md:flex-col lg:flex-row">
            <TableSetting
              title="container-refresh"
              @search="refresh()"
            />
            <TableSearch
              @search="search()"
              v-model:searchName="searchName"
            />
          </div>
        </div>
      </template>
      <template #main>
        <ComplexTable
          :data="data"
          :columns="visibleColumns"
          :pagination-config="paginationConfig"
          v-model:selects="selects"
          @search="search"
          @sort-change="search"
          :row-style="{ height: '65px' }"
          localKey="containerColumn"
          style="width: 100%"
          :row-selection="true"
          row-key="containerID"
          @update:paginationConfig="search"
        >
          <template #pagination>
            <n-pagination
              class="w-full mt-10 flex flex-row justify-end"
              v-model:page="paginationConfig.currentPage"
              v-model:page-size="paginationConfig.pageSize"
              :page-sizes="[5, 10, 20, 50, 100]"
              :show-size-picker="true"
              :page-count="Math.ceil(paginationConfig.total / paginationConfig.pageSize)"
              @update:page="search"
              @update:page-size="(size) => handlePageSizeChange(size)"
            />
          </template>
        </ComplexTable>
      </template>
    </LayoutContent>
    <OpDialog
      ref="opRef"
      @search="search"
    />

    <CodeDialog ref="myDetail" />
    <PruneDialog
      @search="search"
      ref="dialogPruneRef"
    />

    <RenameDialog
      @search="search"
      ref="dialogRenameRef"
    />
    <ContainerLogDialog
      ref="dialogContainerLogRef"
      :mask-closable="false"
    />
    <OperateDialog
      @search="search"
      ref="dialogOperateRef"
    />
    <UpgradeDialog
      @search="search"
      ref="dialogUpgradeRef"
    />
    <CommitDialog
      @search="search"
      ref="dialogCommitRef"
    />
    <MonitorDialog ref="dialogMonitorRef" />
    <TerminalDialog
      ref="dialogTerminalRef"
      :mask-closable="false"
    />

    <PortJumpDialog ref="dialogPortJumpRef" />
  </div>
</template>

<script lang="tsx" setup>
import PruneDialog from "@/views/Container/container/prune/index.vue"
import RenameDialog from "@/views/Container/container/rename/index.vue"
import OperateDialog from "@/views/Container/container/operate/index.vue"
import UpgradeDialog from "@/views/Container/container/upgrade/index.vue"
import CommitDialog from "@/views/Container/container/commit/index.vue"
import MonitorDialog from "@/views/Container/container/monitor/index.vue"
import ContainerLogDialog from "@/views/Container/container/log/index.vue"
import TerminalDialog from "@/views/Container/container/terminal/index.vue"
import CodeDialog from "@/components/CodeDialog.vue"
import PortJumpDialog from "@/components/PortJump.vue"
import OpDialog from "@/components/OpDialog.vue"
import LayoutContent from "@/components/LayoutContent.vue"
import ComplexTable from "@/components/ComplexTable.vue"
import TableSearch from "@/components/TableSearch.vue"
import TableSetting from "@/components/TableSetting.vue"

import { reactive, onMounted, ref, computed, h, resolveComponent } from "vue"
import {
	containerListStats,
	containerOperator,
	inspect,
	loadContainerInfo,
	loadInstanceStatus,
	searchContainer
} from "@/api/modules/container"
import { type Container } from "@/api/interface/container"
import { t } from "@/i18n"
import { MsgWarning } from "@/utils/message"
import { useRouter } from "vue-router"
import { NButton, NSpace, NCard, NDropdown, NTag } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import TableColumnSelect from "@/components/TableColumnSelect.vue"

const router = useRouter()

const stateOptions = computed(() => [
	{ label: t("commons.table.all"), value: "all" },
	{ label: t("commons.status.created"), value: "created" },
	{ label: t("commons.status.running"), value: "running" },
	{ label: t("commons.status.paused"), value: "paused" },
	{ label: t("commons.status.restarting"), value: "restarting" },
	{ label: t("commons.status.removing"), value: "removing" },
	{ label: t("commons.status.exited"), value: "exited" },
	{ label: t("commons.status.dead"), value: "dead" }
])

const loading = ref(false)
const data = ref()
const selects = ref<any>([])
type ColumnSetting = {
	key: string
	title: string
	visible: boolean
	fixed?: boolean
	original?: any
}

const paginationConfig = reactive({
	cacheSizeKey: "container-page-size",
	currentPage: 1,
	pageSize: 10,
	total: 0,
	orderBy: "created_at",
	order: "null"
})
const searchName = ref()
const searchState = ref("all")
const dialogUpgradeRef = ref()
const dialogCommitRef = ref()
const dialogPortJumpRef = ref()
const opRef = ref()
const includeAppStore = ref()
const columns = ref([
	{
		key: "__selection",
		type: "selection",
		width: 50,
		options: ["all", "none"]
	},
	{
		title: t("commons.table.name"),
		key: "name",
		width: 200,
		// 1. 关闭默认的省略号/单行限制
		ellipsis: false,
		render(row: Container.ContainerInfo) {
			return h(
				NButton,
				{
					text: true,
					type: "primary",
					// 2. 这里的样式非常关键
					style: {
						padding: 0,
						textAlign: "left",
						height: "auto", // 允许按钮高度随内容撑开
						whiteSpace: "normal", // 覆盖按钮默认的 nowrap
						wordBreak: "break-all" // 确保长字符串（如无空格的名字）能强制换行
					},
					onClick: () => onInspect(row.containerID)
				},
				{ default: () => row.name }
			)
		}
	},
	{
		title: t("container.image"),
		key: "imageName",
		width: 100
	},
	{
		title: t("commons.table.status"),
		key: "state",
		width: 100,
		render(row: Container.ContainerInfo) {
			const opt = stateOptions.value.find((o: any) => o.value === row.state)
			const label = opt ? opt.label : (row.state ?? "--")
			return h(
				NTag,
				{
					size: "small",
					type: row.state === "running" ? "success" : row.state === "dead" ? "error" : "default",
					bordered: false
				},
				{ default: () => label }
			)
		}
	},
	{
		title: t("container.ip"),
		key: "network",
		width: 100
	},
	{
		title: t("container.source"),
		key: "source",
		width: 110,
		render(row: Container.ContainerInfo) {
			if (!row.hasLoad) {
				return h("div", {}, h("n-button", { loading: true, text: true, size: "small" }, { default: () => "" }))
			}
			return h("div", [
				h("div", { class: "text-xs" }, `CPU: ${row.cpuPercent?.toFixed(2) ?? "--"}%`),
				h(
					"div",
					{ class: "inline-block text-xs" },
					`${t("monitor.memory")}: ${row.memoryPercent?.toFixed(2) ?? "--"}%`
				),
				h(
					resolveComponent("n-popover"),
					{ placement: "right", width: 500, trigger: "click" },
					{
						trigger: () =>
							h(Icon, {
								name: "fluent:more-vertical-16-filled",
								size: 18,
								class: "ml-2 cursor-pointer text-blue-500 hover:text-blue-600"
							}),
						default: () =>
							h("div", [
								h("div", { class: "flex gap-4 justify-around w-full" }, [
									h(
										resolveComponent("n-statistic"),
										{
											class: "w-1/3",
											label: t("container.cpuUsage"),
											value: loadCPUValue(row.cpuTotalUsage ?? 0)
										},
										{
											suffix: () => loadCPUUnit(row.cpuTotalUsage ?? 0)
										}
									),
									h(
										resolveComponent("n-statistic"),
										{
											class: "w-1/3",
											label: t("container.cpuTotal"),
											value: loadCPUValue(row.systemUsage ?? 0)
										},
										{
											suffix: () => loadCPUUnit(row.systemUsage ?? 0)
										}
									),
									h(resolveComponent("n-statistic"), {
										class: "w-1/3",
										label: t("container.core"),
										value: Array.isArray(row.percpuUsage)
											? row.percpuUsage.length
											: (row.percpuUsage ?? "--")
									})
								]),
								h("div", { class: "flex gap-4 justify-around w-full mt-4" }, [
									h(
										resolveComponent("n-statistic"),
										{
											class: "w-1/3",
											label: t("container.memUsage"),
											value: loadMemValue(row.memoryUsage ?? 0),
											precision: 2
										},
										{
											suffix: () => loadMemUnit(row.memoryUsage ?? 0)
										}
									),
									h(
										resolveComponent("n-statistic"),
										{
											class: "w-1/3",
											label: t("container.memCache"),
											value: loadMemValue(row.memoryCache ?? 0),
											precision: 2
										},
										{
											suffix: () => loadMemUnit(row.memoryCache ?? 0)
										}
									),
									h(
										resolveComponent("n-statistic"),
										{
											class: "w-1/3",
											label: t("container.memTotal"),
											value: loadMemValue(row.memoryLimit ?? 0),
											precision: 2
										},
										{
											suffix: () => loadMemUnit(row.memoryLimit ?? 0)
										}
									)
								])
							])
					}
				)
			])
		}
	},
	{
		title: t("commons.table.port"),
		key: "ports",
		width: 130,
		render(row: Container.ContainerInfo) {
			return h(
				NSpace,
				{ vertical: true, size: "small" },
				{
					default: () =>
						(row.ports || []).map((port: string) =>
							h(
								NTag,
								{
									bordered: false,
									size: "small",
									type: "info"
								},
								{ default: () => port }
							)
						)
				}
			)
		}
	},
	{
		title: t("container.upTime"),
		key: "runTime",
		width: 150
	},
	{
		title: t("commons.table.operate"),
		key: "operate",
		width: 120,
		fixed: "right",
		render(row: Container.ContainerInfo) {
			return h(NSpace, null, {
				default: () => [
					h(
						NButton,
						{
							text: true,
							type: "primary",
							disabled: row.state !== "running",
							onClick: () => onTerminal(row)
						},
						{ default: () => t("container.containerTerminal") }
					),
					h(
						NButton,
						{
							text: true,
							type: "primary",
							onClick: () =>
								dialogContainerLogRef.value!.acceptParams({
									containerID: row.containerID,
									container: row.name
								})
						},
						{ default: () => t("commons.button.log") }
					),
					h(
						NDropdown,
						{
							trigger: "hover",
							options: buttons.map(btn => ({
								label: btn.label,
								key: btn.label,
								disabled: btn.disabled ? btn.disabled(row) : false
							})),
							onSelect: (key: string) => {
								const btn = buttons.find(b => b.label === key)
								if (btn) btn.click(row)
							}
						},
						{
							default: () =>
								h(
									NButton,
									{
										text: true,
										type: "primary"
									},
									{ default: () => t("tabs.more") }
								)
						}
					)
				]
			})
		}
	}
])

const columnSettings = ref<ColumnSetting[]>(
	columns.value.map((column: any) => ({
		key: String(column.key || column.type),
		title:
			typeof column.title === "string"
				? column.title
				: column.type === "selection"
					? "选择"
					: String(column.key || ""),
		visible: true,
		fixed: column.type === "selection" || column.fixed === "right" || column.fixed === "left",
		original: column
	}))
)

const visibleColumns = computed(() => {
	const visibleKeys = new Set(columnSettings.value.filter(item => item.visible || item.fixed).map(item => item.key))
	return columns.value.filter((column: any) => visibleKeys.has(String(column.key || column.type)))
})

function handleColumnSettingsChange(nextColumns: ColumnSetting[]) {
	columnSettings.value = nextColumns.map(item => ({
		...item,
		original: item.original || columns.value.find((column: any) => String(column.key || column.type) === item.key)
	}))
}

const dockerStatus = ref("Running")
const loadStatus = async () => {
	loading.value = true
	await loadInstanceStatus()
		.then(res => {
			loading.value = false
			dockerStatus.value = res.data
			if (dockerStatus.value === "Running") {
				search()
			}
		})
		.catch(() => {
			dockerStatus.value = "Failed"
			loading.value = false
		})
}

const goDashboard = async (port: any) => {
	if (port.indexOf("127.0.0.1") !== -1) {
		MsgWarning(t("container.unExposedPort"))
		return
	}
	if (port.indexOf(":") === -1) {
		MsgWarning(t("commons.msg.errPort"))
		return
	}
	let portEx = port.match(/:(\d+)/)[1]

	let matches = port.match(new RegExp(":", "g"))
	let ip = matches && matches.length > 1 ? "ipv6" : "ipv4"
	dialogPortJumpRef.value.acceptParams({ port: portEx, ip: ip })
}

const goSetting = async () => {
	router.push("/container/setting")
}

interface Filters {
	filters?: string
}
const props = withDefaults(defineProps<Filters>(), {
	filters: ""
})

const myDetail = ref()

const dialogContainerLogRef = ref()
const dialogRenameRef = ref()
const dialogPruneRef = ref()

const search = async (column?: any) => {
	localStorage.setItem("includeAppStore", includeAppStore.value ? "true" : "false")
	let filterItem = props.filters ? props.filters : ""
	paginationConfig.orderBy = column?.order ? column.prop : paginationConfig.orderBy
	paginationConfig.order = column?.order ? column.order : paginationConfig.order
	let params = {
		name: searchName.value,
		state: searchState.value || "all",
		page: paginationConfig.currentPage,
		pageSize: paginationConfig.pageSize,
		filters: filterItem,
		orderBy: 'created_at',
		order: paginationConfig.order,
		excludeAppStore: !includeAppStore.value
	}
	loading.value = true
	loadStats()
	await searchContainer(params)
		.then(res => {
			loading.value = false
			data.value = Array.isArray(res.data.items) ? res.data.items : []
			paginationConfig.total = res.data.total
			selects.value = []
		})
		.catch(() => {
			loading.value = false
		})
}

const refresh = async () => {
	let filterItem = props.filters ? props.filters : ""
	let params = {
		name: searchName.value,
		state: searchState.value || "all",
		page: paginationConfig.currentPage,
		pageSize: paginationConfig.pageSize,
		filters: filterItem,
		orderBy: paginationConfig.orderBy,
		order: paginationConfig.order
	}
	loadStats()
	const res = await searchContainer(params)
	let containers = res.data.items || []
	for (const container of containers) {
		for (const c of data.value) {
			c.hasLoad = true
			if (container.containerID == c.containerID) {
				const containerData = container as Record<string, any>
				for (let key in containerData) {
					if (key !== "cpuPercent" && key !== "memoryPercent") {
						;(c as Record<string, any>)[key] = containerData[key]
					}
				}
			}
		}
	}
}

const handlePageSizeChange = (size: number) => {
	paginationConfig.pageSize = size
	paginationConfig.currentPage = 1
	if (paginationConfig.cacheSizeKey) {
		localStorage.setItem(paginationConfig.cacheSizeKey, String(size))
	}
	search()
}

const loadStats = async () => {
	const res = await containerListStats()
	let stats = res.data || []
	if (stats.length === 0) {
		return
	}
	for (const container of data.value) {
		for (const item of stats) {
			if (container.containerID === item.containerID) {
				container.hasLoad = true
				container.cpuTotalUsage = item.cpuTotalUsage
				container.systemUsage = item.systemUsage
				container.cpuPercent = item.cpuPercent
				container.percpuUsage = item.percpuUsage
				container.memoryCache = item.memoryCache
				container.memoryUsage = item.memoryUsage
				container.memoryLimit = item.memoryLimit
				container.memoryPercent = item.memoryPercent
				break
			}
		}
	}
}

const loadCPUUnit = (t: number) => {
	const num = 1000
	if (t < num) return " ns"
	if (t < Math.pow(num, 2)) return " μs"
	if (t < Math.pow(num, 3)) return " ms"
	return " s"
}
function loadCPUValue(t: number) {
	const num = 1000
	if (t < num) return t
	if (t < Math.pow(num, 2)) return Number((t / num).toFixed(2))
	if (t < Math.pow(num, 3)) return Number((t / Math.pow(num, 2)).toFixed(2))
	return Number((t / Math.pow(num, 3)).toFixed(2))
}
const loadMemUnit = (t: number) => {
	if (t == 0) {
		return ""
	}
	const num = 1024
	if (t < num) return " B"
	if (t < Math.pow(num, 2)) return " KiB"
	if (t < Math.pow(num, 3)) return " MiB"
	return " GiB"
}
function loadMemValue(t: number) {
	const num = 1024
	if (t < num) return t
	if (t < Math.pow(num, 2)) return Number((t / num).toFixed(2))
	if (t < Math.pow(num, 3)) return Number((t / Math.pow(num, 2)).toFixed(2))
	return Number((t / Math.pow(num, 3)).toFixed(2))
}

const dialogOperateRef = ref()
const onEdit = async (container: string) => {
	const res = await loadContainerInfo(container)
	if (res.data) {
		onOpenDialog("edit", res.data)
	}
}
const onOpenDialog = async (
	title: string,
	rowData: Partial<Container.ContainerHelper> = {
		cmd: [],
		cmdStr: "",
		publishAllPorts: false,
		exposedPorts: [],
		cpuShares: 1024,
		nanoCPUs: 0,
		memory: 0,
		memoryItem: 0,
		volumes: [],
		labels: [],
		env: [],
		restartPolicy: "no"
	}
) => {
	let params = {
		title,
		rowData: { ...rowData }
	}
	dialogOperateRef.value!.acceptParams(params)
}

const dialogMonitorRef = ref()
const onMonitor = (row: any) => {
	dialogMonitorRef.value!.acceptParams({ containerID: row.containerID, container: row.name })
}

const dialogTerminalRef = ref()
const onTerminal = (row: any) => {
	dialogTerminalRef.value!.acceptParams({ containerID: row.containerID, container: row.name })
}

const onInspect = async (id: string) => {
	console.log("inspect click", id)
	const res = await inspect({ id: id, type: "container" })
	console.log("inspect result", res)
	let detailInfo = JSON.stringify(JSON.parse(res.data), null, 2)
	let param = {
		header: t("commons.button.view"),
		detailInfo: detailInfo
	}
	console.log("myDetail", myDetail.value)
	myDetail.value?.acceptParams(param)
}

const onClean = () => {
	dialogPruneRef.value!.acceptParams()
}

const checkStatus = (operation: string, row: Container.ContainerInfo | null) => {
	let opList: Container.ContainerInfo[] = []
	if (row) {
		opList = [row]
	} else {
		if (selects.value && selects.value.length > 0) {
			const selectedIds = new Set(
				selects.value.map((item: any) => (typeof item === "object" ? item.containerID : item))
			)
			opList = data.value.filter((item: Container.ContainerInfo) => selectedIds.has(item.containerID))
		}
	}

	if (opList.length < 1) {
		return true
	}
	switch (operation) {
		case "start":
			for (const item of opList) {
				if (!item) continue
				if (item.state === "running") {
					return true
				}
			}
			return false
		case "stop":
			for (const item of opList) {
				if (item.state === "stopped" || item.state === "exited") {
					return true
				}
			}
			return false
		case "pause":
			for (const item of opList) {
				if (item.state === "paused" || item.state === "exited") {
					return true
				}
			}
			return false
		case "unpause":
			for (const item of opList) {
				if (item.state !== "paused") {
					return true
				}
			}
			return false
	}
}

const onOperate = async (op: string, row: Container.ContainerInfo | null) => {
	let opList: Container.ContainerInfo[] = []
	if (row) {
		opList = [row]
	} else {
		if (selects.value && selects.value.length > 0) {
			const selectedIds = new Set(
				selects.value.map((item: any) => (typeof item === "object" ? item.containerID : item))
			)
			opList = data.value.filter((item: Container.ContainerInfo) => selectedIds.has(item.containerID))
		}
	}
	let msg = t("container.operatorHelper", [t("container." + op)])
	let names = []
	for (const item of opList) {
		names.push(item.name)
		if (item.isFromApp) {
			msg = t("container.operatorAppHelper", [t("container." + op)])
		}
	}
	const successMsg = `${t("container." + op)}${t("commons.status.success")}`
	opRef.value.acceptParams({
		title: t("container." + op),
		names: names,
		msg: msg,
		api: containerOperator,
		params: { names: names, operation: op },
		successMsg
	})
}

const buttons = [
	// {
	// 	label: t("container.containerTerminal"),
	// 	disabled: (row: Container.ContainerInfo) => {
	// 		return row.state !== "running"
	// 	},
	// 	click: (row: Container.ContainerInfo) => {
	// 		onTerminal(row)
	// 	}
	// },
	// {
	// 	label: t("commons.button.log"),
	// 	click: (row: Container.ContainerInfo) => {
	// 		dialogContainerLogRef.value!.acceptParams({ containerID: row.containerID, container: row.name })
	// 	}
	// },
	{
		label: t("commons.button.edit"),
		click: (row: Container.ContainerInfo) => {
			onEdit(row.containerID)
		}
	},
	{
		label: t("commons.button.upgrade"),
		click: (row: Container.ContainerInfo) => {
			dialogUpgradeRef.value!.acceptParams({ container: row.name, image: row.imageName, fromApp: row.isFromApp })
		}
	},
	{
		label: t("container.monitor"),
		disabled: (row: Container.ContainerInfo) => {
			return row.state !== "running"
		},
		click: (row: Container.ContainerInfo) => {
			onMonitor(row)
		}
	},
	{
		label: t("container.rename"),
		click: (row: Container.ContainerInfo) => {
			dialogRenameRef.value!.acceptParams({ container: row.name })
		},
		disabled: (row: any) => {
			return row.isFromCompose
		}
	},
	{
		label: t("container.makeImage"),
		click: (row: Container.ContainerInfo) => {
			dialogCommitRef.value!.acceptParams({ containerID: row.containerID, containerName: row.name })
		},
		disabled: (row: any) => {
			return checkStatus("commit", row)
		}
	},
	{
		label: t("container.start"),
		click: (row: Container.ContainerInfo) => {
			onOperate("start", row)
		},
		disabled: (row: any) => {
			return checkStatus("start", row)
		}
	},
	{
		label: t("container.stop"),
		click: (row: Container.ContainerInfo) => {
			onOperate("stop", row)
		},
		disabled: (row: any) => {
			return checkStatus("stop", row)
		}
	},
	{
		label: t("container.restart"),
		click: (row: Container.ContainerInfo) => {
			onOperate("restart", row)
		},
		disabled: (row: any) => {
			return checkStatus("restart", row)
		}
	},
	{
		label: t("container.kill"),
		click: (row: Container.ContainerInfo) => {
			onOperate("kill", row)
		},
		disabled: (row: any) => {
			return checkStatus("kill", row)
		}
	},
	{
		label: t("container.pause"),
		click: (row: Container.ContainerInfo) => {
			onOperate("pause", row)
		},
		disabled: (row: any) => {
			return checkStatus("pause", row)
		}
	},
	{
		label: t("container.unpause"),
		click: (row: Container.ContainerInfo) => {
			onOperate("unpause", row)
		},
		disabled: (row: any) => {
			return checkStatus("unpause", row)
		}
	},
	{
		label: t("container.remove"),
		click: (row: Container.ContainerInfo) => {
			onOperate("remove", row)
		},
		disabled: (row: any) => {
			return checkStatus("remove", row)
		}
	}
]

onMounted(() => {
	let includeItem = localStorage.getItem("includeAppStore")
	includeAppStore.value = !includeItem || includeItem === "true"
	loadStatus()
})
</script>

<style scoped lang="scss">
.tagMargin {
	margin-top: 2px;
}
.source-font {
	font-size: 12px;
}
.svg-icon {
	margin-top: -3px;
	font-size: 6px;
	cursor: pointer;
}
.cell-button-class {
	button,
	:deep(span) {
		max-width: 100%;
		overflow: hidden;
	}
}
</style>
