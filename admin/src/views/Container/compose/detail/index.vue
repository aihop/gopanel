<template>
  <div v-loading="loading">
    <div
      class="app-content"
      style="margin-top: 20px"
    >
      <n-card class="app-card">
        <n-row :gutter="20">
          <div>
            <n-tag
              effect="dark"
              type="success"
            >
              {{ composeName }}
            </n-tag>
          </div>
          <div
            v-if="createdBy === 'GoPanel'"
            style="margin-left: 50px"
          >
            <n-button
              link
              type="primary"
              @click="onComposeOperate('up')"
            >
              {{ $t("container.start") }}
            </n-button>
            <n-divider direction="vertical" />
            <n-button
              link
              type="primary"
              @click="onComposeOperate('stop')"
            >
              {{ $t("container.stop") }}
            </n-button>
            <n-divider direction="vertical" />
            <n-button
              link
              type="primary"
              @click="onComposeOperate('down')"
            >
              {{ $t("container.remove") }}
            </n-button>
          </div>
          <div v-else>
            <n-alert
              style="margin-top: -5px; margin-left: 50px"
              :closable="false"
              show-icon
              :title="$t('container.composeDetailHelper')"
              type="info"
            />
          </div>
        </n-row>
      </n-card>
    </div>
    <LayoutContent
      style="margin-top: 30px"
      back-name="Compose"
      :title="$t('container.containerList')"
      :reload="true"
    >
      <template #main>
        <n-button-group>
          <n-button
            :disabled="checkStatus('start')"
            @click="onOperate('start')"
          >
            {{ $t("container.start") }}
          </n-button>
          <n-button
            :disabled="checkStatus('stop')"
            @click="onOperate('stop')"
          >
            {{ $t("container.stop") }}
          </n-button>
          <n-button
            :disabled="checkStatus('restart')"
            @click="onOperate('restart')"
          >
            {{ $t("container.restart") }}
          </n-button>
          <n-button
            :disabled="checkStatus('kill')"
            @click="onOperate('kill')"
          >
            {{ $t("container.kill") }}
          </n-button>
          <n-button
            :disabled="checkStatus('pause')"
            @click="onOperate('pause')"
          >
            {{ $t("container.pause") }}
          </n-button>
          <n-button
            :disabled="checkStatus('unpause')"
            @click="onOperate('unpause')"
          >
            {{ $t("container.unpause") }}
          </n-button>
          <n-button
            :disabled="checkStatus('remove')"
            @click="onOperate('remove')"
          >
            {{ $t("container.remove") }}
          </n-button>
        </n-button-group>
        <ComplexTable
          v-model:selects="selects"
          :pagination-config="paginationConfig"
          style="margin-top: 20px"
          :data="data"
          @search="search"
        >
          <n-table-column
            type="selection"
            fix
          />
          <n-table-column
            :label="$t('commons.table.name')"
            min-width="100"
            prop="name"
            fix
            show-overflow-tooltip
          >
            <template #default="{ row }">
              <n-button
                text
                type="primary"
                @click="onInspect(row.containerID)"
              >
                {{ row.name }}
              </n-button>
            </template>
          </n-table-column>
          <n-table-column
            :label="$t('container.image')"
            show-overflow-tooltip
            min-width="100"
            prop="imageName"
          />
          <n-table-column
            :label="$t('container.runtimeType')"
            min-width="160"
          >
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <div class="flex flex-wrap items-center gap-2">
                  <n-tag
                    size="small"
                    :type="row.runtimeKind === 'docker' ? 'success' : 'warning'"
                  >
                    {{ getRuntimeKindLabel(row) }}
                  </n-tag>
                  <n-tag
                    size="small"
                    :type="row.runtimeMode === 'rootless' ? 'warning' : 'default'"
                  >
                    {{ getRuntimeModeLabel(row) }}
                  </n-tag>
                </div>
                <div class="text-xs text-slate-500">
                  {{ $t("container.runUser") }}: {{ getRunUserLabel(row) }}
                </div>
              </div>
            </template>
          </n-table-column>
          <n-table-column
            :label="$t('commons.table.status')"
            min-width="50"
            prop="state"
            fix
          >
            <template #default="{ row }">
              <Status
                :key="row.state"
                :status="row.state"
              ></Status>
            </template>
          </n-table-column>
          <n-table-column
            :label="$t('container.upTime')"
            min-width="100"
            prop="runTime"
            fix
          />
          <n-table-column
            prop="createTime"
            :label="$t('commons.table.date')"
            :formatter="dateFormat"
            show-overflow-tooltip
          />
          <n-table-operations
            width="220"
            :ellipsis="10"
            :buttons="buttons"
            :label="$t('commons.table.operate')"
            fix
          />
        </ComplexTable>

        <CodeDialog ref="mydetail" />
        <OpDialog
          ref="opRef"
          @search="search"
        />

        <ContainerLogDialog ref="dialogContainerLogRef" />
        <MonitorDialog ref="dialogMonitorRef" />
        <TerminalDialog ref="dialogTerminalRef" />
      </template>
    </LayoutContent>
  </div>
</template>

<script lang="ts" setup>
import type { Container } from "@/api/interface/container"
import { composeOperator, containerOperator, inspect, containerListAPI } from "@/api/modules/container"
import CodeDialog from "@/components/CodeDialog.vue"
import LayoutContent from "@/components/LayoutContent.vue"
import OpDialog from "@/components/OpDialog.vue"
import Status from "@/components/Status.vue"
import { t } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { dateFormat } from "@/utils/util"
import { buildRuntimeSummaryText, getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"
import ContainerLogDialog from "@/views/Container/container/log/index.vue"
import MonitorDialog from "@/views/Container/container/monitor/index.vue"
import TerminalDialog from "@/components/Terminal.vue"
import { reactive, ref } from "vue"

import { useDialog } from "naive-ui"

// 在 setup 中
const dialog = useDialog()

const composeName = ref()
const composePath = ref()
const filters = ref()
const createdBy = ref()

const dialogContainerLogRef = ref()

const opRef = ref()

interface DialogProps {
	createdBy: string
	name: string
	path: string
	filters: string
}
function acceptParams(props: DialogProps): void {
	composePath.value = props.path
	composeName.value = props.name
	filters.value = props.filters
	createdBy.value = props.createdBy
	search()
}

const data = ref()
const selects = ref<any>([])
const paginationConfig = reactive({
	cacheSizeKey: "container-page-size",
	currentPage: 1,
	limit: 10,
	total: 0
})

const loading = ref(false)

async function search() {
	let filterItem = filters.value
	let params = {
		name: "",
		state: "all",
		page: paginationConfig.currentPage,
		limit: paginationConfig.limit,
		filters: filterItem,
		orderBy: "created_at",
		order: "null"
	}
	loading.value = true
	await containerListAPI(params)
		.then(res => {
			loading.value = false
			data.value = res.data.items || []
			paginationConfig.total = res.data.total
		})
		.catch(() => {
			loading.value = false
		})
}

const detailInfo = ref()
const mydetail = ref()
async function onInspect(id: string) {
	const row = (data.value || []).find((item: Container.ContainerInfo) => item.containerID === id)
	const res = await inspect({ id, type: "container", runtimeHost: row?.runtimeHost || "" })
	detailInfo.value = JSON.stringify(JSON.parse(res.data), null, 2)
	let param = {
		header: t("commons.button.view"),
		detailInfo: detailInfo.value,
		summary: buildRuntimeSummaryText(row, {
			kindFallback: t("container.runtimeType"),
			rootlessLabel: t("container.rootless"),
			rootfulLabel: t("container.rootful"),
			defaultModeLabel: t("container.defaultMode"),
			userFallback: t("container.userDefault"),
			runUserPrefix: `${t("container.runUser")}: `
		})
	}
	mydetail.value!.acceptParams(param)
}

function checkStatus(operation: string) {
	if (selects.value.length < 1) {
		return true
	}
	switch (operation) {
		case "start":
			for (const item of selects.value) {
				if (item.state === "running") {
					return true
				}
			}
			return false
		case "stop":
			for (const item of selects.value) {
				if (item.state === "stopped" || item.state === "exited") {
					return true
				}
			}
			return false
		case "pause":
			for (const item of selects.value) {
				if (item.state === "paused" || item.state === "exited") {
					return true
				}
			}
			return false
		case "unpause":
			for (const item of selects.value) {
				if (item.state !== "paused") {
					return true
				}
			}
			return false
	}
}

async function onOperate(op: string) {
	let msg = t("container.operatorHelper", [t(`container.${op}`)])
	let names = []
	for (const item of selects.value) {
		names.push(item.name)
		if (item.isFromApp) {
			msg = t("container.operatorAppHelper", [t(`container.${op}`)])
		}
	}
	opRef.value.acceptParams({
		title: t(`container.${op}`),
		names,
		msg,
		api: containerOperator,
		params: { names, operation: op },
		successMsg: `${t(`container.${op}`)}${t("commons.status.success")}`
	})
}

async function onComposeOperate(operation: string) {
	let mes =
		operation === "down"
			? t("container.composeDownHelper", [composeName.value])
			: t("container.composeOperatorHelper", [composeName.value, t(`container.${operation}`)])

	dialog.warning({
		title: t(`container.${operation}`),
		content: mes,
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			let params = {
				name: composeName.value,
				path: composePath.value,
				operation,
				withFile: false
			}
			loading.value = true
			await composeOperator(params)
				.then(() => {
					loading.value = false
					MsgSuccess(t("commons.msg.operationSuccess"))
					search()
				})
				.catch(() => {
					loading.value = false
				})
		},
		onNegativeClick: () => {
			// 取消时的逻辑（可选）
		}
	})
}

const dialogMonitorRef = ref()
function onMonitor(row: any) {
	dialogMonitorRef.value!.acceptParams({
		containerID: row.containerID,
		container: row.name,
		runtimeSummary: buildRuntimeSummaryText(row, {
			kindFallback: t("container.runtimeType"),
			rootlessLabel: t("container.rootless"),
			rootfulLabel: t("container.rootful"),
			defaultModeLabel: t("container.defaultMode"),
			userFallback: t("container.userDefault"),
			runUserPrefix: `${t("container.runUser")}: `
		})
	})
}

const dialogTerminalRef = ref()
function onTerminal(row: any) {
	dialogTerminalRef.value!.acceptParams({
		containerID: row.containerID,
		container: row.name,
		runtimeHost: row.runtimeHost || "",
		runtimeSummary: buildRuntimeSummaryText(row, {
			kindFallback: t("container.runtimeType"),
			rootlessLabel: t("container.rootless"),
			rootfulLabel: t("container.rootful"),
			defaultModeLabel: t("container.defaultMode"),
			userFallback: t("container.userDefault"),
			runUserPrefix: `${t("container.runUser")}: `
		})
	})
}

const buttons = [
	{
		label: t("file.terminal"),
		disabled: (row: Container.ContainerInfo) => {
			return row.state !== "running"
		},
		click: (row: Container.ContainerInfo) => {
			onTerminal(row)
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
		label: t("commons.button.log"),
		click: (row: Container.ContainerInfo) => {
			dialogContainerLogRef.value!.acceptParams({
				containerID: row.containerID,
				container: row.name,
				runtimeHost: row.runtimeHost || "",
				runtimeSummary: buildRuntimeSummaryText(row, {
					kindFallback: t("container.runtimeType"),
					rootlessLabel: t("container.rootless"),
					rootfulLabel: t("container.rootful"),
					defaultModeLabel: t("container.defaultMode"),
					userFallback: t("container.userDefault"),
					runUserPrefix: `${t("container.runUser")}: `
				})
			})
		}
	}
]

defineExpose({
	acceptParams
})
</script>

<style lang="scss" scoped>
.app-card {
	font-size: 14px;
	height: 60px;
}

.app-content {
	height: 50px;
}

body {
	margin: 0;
}
</style>
