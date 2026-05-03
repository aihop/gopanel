<script setup lang="ts">
import { h, ref, watch } from "vue"
import { NModal, NDataTable, NButton, NTag, NSpace, NPopconfirm, NPopover, useMessage, type DataTableColumns } from "naive-ui"
import { getPipelineRecords, runPipeline, stopPipeline, deletePipelineRecord, publishPipelineRelease } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import PipelineLogsModal from "./PipelineLogsModal.vue"
import { useAuthStore } from "@/store/auth"
import dayjs from "dayjs"
import { getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"
const props = defineProps<{ show: boolean; pipelineId: number }>()
const emit = defineEmits(["update:show"])

const message = useMessage()
const authStore = useAuthStore()
const loading = ref(false)
const data = ref<Pipeline.ResRecord[]>([])
const isSubAdmin = authStore.user?.role === "SUB_ADMIN"

const handleCopy = async (text: string, successText: string) => {
  const value = String(text || "").trim()
  if (!value) {
    message.warning("没有可复制的内容")
    return
  }
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
    } else {
      const textarea = document.createElement("textarea")
      textarea.value = value
      textarea.setAttribute("readonly", "true")
      textarea.style.position = "fixed"
      textarea.style.left = "-9999px"
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand("copy")
      document.body.removeChild(textarea)
    }
    message.success(successText)
  } catch (_error) {
    message.error("复制失败")
  }
}

const pagination = ref({
  page: 1,
  limit: 10,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.value.page = page
    fetchData()
  }
})

const logsModalShow = ref(false)
const currentRecordId = ref<number | null>(null)
const currentRecordVersion = ref<string>("")

const handleRetryFromLogs = async () => {
  try {
    const res = await runPipeline({ id: props.pipelineId, version: currentRecordVersion.value })
    message.success(`已重新触发执行，版本号: v${currentRecordVersion.value}`)
    
    if (res.data && res.data.recordId) {
      currentRecordId.value = res.data.recordId
    }
    
    fetchData()
  } catch (error: any) {
    message.error(error.message || "触发失败")
  }
}

const handleRerun = async (row: Pipeline.ResRecord) => {
  try {
    const res = await runPipeline({ id: props.pipelineId, version: row.version })
    message.success(`已重新触发执行，版本号: v${row.version}`)
    
    if (res.data && res.data.recordId) {
      currentRecordId.value = res.data.recordId
      currentRecordVersion.value = row.version || ""
      logsModalShow.value = true
    }
    
    fetchData()
  } catch (error: any) {
    message.error(error.message || "触发失败")
  }
}

const handleStop = async (row: Pipeline.ResRecord) => {
  try {
    await stopPipeline({ id: row.id })
    message.success("已发送强制停止指令")
    fetchData()
  } catch (error: any) {
    message.error(error.message || "停止失败")
  }
}

const handleDelete = async (row: Pipeline.ResRecord) => {
  try {
    await deletePipelineRecord({ id: row.id })
    message.success("删除成功")
    if (data.value.length === 1 && pagination.value.page > 1) {
      pagination.value.page -= 1
    }
    fetchData()
  } catch (error: any) {
    message.error(error.message || "删除失败")
  }
}

const handlePublishRelease = async (row: Pipeline.ResRecord) => {
  try {
    await publishPipelineRelease({ id: row.id })
    message.success(row.released ? "该记录已存在正式版本" : "已发布为正式版本")
    fetchData()
  } catch (error: any) {
    message.error(error.message || "发布正式版本失败")
  }
}

const renderActiveWebsiteTag = (count?: number, names?: string[]) => {
  if (!count) return null
  const websiteNames = Array.isArray(names) ? names.filter(Boolean) : []
  return h(
    NPopover,
    { trigger: "hover" },
    {
      trigger: () =>
        h(
          NTag,
          { size: "small", type: "warning", style: "cursor:pointer;" },
          { default: () => `线上 ${count} 站点` }
        ),
      default: () =>
        h(
          "div",
          { class: "max-w-[260px] space-y-1 text-xs text-slate-600" },
          websiteNames.length
            ? websiteNames.map((name) => h("div", { class: "break-all" }, name))
            : [h("div", "暂无站点信息")]
        )
    }
  )
}

const columns: DataTableColumns<Pipeline.ResRecord> = [
  { title: "ID", key: "id", width: 60 },
  { title: "创建时间", key: "createdAt", width: 180, render: (row: Pipeline.ResRecord) => row.createdAt ? dayjs(row.createdAt).format("YYYY-MM-DD HH:mm") : "-" },
  { title: "版本", key: "version", render: (row: Pipeline.ResRecord) => h(NTag, { type: "success", size: "small" }, { default: () => `v${row.version || '-'}` }) },
  {
    title: "Commit",
    key: "commitHash",
    width: 170,
    render(row: Pipeline.ResRecord) {
      if (!row.commitHash) {
        return h("span", { class: "text-slate-400 text-xs" }, "-")
      }
      return h("div", { class: "flex items-center gap-1 text-xs min-w-0" }, [
        h("span", { class: "font-mono text-slate-700 shrink-0" }, row.commitHash.slice(0, 12)),
        h("button", {
          type: "button",
          class: "shrink-0 text-[12px] leading-none text-blue-600 hover:text-blue-700",
          onClick: (event: MouseEvent) => {
            event.stopPropagation()
            void handleCopy(row.commitHash || "", "Commit SHA 已复制")
          }
        }, "复制")
      ])
    }
  },
  {
    title: "状态",
    key: "status",
    render(row: Pipeline.ResRecord) {
      let type: "default" | "info" | "success" | "warning" | "error" = "default"
      switch (row.status) {
        case "pending": type = "default"; break
        case "cloning": type = "info"; break
        case "building": type = "warning"; break
        case "deploying": type = "info"; break
        case "success": type = "success"; break
        case "failed": type = "error"; break
      }
      return h(NTag, { type }, { default: () => row.status })
    }
  },
  {
    title: "正式版本",
    key: "released",
    width: 160,
    render(row: Pipeline.ResRecord) {
      return h("div", { class: "flex flex-wrap gap-1" }, [
        h(
          NTag,
          { size: "small", type: row.released ? "success" : "default" },
          { default: () => (row.released ? "已发布" : "未发布") }
        ),
        renderActiveWebsiteTag(row.activeWebsiteCount, row.activeWebsiteNames)
      ])
    }
  },
  {
    title: "Runner",
    key: "runner",
    width: 260,
    render(row: Pipeline.ResRecord) {
      if (!row.runnerHostPort) {
        return h("span", { class: "text-slate-400 text-xs" }, "未启用 / 未产出")
      }
      return h("div", { class: "flex flex-col gap-0.5 text-xs min-w-0" }, [
        h("div", { class: "flex items-center gap-1 min-w-0" }, [
          h("span", { class: "font-mono text-emerald-600 shrink-0" }, `127.0.0.1:${row.runnerHostPort}`),
          h("button", {
            type: "button",
            class: "shrink-0 text-[12px] leading-none text-blue-600 hover:text-blue-700",
            onClick: (event: MouseEvent) => {
              event.stopPropagation()
              void handleCopy(`127.0.0.1:${row.runnerHostPort}`, "Runner 地址已复制")
            }
          }, "复制")
        ]),
        row.runnerContainerId ? h("span", { class: "font-mono text-slate-500" }, row.runnerContainerId.slice(0, 12)) : null
      ])
    }
  },
  {
    title: "运行类型",
    key: "runtimeType",
    width: 250,
    render(row: Pipeline.ResRecord) {
      if (!row.runnerContainerId) {
        return h("span", { class: "text-slate-400 text-xs" }, "无 Runner 容器")
      }
      return h("div", { class: "flex flex-col gap-1 text-xs" }, [
        h("div", { class: "flex flex-wrap items-center gap-2" }, [
          h(NTag, { size: "small", type: row.runtimeKind === "docker" ? "success" : "warning" }, {
            default: () => getRuntimeKindLabel(row, { kindFallback: "运行时" })
          }),
          h(NTag, { size: "small", type: row.runtimeMode === "rootless" ? "warning" : "default" }, {
            default: () => getRuntimeModeLabel(row)
          })
        ]),
        h("span", { class: "text-slate-500" }, `运行用户: ${getRunUserLabel(row)}`),
        row.runtimeHost
          ? h("span", { class: "text-slate-500 break-all" }, `Host: ${row.runtimeHost}`)
          : null
      ])
    }
  },
  { title: "错误信息", key: "errorMessage", ellipsis: true },
  {
    title: "操作",
    key: "actions",
    width: 160,
    fixed: "right",
    render(row: Pipeline.ResRecord, index: number) {
      const isFirstRow = index === 0 && pagination.value.page === 1;
      const isRunning = ["pending", "cloning", "building", "deploying"].includes(row.status);

      const btns = [
        h(NButton, {
          size: "small",
          type: "primary",
          ghost: true,
          onClick: () => {
            currentRecordId.value = row.id
            currentRecordVersion.value = row.version || ""
            logsModalShow.value = true
          }
        }, { default: () => "日志" })
      ];

      if (isFirstRow) {
        if (isRunning) {
          btns.push(
            h(NPopconfirm, {
              onPositiveClick: () => handleStop(row),
              positiveText: "确定停止"
            }, {
              trigger: () => h(NButton, {
                size: "small",
                type: "error",
                ghost: true
              }, { default: () => "强制停止" }),
              default: () => "确定要强制终止正在执行的流水线吗？"
            })
          )
        } else {
          btns.push(
            h(NButton, {
              size: "small",
              type: "warning",
              ghost: true,
              onClick: () => handleRerun(row)
            }, { default: () => "重新执行" })
          )
        }
      }

      if (!isSubAdmin && row.status === "success") {
        btns.push(
          h(NButton, {
            size: "small",
            type: row.released ? "default" : "success",
            ghost: !row.released,
            disabled: !!row.released,
            onClick: () => handlePublishRelease(row)
          }, { default: () => (row.released ? "已发布" : "发布为正式版本") })
        )
      }

      if (!isSubAdmin && !isRunning) {
        btns.push(
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row),
            positiveText: "确定删除"
          }, {
            trigger: () => h(NButton, {
              size: "small",
              type: "error",
              ghost: true
            }, { default: () => "删除" }),
            default: () => "确定要删除这条执行记录吗？"
          })
        )
      }

      return h(NSpace, {}, { default: () => btns })
    }
  }
]

const fetchData = async () => {
  if (!props.pipelineId) return
  loading.value = true
  try {
    const res = await getPipelineRecords({
      pipelineId: props.pipelineId,
      page: pagination.value.page,
      limit: pagination.value.limit
    })
    data.value = res.data.items
    pagination.value.itemCount = res.data.total
  } catch (error: any) {
    message.error(error.message || "获取记录失败")
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      fetchData()
    }
  },
  { immediate: true }
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="执行记录"
    style="width:1080px;"
    class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(15,23,42,0.12)] sm:w-[90%]"
    @update:show="(val) => emit('update:show', val)"
  >
    <div class="mb-4 text-sm text-slate-500">
      执行记录反映每次构建与自动部署结果；如需稳定发布与回滚，请手动发布为正式版本。
    </div>
    <div class="h-[500px] overflow-auto">
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
      />
    </div>

    <PipelineLogsModal
      v-if="currentRecordId"
      v-model:show="logsModalShow"
      :record-id="currentRecordId"
      :pipeline-id="props.pipelineId"
      @finished="fetchData"
      @retry="handleRetryFromLogs"
    />
  </n-modal>
</template>
