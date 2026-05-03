<template>
  <div>
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
        <ContainerListToolbar
          :search-state="searchState"
          :search-name="searchName"
          :state-options="stateOptions"
          :column-settings="columnSettings"
          :bulk-actions="bulkActions"
          @update:search-state="handleSearchStateChange"
          @update:search-name="searchName = $event"
          @update:columns="handleColumnSettingsChange"
          @search="search"
          @refresh="refresh"
          @create="onOpenDialog('create')"
          @prune="onClean"
          @bulk-operate="onOperate($event, null)"
        />
      </template>
      <template #main>
        <ComplexTable
          :loading="loading"
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
              v-model:page-size="paginationConfig.limit"
              :page-sizes="[5, 10, 20, 50, 100]"
              :show-size-picker="true"
              :page-count="Math.ceil(paginationConfig.total / paginationConfig.limit)"
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
import OpDialog from "@/components/OpDialog.vue"
import ContainerListToolbar from "@/views/Container/container/ContainerListToolbar.vue"
import LayoutContent from "@/components/LayoutContent.vue"
import ComplexTable from "@/components/ComplexTable.vue"

import { onMounted, ref, computed } from "vue"
import {
	containerOperator,
	inspect,
	loadContainerInfo
} from "@/api/modules/container"
import { type Container } from "@/api/interface/container"
import { t } from "@/i18n"
import { useRouter } from "vue-router"
import { buildContainerRuntimeSummary, createColumnSettings, createContainerColumns, type ColumnSetting } from "@/views/Container/container/containerTableColumns"
import { useContainerListData } from "@/views/Container/container/useContainerListData"

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

const dialogUpgradeRef = ref()
const dialogCommitRef = ref()
const opRef = ref()
const getRowActions = (row: Container.ContainerInfo) => [
	{
		label: t("commons.button.edit"),
		click: () => onEdit(row)
	},
	{
		label: t("commons.button.upgrade"),
		click: () => {
			dialogUpgradeRef.value!.acceptParams({ container: row.name, image: row.imageName, fromApp: row.isFromApp })
		}
	},
	{
		label: t("container.monitor"),
		disabled: row.state !== "running",
		click: () => onMonitor(row)
	},
	{
		label: t("container.rename"),
		disabled: row.isFromCompose,
		click: () => {
			dialogRenameRef.value!.acceptParams({ container: row.name })
		}
	},
	{
		label: t("container.makeImage"),
		disabled: checkStatus("commit", row),
		click: () => {
			dialogCommitRef.value!.acceptParams({ containerID: row.containerID, containerName: row.name })
		}
	},
	{
		label: t("container.start"),
		disabled: checkStatus("start", row),
		click: () => onOperate("start", row)
	},
	{
		label: t("container.stop"),
		disabled: checkStatus("stop", row),
		click: () => onOperate("stop", row)
	},
	{
		label: t("container.restart"),
		disabled: checkStatus("restart", row),
		click: () => onOperate("restart", row)
	},
	{
		label: t("container.kill"),
		disabled: checkStatus("kill", row),
		click: () => onOperate("kill", row)
	},
	{
		label: t("container.pause"),
		disabled: checkStatus("pause", row),
		click: () => onOperate("pause", row)
	},
	{
		label: t("container.unpause"),
		disabled: checkStatus("unpause", row),
		click: () => onOperate("unpause", row)
	},
	{
		label: t("container.remove"),
		disabled: checkStatus("remove", row),
		click: () => onOperate("remove", row)
	}
]

const openLog = (row: Container.ContainerInfo) => {
	dialogContainerLogRef.value!.acceptParams({
		containerID: row.containerID,
		container: row.name,
		runtimeHost: row.runtimeHost || "",
		runtimeSummary: buildContainerRuntimeSummary(row)
	})
}

const columns = ref<any[]>([])

const columnSettings = ref<ColumnSetting[]>([])

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

function handleSearchStateChange(value: string) {
	searchState.value = value
	search()
}

const bulkActions = computed(() => [
	{ key: "start", label: t("container.start"), disabled: checkStatus("start", null) },
	{ key: "stop", label: t("container.stop"), disabled: checkStatus("stop", null) },
	{ key: "restart", label: t("container.restart"), disabled: checkStatus("restart", null) },
	{ key: "kill", label: t("container.kill"), disabled: checkStatus("kill", null) },
	{ key: "pause", label: t("container.pause"), disabled: checkStatus("pause", null) },
	{ key: "unpause", label: t("container.unpause"), disabled: checkStatus("unpause", null) },
	{ key: "remove", label: t("container.remove"), disabled: checkStatus("remove", null) }
])

const goSetting = async () => router.push("/container/setting")

interface Filters {
	filters?: string
}
const props = withDefaults(defineProps<Filters>(), {
	filters: ""
})

const {
	loading,
	data,
	selects,
	searchName,
	searchState,
	dockerStatus,
	paginationConfig,
	search,
	refresh,
	handlePageSizeChange,
	loadStatus,
	checkStatus,
	initIncludeAppStore
} = useContainerListData({ filters: props.filters })

const myDetail = ref()

const dialogContainerLogRef = ref()
const dialogRenameRef = ref()
const dialogPruneRef = ref()

const dialogOperateRef = ref()
const onEdit = async (row: Container.ContainerInfo) => {
	const res = await loadContainerInfo(row.containerID, row.runtimeHost || "")
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
	dialogMonitorRef.value!.acceptParams({
		containerID: row.containerID,
		container: row.name,
		runtimeSummary: buildContainerRuntimeSummary(row)
	})
}

const dialogTerminalRef = ref()
const onTerminal = (row: any) => {
	dialogTerminalRef.value!.acceptParams({
		containerID: row.containerID,
		container: row.name,
		runtimeHost: row.runtimeHost || "",
		runtimeSummary: buildContainerRuntimeSummary(row)
	})
}

const onInspect = async (row: Container.ContainerInfo) => {
	const res = await inspect({ id: row.containerID, type: "container", runtimeHost: row.runtimeHost || "" })
	let detailInfo = JSON.stringify(JSON.parse(res.data), null, 2)
	let param = {
		header: t("commons.button.view"),
		detailInfo: detailInfo,
		summary: buildContainerRuntimeSummary(row)
	}
	myDetail.value?.acceptParams(param)
}

const initColumns = () => {
	columns.value = createContainerColumns({
		stateOptions: stateOptions.value,
		onInspect,
		onTerminal,
		onLog: openLog,
		getRowActions
	})
	columnSettings.value = createColumnSettings(columns.value)
}

const onClean = () => dialogPruneRef.value!.acceptParams()

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

onMounted(() => { initIncludeAppStore(); initColumns(); loadStatus() })
</script>
