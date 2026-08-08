<script setup lang="ts">
import { h, ref, watch } from "vue"
import { NDataTable, NModal, NTag, useMessage, type DataTableColumns } from "naive-ui"
import { getPipelineReleases } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import { formatTime } from "@/utils/date"
import { renderChangelogCell } from "@/utils/changelog"

const props = defineProps<{ show: boolean; pipelineId: number }>()
const emit = defineEmits(["update:show"])

const message = useMessage()
const loading = ref(false)
const data = ref<Pipeline.ResRelease[]>([])

const pagination = ref({
  page: 1,
  limit: 10,
  itemCount: 0,
  onChange: (page: number) => {
    pagination.value.page = page
    fetchData()
  }
})

const handleCopy = async (text: string, successText: string) => {
  const value = String(text || "").trim()
  if (!value) {
    message.warning("没有可复制的内容")
    return
  }
  try {
    await navigator.clipboard.writeText(value)
    message.success(successText)
  } catch (_error) {
    message.error("复制失败")
  }
}

const sourceTypeLabelMap: Record<string, string> = {
  image: "镜像引用",
  archive: "归档包",
  release_dir: "发布目录"
}

const parseArtifactMeta = (row: Pipeline.ResRelease) => {
  if (!row.artifactMeta) return {}
  try {
    const parsed = JSON.parse(row.artifactMeta)
    return parsed && typeof parsed === "object" ? parsed : {}
  } catch (_error) {
    return {}
  }
}

const getArtifactSummary = (row: Pipeline.ResRelease) => {
  const artifactMeta = parseArtifactMeta(row) as { runnerMode?: string }
  const isRunnerMode = (artifactMeta.runnerMode || "").toLowerCase() === "runner"

  if (row.sourceType === "image") {
    return {
      label: isRunnerMode ? "镜像引用" : "脚本识别镜像",
      value: row.imageTag || "-"
    }
  }

  if (row.sourceType === "archive") {
    return {
      label: isRunnerMode ? "代码归档包" : "脚本归档包",
      value: row.archiveFile || "-"
    }
  }

  return {
    label: isRunnerMode ? "发布目录" : "脚本发布目录",
    value: row.releaseDir || "-"
  }
}

const columns: DataTableColumns<Pipeline.ResRelease> = [
  { title: "ID", key: "id", width: 60 },
  {
    title: "发布时间",
    key: "createdAt",
    width: 180,
    render: (row: Pipeline.ResRelease) => formatTime(row.createdAt || "")
  },
  {
    title: "版本",
    key: "version",
    width: 220,
    render: (row: Pipeline.ResRelease) =>
      h(NTag, { type: "success", size: "small" }, { default: () => `v${row.version}` })
  },
  {
    title: "Commit",
    key: "commitHash",
    width: 180,
    render(row: Pipeline.ResRelease) {
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
    title: "更新内容",
    key: "changelog",
    minWidth: 240,
    render: (row: Pipeline.ResRelease) => renderChangelogCell(row.changelog)
  },
  {
    title: "结果来源",
    key: "sourceType",
    width: 100,
    render: (row: Pipeline.ResRelease) => h(NTag, { size: "small", type: "info" }, {
      default: () => sourceTypeLabelMap[row.sourceType] || row.sourceType || "-"
    })
  },
  {
    title: "交付结果",
    key: "artifact",
    minWidth: 320,
    render(row: Pipeline.ResRelease) {
      const artifact = getArtifactSummary(row)
      return h("div", { class: "flex flex-col gap-1 min-w-0" }, [
        h(
          NTag,
          { size: "small", type: row.sourceType === "image" ? "warning" : "default" },
          { default: () => artifact.label }
        ),
        h("div", { class: "break-all text-xs text-slate-500" }, artifact.value)
      ])
    }
  },
  {
    title: "状态",
    key: "status",
    width: 100,
    render: (row: Pipeline.ResRelease) => h(NTag, { size: "small", type: row.status === "ready" ? "success" : "warning" }, {
      default: () => row.status
    })
  },
  {
    title: "构建记录",
    key: "pipelineRecordId",
    width: 100,
    render: (row: Pipeline.ResRelease) => h("span", { class: "font-mono text-xs text-slate-500" }, `#${row.pipelineRecordId}`)
  }
]

const fetchData = async () => {
  if (!props.pipelineId) return
  loading.value = true
  try {
    const res = await getPipelineReleases({
      pipelineId: props.pipelineId,
      page: pagination.value.page,
      limit: pagination.value.limit
    })
    data.value = res.data.items
    pagination.value.itemCount = res.data.total
  } catch (error: any) {
    // 错误提示由请求拦截器统一处理
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
    title="正式版本"
    style="width: 1180px;"
    class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(15,23,42,0.12)] sm:w-[90%]"
    @update:show="(val) => emit('update:show', val)"
  >
    <div class="mb-4 text-sm text-slate-500">
      正式版本承接稳定上线、网站切换与回滚；这里展示每个版本的交付结果，以及当前有哪些站点正在使用它。
    </div>
    <div class="h-[520px] overflow-auto">
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
      />
    </div>
  </n-modal>
</template>
