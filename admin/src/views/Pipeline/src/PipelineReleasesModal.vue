<script setup lang="ts">
import { h, ref, watch } from "vue"
import { NDataTable, NModal, NTag, NPopover, useMessage, type DataTableColumns } from "naive-ui"
import { getPipelineReleases } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import { formatTime } from "@/utils/date"

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
  image: "镜像",
  archive: "归档",
  release_dir: "目录"
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
          { type: "warning", size: "small", style: "cursor:pointer;" },
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
      h("div", { class: "flex flex-wrap gap-1" }, [
        h(NTag, { type: "success", size: "small" }, { default: () => `v${row.version}` }),
        renderActiveWebsiteTag(row.activeWebsiteCount, row.activeWebsiteNames)
      ])
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
    title: "来源",
    key: "sourceType",
    width: 100,
    render: (row: Pipeline.ResRelease) => h(NTag, { size: "small", type: "info" }, {
      default: () => sourceTypeLabelMap[row.sourceType] || row.sourceType || "-"
    })
  },
  {
    title: "交付物",
    key: "artifact",
    minWidth: 320,
    render(row: Pipeline.ResRelease) {
      const artifact = row.imageTag || row.archiveFile || row.releaseDir || "-"
      return h("div", { class: "break-all text-xs text-slate-500" }, artifact)
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
    message.error(error.message || "获取正式版本失败")
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
      正式版本用于稳定发布与回滚；执行记录仅表示最新构建与自动部署状态，请先手动发布成功构建。
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
