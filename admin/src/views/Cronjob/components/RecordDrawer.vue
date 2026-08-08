<script setup lang="ts">
import { ref, watch, h } from "vue"
import { NDrawer, NDrawerContent, NDataTable, NButton, NTag, useMessage, useDialog } from "naive-ui"
import { cronjobRecordListAPI, cronjobRecordDeleteAPI } from "@/api/modules/cronjob"
import FtEditor from "@/components/FtEditor/index.vue"
import { typeLabels } from "../cronjobColumns"

const props = defineProps<{
	show: boolean
	job: any | null
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
}>()

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const records = ref<any[]>([])
const detailVisible = ref(false)
const detailMessage = ref("")

const statusTag = (status: string) => {
	const map: Record<string, "success" | "error" | "warning" | "default"> = {
		Running: "warning",
		Success: "success",
		Failed: "error"
	}
	return map[status] || "default"
}

const columns = [
	{
		title: "开始时间",
		key: "startTime",
		render: (row: any) => row.startTime?.replace("T", " ").slice(0, 19) || "-"
	},
	{
		title: "结束时间",
		key: "endTime",
		render: (row: any) => (row.endTime && !row.endTime.startsWith("0001") ? row.endTime.replace("T", " ").slice(0, 19) : "-")
	},
	{
		title: "状态",
		key: "status",
		render: (row: any) => h(NTag, { size: "small", type: statusTag(row.status), bordered: false }, { default: () => row.status })
	},
	{
		title: "操作",
		key: "actions",
		render: (row: any) =>
			h(
				NButton,
				{
					text: true,
					type: "primary",
					onClick: () => {
						detailMessage.value = row.message || "（无输出）"
						detailVisible.value = true
					}
				},
				{ default: () => "查看详情" }
			)
	}
]

const fetchRecords = async () => {
	if (!props.job) return
	loading.value = true
	try {
		const res: any = await cronjobRecordListAPI({ cronjobID: props.job.id, limit: 50 })
		records.value = res.code === 0 ? res.data || [] : []
	} finally {
		loading.value = false
	}
}

const handleClear = () => {
	if (!props.job) return
	dialog.warning({
		title: "确认清空",
		content: "确定要清空该计划任务的全部执行记录吗？",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await cronjobRecordDeleteAPI({ cronjobID: props.job!.id })
				message.success("已清空")
				fetchRecords()
			} catch {
				// 错误提示由请求拦截器统一处理
			}
		}
	})
}

watch(
	() => props.show,
	visible => {
		if (visible) fetchRecords()
	}
)

const close = () => emit("update:show", false)
</script>

<template>
	<n-drawer :show="show" :width="700" @update:show="(val: boolean) => !val && close()">
		<n-drawer-content closable>
			<template #header>
				<div class="flex items-center gap-4">
					<span>{{ job ? `${job.name}（${typeLabels[job.type] || job.type}）执行记录` : "执行记录" }}</span>
					<n-button text type="error" @click="handleClear">清空记录</n-button>
				</div>
			</template>
			<n-data-table :loading="loading" :columns="columns" :data="records" :row-key="(row: any) => row.id" />
			<n-modal :show="detailVisible" preset="card" title="执行详情" style="width: 640px" @update:show="(v: boolean) => (detailVisible = v)">
				<FtEditor :model-value="detailMessage" :readonly="true" :show-toolbar="false" height="50vh" language="shell" />
			</n-modal>
		</n-drawer-content>
	</n-drawer>
</template>
