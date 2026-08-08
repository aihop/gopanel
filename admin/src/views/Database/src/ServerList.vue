<script setup lang="ts">
import { NButton, NInput, NPopconfirm, NTag } from "naive-ui"

import { formatTime } from "@/utils/date"
import UpdateServerModal from "./UpdateServerModal.vue"
import { useI18n } from "vue-i18n"
import { h, onMounted, onUnmounted, reactive, ref } from "vue"
import emitter from "@/utils/emitter"
import { useTable } from "@/composables/useTable"
import {
	databaseServerListAPI,
	databaseServerCountAPI,
	databaseServerDeleteAPI,
	databaseServerSyncAPI,
	databaseServerSyncContainersAPI,
	type DatabaseContainerSyncResult
} from "@/api/modules/database"
import { MsgSuccess, MsgError, MsgWarning } from "@/utils/message"
import { isSucc } from "@/utils/is"

const { t } = useI18n()
const updateModal = ref(false)
const updateID = ref(0)
const pageCount = ref(0)
const page = ref(0)
const containerSyncing = ref(false)

const columns: any = [
	{
		title: t("Type"),
		key: "type",
		width: 100,
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
							default:
								return row.type
						}
					}
				}
			)
		}
	},
	{
		title: t("Name"),
		key: "name",
		width: 120,
		resizable: true,
		ellipsis: { tooltip: true }
	},
	{
		title: t("Username"),
		key: "username",
		width: 120,
		ellipsis: { tooltip: true },
		render(row: any) {
			return row.username || t("None")
		}
	},
	{
		title: t("Host"),
		key: "host",
		width: 180,
		render(row: any) {
			return h(NTag, null, {
				default: () => `${row.host}:${row.port}`
			})
		}
	},
	{
		title: t("database.comment"),
		key: "remark",
		width: 180,
		resizable: true,
		ellipsis: { tooltip: true },
		render(row: any) {
			return h(NInput, {
				size: "small",
				type: "textarea",
				value: row.remark,
				onBlur: () => handleRemark(row),
				onUpdateValue(v) {
					row.remark = v
				}
			})
		}
	},
	{
		title: t("database.status"),
		key: "status",
		width: 100,
		render(row: any) {
			return h(
				NTag,
				{ type: row.status === 20 ? "success" : "error" },
				{ default: () => (row.status === 20 ? t("database.valid") : t("database.invalid")) }
			)
		}
	},
	{
		title: t("database.updateTime"),
		key: "updatedAt",
		width: 200,
		ellipsis: { tooltip: true },
		render(row: any) {
			return formatTime(row.updatedAt)
		}
	},
	{
		title: t("database.actions"),
		key: "actions",
		width: 220,
		hideInExcel: true,
		fixed: "right",
		render(row: any) {
			return [
				h(
					NPopconfirm,
					{
						onPositiveClick: () => {
							databaseServerSyncAPI({ id: row.id }).then(res => {
								if (isSucc(res.code)) {
									MsgSuccess(res.msg)
									getData()
								}
							})
						}
					},
					{
						default: () => {
							return t("database.serverSyncTips")
						},
						trigger: () => {
							return h(
								NButton,
								{
									size: "small",
									type: "success"
								},
								{
									default: () => t("database.sync")
								}
							)
						}
					}
				),
				h(
					NButton,
					{
						size: "small",
						type: "primary",
						style: "margin-left: 15px;",
						onClick: () => {
							updateID.value = row.id
							updateModal.value = true
						}
					},
					{
						default: () => t("database.modify")
					}
				),
				h(
					NPopconfirm,
					{
						onPositiveClick: () => {
							// 防手贱
							if (["local_mysql", "local_postgresql"].includes(row.name)) {
								MsgError(t("database.serverDeleteHelperTips"))
								return
							}
							handleDelete(row.id)
						}
					},
					{
						default: () => {
							return t("database.serverDeleteTips")
						},
						trigger: () => {
							return h(
								NButton,
								{
									size: "small",
									type: "error",
									style: "margin-left: 15px;",
									secondary: true
								},
								{
									default: () => t("database.delete")
								}
							)
						}
					}
				)
			]
		}
	}
]

const params = reactive({
	listAPI: databaseServerListAPI,
	countAPI: databaseServerCountAPI,
	params: {
		wheres: []
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

const handleDelete = (id: number) => {
	databaseServerDeleteAPI({ id }).then(res => {
		getData()
		MsgSuccess(res.msg)
	})
}

const handleContainerSync = async () => {
	if (containerSyncing.value) return
	containerSyncing.value = true
	try {
		const res = await databaseServerSyncContainersAPI()
		if (!isSucc(res.code)) {
			MsgError(res.msg || t("commons.res.commonError"))
			return
		}
		const result = res.data as DatabaseContainerSyncResult
		await getData()
		const message = [
			`${t("commons.status.created")} ${result.created}`,
			`${t("commons.operate.update")} ${result.updated}`,
			`${t("commons.button.skip")} ${result.skipped}`,
			`${t("commons.status.failed")} ${result.failed}`
		].join(" / ")
		if (result.failed > 0) {
			MsgWarning(message, 5000)
		} else {
			MsgSuccess(message, 5000)
		}
	} catch {
		void 0
	} finally {
		containerSyncing.value = false
	}
}

getData()

const handleRemark = (row: any) => {}

onMounted(() => {
	emitter.on("database-server:refresh", () => {
		getData()
	})
})

onUnmounted(() => {
	emitter.off("database-server:refresh")
})
</script>

<template>
	<div class="mb-4 flex justify-end">
		<n-button type="primary" :loading="containerSyncing" :disabled="containerSyncing" @click="handleContainerSync">
			<template #icon>
				<Icon name="mdi:database-sync" />
			</template>
			{{ `${$t("container.containerList")} · ${$t("commons.button.sync")}` }}
		</n-button>
	</div>
	<n-data-table
		striped
		remote
		:scroll-x="1700"
		:loading="loading"
		:columns="columns"
		:data="list"
		:row-key="(row: any) => row.name"
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
	<update-server-modal v-model:id="updateID" v-model:show="updateModal" />
</template>
