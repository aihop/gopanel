<script setup lang="ts">
import { h, onMounted, ref, computed } from "vue"
import { NButton, NDataTable, NSpace, NTag, useMessage, useDialog, NModal, NForm, NFormItem, NInput } from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import { getPipelinePage, deletePipeline, runPipeline } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import PipelineRecordsModal from "./PipelineRecordsModal.vue"
import PipelineLogsModal from "./PipelineLogsModal.vue"
import PipelineReleasesModal from "./PipelineReleasesModal.vue"
import { useAuthStore } from "@/store/auth"
import { t } from "@/i18n"
import { buildRuntimeDetailText, getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"
import { formatTime } from "@/utils/date"
const authStore = useAuthStore()
const isSubAdmin = computed(() => authStore.user?.role === 'SUB_ADMIN')
const isSuperAdmin = computed(() => authStore.user?.role === 'SUPER')

const emit = defineEmits(["edit"])

const message = useMessage()
const dialog = useDialog()
const data = ref<Pipeline.ResPipeline[]>([])
const loading = ref(false)
const pagination = ref({
  page: 1,
  limit: 10,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.value.page = page
    fetchData()
  }
})

const recordsModalShow = ref(false)
const releasesModalShow = ref(false)
const currentPipelineId = ref<number | null>(null)

// 执行版本号弹窗
const runModalShow = ref(false)
const runLoading = ref(false)
const currentRunRow = ref<Pipeline.ResPipeline | null>(null)
const runFormModel = ref({ version: "" })

// 执行后直接看日志
const logsModalShow = ref(false)
const currentRecordId = ref<number | null>(null)

// 辅助函数：自动递增版本号 (例如 1.0.0 -> 1.0.1)
const incrementVersion = (v: string) => {
  const parts = v.split('.')
  if (parts.length > 0 && !isNaN(Number(parts[parts.length - 1]))) {
    parts[parts.length - 1] = String(Number(parts[parts.length - 1]) + 1)
    return parts.join('.')
  }
  return v + ".1"
}

const buildPipelineRuntimeText = (row: Pipeline.ResPipeline | null, options?: { prefix?: string; runUserPrefix?: string }) => {
  if (!row) return ""
  const detail = buildRuntimeDetailText(row, {
    prefix: options?.prefix || "",
    kindFallback: t("container.runtimeType"),
    userFallback: t("container.userDefault"),
    runtimePrefix: "",
    runUserPrefix: options?.runUserPrefix || `${t("container.runUser")}: `
  })
  const host = String(row.runtimeHost || "").trim()
  return host ? `${detail} · Host: ${host}` : detail
}

const columns: DataTableColumns<Pipeline.ResPipeline> = [
  { title: "ID", key: "id", width: 60 },
  { title: t("pipeline.pipelineName"), key: "name" },
  {
    title: "模式",
    key: "runnerMode",
    render: (row: Pipeline.ResPipeline) => {
      const isRunner = (row.runnerMode || "").toLowerCase() === "runner"
      return h(NTag, { type: isRunner ? "success" : "warning", size: "small" }, {
        default: () => isRunner ? "代码产物部署" : "纯脚本"
      })
    }
  },
  {
    title: t("container.runtimeType"),
    key: "runtimeType",
    minWidth: 300,
    render: (row: Pipeline.ResPipeline) => {
      return h("div", { class: "flex flex-col gap-1" }, [
        h("div", { class: "flex flex-wrap items-center gap-2" }, [
          h(NTag, { type: row.runtimeKind === "docker" ? "success" : "warning", size: "small" }, {
            default: () => getRuntimeKindLabel(row, { kindFallback: t("container.runtimeType") })
          }),
          h(NTag, { type: row.runtimeMode === "rootless" ? "warning" : "default", size: "small" }, {
            default: () => getRuntimeModeLabel(row, {
              rootlessLabel: t("container.rootless"),
              rootfulLabel: t("container.rootful"),
              defaultModeLabel: t("container.defaultMode")
            })
          })
        ]),
        h("div", { class: "text-xs text-slate-500" }, `${t("container.runUser")}: ${getRunUserLabel(row, { userFallback: t("container.userDefault") })}`),
        row.runtimeHost
          ? h("div", { class: "text-xs text-slate-500 break-all" }, `Host: ${row.runtimeHost}`)
          : null
      ])
    }
  },
  { title: t("pipeline.branch"), key: "branch", render: (row: Pipeline.ResPipeline) => h(NTag, { type: "info", size: "small" }, { default: () => row.branch }) },
  { title: t("pipeline.currentVersion"), key: "version", render: (row: Pipeline.ResPipeline) => h(NTag, { type: "success", size: "small" }, { default: () => `v${row.version}` }) },
  { title: t("pipeline.artifactPath"), key: "artifactPath" },
  { title: t("commons.table.createdAt"), key: "createdAt", render: (row: Pipeline.ResPipeline) => h("div", { class: "text-xs text-slate-500" }, formatTime(row.createdAt || "")) },
  {
    title: t("pipeline.actions"),
    key: "actions",
    width: 240,
    fixed: "right",
    render(row: Pipeline.ResPipeline) {
      return h(NSpace, {}, {
        default: () => {
          const btns = [
            h(NButton, {
              size: "small",
              type: "primary",
              onClick: () => handleRun(row)
            }, { default: () => t("pipeline.run") }),
            h(NButton, {
              size: "small",
              onClick: () => handleViewRecords(row)
            }, { default: () => t("pipeline.records") }),
            h(NButton, {
              size: "small",
              onClick: () => handleViewReleases(row)
            }, { default: () => "正式版本" }),
          ]
          
          if (isSuperAdmin.value) {
            btns.push(
              h(NButton, {
                size: "small",
                type: "warning",
                ghost: true,
                onClick: () => emit("edit", row)
              }, { default: () => t("pipeline.edit") })
            )
          }

          if (!isSubAdmin.value) {
            btns.push(
              h(NButton, {
                size: "small",
                type: "error",
                ghost: true,
                onClick: () => handleDelete(row)
              }, { default: () => t("pipeline.delete") })
            )
          }
          
          return btns
        }
      })
    }
  }
]

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getPipelinePage({
      page: pagination.value.page,
      limit: pagination.value.limit
    })
    data.value = res.data.items
    pagination.value.itemCount = res.data.total
  } catch (error: any) {
    message.error(error.message || "获取列表失败")
  } finally {
    loading.value = false
  }
}

const handleRun = (row: Pipeline.ResPipeline) => {
  currentRunRow.value = row
  runFormModel.value.version = incrementVersion(row.version)
  runModalShow.value = true
}

const confirmRun = async () => {
  if (!currentRunRow.value) return
  if (!runFormModel.value.version) {
    message.warning("请输入本次构建的版本号")
    return
  }

  runLoading.value = true
  try {
    const res = await runPipeline({ 
      id: currentRunRow.value.id,
      version: runFormModel.value.version
    })
    message.success(`已触发流水线执行，版本号: v${runFormModel.value.version}`)
    runModalShow.value = false
    
    // 打开日志流弹窗
    if (res.data && res.data.recordId) {
      currentRecordId.value = res.data.recordId
      logsModalShow.value = true
    } else {
      handleViewRecords(currentRunRow.value)
    }
    
    fetchData() // 刷新列表以更新当前版本号
  } catch (error: any) {
    message.error(error.message || "触发失败")
  } finally {
    runLoading.value = false
  }
}

const handleRetryFromLogs = async () => {
  if (!currentRunRow.value) return
  runLoading.value = true
  try {
    const res = await runPipeline({ 
      id: currentRunRow.value.id,
      version: runFormModel.value.version
    })
    message.success(`已重新触发流水线执行，版本号: v${runFormModel.value.version}`)
    
    if (res.data && res.data.recordId) {
      currentRecordId.value = res.data.recordId
    }
    
    fetchData()
  } catch (error: any) {
    message.error(error.message || "重新触发失败")
  } finally {
    runLoading.value = false
  }
}

const handleViewRecords = (row: Pipeline.ResPipeline) => {
  currentPipelineId.value = row.id
  recordsModalShow.value = true
}

const handleViewReleases = (row: Pipeline.ResPipeline) => {
  currentPipelineId.value = row.id
  releasesModalShow.value = true
}


const handleDelete = async (row: Pipeline.ResPipeline) => {
  await dialog.warning({
    title: "确认删除",
    content: `确定要删除流水线 ${row.name} 吗？`,
    positiveText: "确定",
    negativeText: "取消",
    onPositiveClick: async () => {
      try {
        await deletePipeline({ id: row.id })
        message.success("删除成功")
        fetchData()
      } catch (error: any) {
        message.error(error.message || "删除失败")
      }
    }
  })
}

onMounted(() => {
  fetchData()
})

defineExpose({
  refresh: fetchData
})
</script>

<template>
  <div class="w-full">
    <n-data-table
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      :bordered="false"
      :scroll-x="1080"
      class="bg-transparent"
    />
    <PipelineRecordsModal
      v-if="currentPipelineId"
      v-model:show="recordsModalShow"
      :pipeline-id="currentPipelineId"
    />
    <PipelineReleasesModal
      v-if="currentPipelineId"
      v-model:show="releasesModalShow"
      :pipeline-id="currentPipelineId"
    />

    <!-- 新增流水线日志弹窗 -->
    <PipelineLogsModal
      v-if="currentRecordId"
      v-model:show="logsModalShow"
      :record-id="currentRecordId"
      :pipeline-id="currentRunRow?.id || currentPipelineId"
      @finished="fetchData"
      @retry="handleRetryFromLogs"
    />

    <!-- 执行流水线配置版本号弹窗 -->
    <n-modal
      v-model:show="runModalShow"
      preset="card"
      title="配置构建版本"
      style="width: 400px;"
      class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(15,23,42,0.12)] sm:w-[90%]"
    >
      <div
        v-if="currentRunRow"
        class="mb-4 text-sm text-slate-500"
      >
        正在触发 <span class="font-semibold text-blue-600">{{ currentRunRow.name }}</span> 流水线<br />
        当前版本: <span class="font-mono font-semibold">v{{ currentRunRow.version }}</span>
      </div>
      <div
        v-if="currentRunRow"
        class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600"
      >
        {{ buildPipelineRuntimeText(currentRunRow) }}
      </div>
      <n-form
        :model="runFormModel"
        label-placement="left"
        label-width="80"
      >
        <n-form-item label="构建版本">
          <n-input
            v-model:value="runFormModel.version"
            placeholder="请填写新的版本号"
          />
        </n-form-item>
        <div class="mt-4 flex justify-end gap-3">
          <n-button @click="runModalShow = false">取消</n-button>
          <n-button
            type="primary"
            :loading="runLoading"
            @click="confirmRun"
          >确认执行</n-button>
        </div>
      </n-form>
    </n-modal>
  </div>
</template>
