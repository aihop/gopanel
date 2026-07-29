<script setup lang="ts">
import { NButton, NInput, NPopconfirm, NTag } from "naive-ui"
import { useI18n } from "vue-i18n"
import { useTable } from "@/composables/useTable"
import { databaseListAPI, databaseCountAPI, databaseCommentAPI, databaseDeleteAPI } from "@/api/modules/database"
import { reactive, h, onMounted, onUnmounted, ref, watch, inject, Ref } from "vue"
import emitter from "@/utils/emitter"
import Backup from "@/components/Backup.vue"
import FullModal from "@/components/FullModal.vue"
import UploadDialog from "@/components/UploadDialog.vue"
import DatabaseManager from "./DatabaseManager.vue"
import { MsgSuccess } from "@/utils/message"

const { t } = useI18n()
const dialogBackupRef = ref()
const uploadRef = ref()
const pageCount = ref(0)
const page = ref(1)

const managerModalShow = ref(false)
const managerServerId = ref<number | null>(null)
const managerDatabaseName = ref<string | null>(null)

const openManager = (row: any) => {
	managerServerId.value = row.serverId
	managerDatabaseName.value = row.name
	managerModalShow.value = true
}

const columns: any = [
	{
		title: t("database.type"),
		key: "type",
		width: 150,
		render(row: any) {
			return h(
				NTag,
				{ type: "info" },
				{
					default: () => {
						switch (row.type) {
							case "mysql":
								return "MySQL"
							case "postgresql":
								return "PostgreSQL"
							case "sqlite":
								return "SQLite"
							default:
								return row.type
						}
					}
				}
			)
		}
	},
	{
		title: t("database.databaseName"),
		key: "name",
		minWidth: 100,
		resizable: true,
		ellipsis: { tooltip: true }
	},
	{
		title: t("database.server"),
		key: "server",
		width: 150
	},
	{
		title: t("database.encoding"),
		key: "encoding",
		width: 150,
		render(row: any) {
			return h(NTag, null, {
				default: () => row.encoding
			})
		}
	},
	{
		title: t("database.comment"),
		key: "comment",
		minWidth: 200,
		resizable: true,
		ellipsis: { tooltip: true },
		render(row: any) {
			return h(NInput, {
				size: "small",
				type: "textarea",
				value: row.comment,
				onBlur: () => handleComment(row),
				onUpdateValue(v) {
					row.comment = v
				}
			})
		}
	},
	{
		title: t("database.actions"),
		key: "actions",
		width: 250,
		hideInExcel: true,
		fixed: "right",
		render(row: any) {
			const allowOps = row.type !== "sqlite"
			return [
				h(
					NButton,
					{
						size: "small",
						type: "primary",
						onClick: () => openManager(row)
					},
					{ default: () => t("commons.button.manage", "管理") }
				),
				// 备份按钮
				allowOps ? h(
					NButton,
					{
						size: "small",
						style: "margin-left: 8px;",
						onClick: () => openBackup(row)
					},
					{ default: () => t("commons.button.backup") }
				) : null,
				// 导入按钮
				allowOps ? h(
					NButton,
					{
						size: "small",
						style: "margin-left: 8px;",
						onClick: () => openRecover(row)
					},
					{ default: () => t("commons.button.recover") }
				) : null,
				allowOps ? h(
					NPopconfirm,
					{
						onPositiveClick: () => handleDelete(row.serverId, row.name)
					},
					{
						default: () => {
							return t("database.deleteDatabaseTips")
						},
						trigger: () => {
							return h(
								NButton,
								{
									size: "small",
									type: "error",
									style: "margin-left: 8px;",
									secondary: true
								},
								{
									default: () => t("database.delete")
								}
							)
						}
					}
				) : null
			]
		}
	}
]

const globalSelectedServerId = inject<Ref<number | null>>("globalSelectedServerId")

const buildWheres = (serverId: number | null) => {
	if (!serverId) return []
	return [
		{
			field: "server_id",
			rule: "eq",
			val: serverId.toString()
		}
	]
}

const params = reactive({
	listAPI: databaseListAPI,
	countAPI: databaseCountAPI,
	params: {
		wheres: buildWheres(globalSelectedServerId?.value || null)
	}
})

const {
	list,
	pages,
	curPage,
	total,
	pageSize,
	getList,
	loading,
	getData,
	onPageSizeChange,
	onPageChange,
	pageSizeOptions
} = useTable(params)

watch(() => globalSelectedServerId?.value, (newVal) => {
	params.params.wheres = buildWheres(newVal || null)
	page.value = 1
	getData()
}, { immediate: true })

const handleDelete = (serverId: number, name: string) => {
	databaseDeleteAPI({ serverId, name }).then(res => {
		getData()
		MsgSuccess(res.msg)
	})
}

const handleComment = (row: any) => {
	databaseCommentAPI({ serverId: row.serverId, name: row.name, comment: row.comment }).then(res => {
		MsgSuccess(res.msg)
	})
}

// 新增：打开备份面板
const openBackup = (row: any) => {
	if (!dialogBackupRef.value || !dialogBackupRef.value.acceptParams) return
	// 这里传入的参数可根据你的备份组件需求调整
	dialogBackupRef.value.acceptParams({
		type: row.type,
		name: row.server || "",
		detailName: row.name || "",
		detailId: row.serverId || 0,
		status: row.status || ""
	})
}

const openRecover = (row: any) => {
	if (!uploadRef.value || !uploadRef.value.acceptParams) return
	// 这里传入的参数可根据你的上传组件需求调整
	uploadRef.value.acceptParams({
		type: row.type,
		name: row.server || "",
		detailName: row.name || "",
		detailId: row.serverId || 0,
		status: row.status || ""
	})
}

const handleDatabaseRefresh = () => {
	getData()
}

onMounted(() => {
	emitter.on("database:refresh", handleDatabaseRefresh)
})

onUnmounted(() => {
	// 必须传入具体 handler：不带参数的 off 会清空该事件的全部监听器，
	// 连同 Backup.vue 等其他组件注册的一起移除
	emitter.off("database:refresh", handleDatabaseRefresh)
})
</script>

<template>
  <n-data-table
    striped
    remote
    :scroll-x="1000"
    :loading="loading"
    :columns="columns"
    :data="list"
    :row-key="(row) => row.name"
    v-model:page="page"
    v-model:pageSize="pageSize"
    :pagination="{
			page: page,
			pageCount: pageCount,
			pageSize: pageSize,
			itemCount: total,
			showQuickJumper: true,
			showSizePicker: true,
			pageSizes: [20, 50, 100, 200]
		}"
  />
  <Backup ref="dialogBackupRef" />
  <UploadDialog ref="uploadRef">
    <template #actions="{ openLocalRecover, loading: slotLoading, isUpload: slotUploading }">
      <n-button
        :disabled="slotLoading || slotUploading"
        @click="openLocalRecover"
      >
        输入本地地址
      </n-button>
    </template>
  </UploadDialog>
  <FullModal
    v-model:show="managerModalShow"
    :title="t('database.workspace')"
    width="min(1200px, calc(100vw - 48px))"
    height="min(860px, calc(100vh - 48px))"
    body-padding="10px"
  >
    <DatabaseManager
      fill-height
      :default-server-id="managerServerId"
      :default-database-name="managerDatabaseName"
    />
  </FullModal>
</template>
