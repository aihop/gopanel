<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useMessage, useDialog } from "naive-ui"
import CommonPage from "@/components/page/Common.vue"
import { cronjobListAPI, cronjobSetStatusAPI, cronjobRunAPI, cronjobDeleteAPI } from "@/api/modules/cronjob"
import { createCronjobColumns } from "./cronjobColumns"
import CronjobModal from "./components/CronjobModal.vue"
import RecordDrawer from "./components/RecordDrawer.vue"

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const jobs = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const modalVisible = ref(false)
const editingJob = ref<any | null>(null)

const recordJob = ref<any | null>(null)
const recordDrawerVisible = ref(false)

const fetchData = async () => {
	loading.value = true
	try {
		const res: any = await cronjobListAPI({ page: page.value, limit: pageSize.value, wheres: [] })
		if (res.code === 0 && res.data) {
			jobs.value = res.data.items || []
			total.value = res.data.total || 0
		} else {
			jobs.value = []
			total.value = 0
		}
	} finally {
		loading.value = false
	}
}

const openCreate = () => {
	editingJob.value = null
	modalVisible.value = true
}

const openEdit = (row: any) => {
	editingJob.value = row
	modalVisible.value = true
}

const openRecords = (row: any) => {
	recordJob.value = row
	recordDrawerVisible.value = true
}

const handleRun = async (row: any) => {
	try {
		await cronjobRunAPI({ id: row.id })
		message.success("已提交执行，稍后可在执行记录中查看结果")
		setTimeout(fetchData, 1500)
	} catch {
		void 0
	}
}

const handleToggleStatus = async (row: any, enabled: boolean) => {
	try {
		await cronjobSetStatusAPI({ id: row.id, status: enabled ? "Enable" : "Disable" })
		message.success(enabled ? "已启用" : "已禁用")
		fetchData()
	} catch {
		void 0
	}
}

const handleDelete = (row: any) => {
	dialog.warning({
		title: "确认删除",
		content: `确定要删除计划任务 ${row.name} 吗？执行记录也会一并清除。`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await cronjobDeleteAPI({ id: row.id })
				message.success("删除成功")
				fetchData()
			} catch {
				void 0
			}
		}
	})
}

const columns = createCronjobColumns({
	openEdit,
	openRecords,
	handleRun,
	handleDelete,
	handleToggleStatus
})

const onSaved = () => {
	modalVisible.value = false
	fetchData()
}

onMounted(fetchData)
</script>

<template>
	<CommonPage>
		<div class="mb-4 flex justify-end">
			<n-button type="primary" @click="openCreate">新建计划任务</n-button>
		</div>
		<n-data-table
			striped
			remote
			:loading="loading"
			:columns="columns"
			:data="jobs"
			:row-key="(row: any) => row.id"
			:pagination="{
				page,
				pageSize,
				itemCount: total,
				showQuickJumper: true,
				showSizePicker: true,
				pageSizes: [20, 50, 100],
				onUpdatePage: (p: number) => {
					page = p
					fetchData()
				},
				onUpdatePageSize: (s: number) => {
					pageSize = s
					page = 1
					fetchData()
				}
			}"
		/>
		<CronjobModal v-model:show="modalVisible" :job="editingJob" @saved="onSaved" />
		<RecordDrawer v-model:show="recordDrawerVisible" :job="recordJob" />
	</CommonPage>
</template>
