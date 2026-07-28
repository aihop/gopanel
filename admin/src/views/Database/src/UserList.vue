<script setup lang="ts">
import { NButton, NFlex, NInput, NPopconfirm, NTag } from "naive-ui"

import { formatTime } from "@/utils/date"
import UpdateUserModal from "./UpdateUserModal.vue"
import { useI18n } from "vue-i18n"
import { ref, h, reactive, onMounted, onUnmounted, inject, Ref, watch } from "vue"
import emitter from "@/utils/emitter"
import {
	databaseUserListAPI,
	databaseUserDeleteAPI,
	databaseUserRemarkAPI,
	databaseUserCountAPI
} from "@/api/modules/database"
import { useTable } from "@/composables/useTable"
import { MsgSuccess } from "@/utils/message"
import { isSucc } from "@/utils/is"
import { useMessage } from "naive-ui"

const { t } = useI18n()

const updateModal = ref(false)
const updateID = ref(0)
const updateServerId = ref(0)
const updateUsername = ref("")
const updateHost = ref("")
const page = ref(1)
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
						switch (row.server.type) {
							case "mysql":
								return "MySQL"
							case "postgresql":
								return "PostgreSQL"
							default:
								return row.server.type
						}
					}
				}
			)
		}
	},
	{
		title: t("database.username"),
		key: "username",
		width: 120,
		resizable: true,
		ellipsis: { tooltip: true },
		render(row: any) {
			return row.username || t("None")
		}
	},
	{
		title: t("database.host"),
		key: "host",
		width: 150,
		render(row: any) {
			return h(NTag, null, {
				default: () => row.host || t("None")
			})
		}
	},
	{
		title: t("database.server"),
		key: "server",
		width: 150,
		render(row: any) {
			return row.server.name
		}
	},
	{
		title: t("database.privileges"),
		key: "privileges",
		width: 150,
		render(row: any) {
			if (!Array.isArray(row.privileges) || row.privileges.length === 0) {
				return t("None")
			}
			return h(NFlex, null, {
				default: () =>
					row.privileges.map((privilege: string) =>
						h(NTag, null, {
							default: () => privilege
						})
					)
			})
		}
	},
	{
		title: t("database.comment"),
		key: "remark",
		width: 200,
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
		width: 150,
		hideInExcel: true,
		fixed: "right",
		render(row: any) {
			return [
				h(
					NButton,
					{
						size: "small",
						type: "primary",
						onClick: () => {
							updateID.value = row.id
							updateServerId.value = row.serverId
							updateUsername.value = row.username
							updateHost.value = row.host
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
						onPositiveClick: () => handleDelete(row.id,row.serverId,row.username)
					},
					{
						default: () => {
							return t("database.deleteUserTips")
						},
						trigger: () => {
							return h(
								NButton,
								{
									size: "small",
									type: "error",
									secondary: true,
									style: "margin-left: 15px;"
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
	listAPI: databaseUserListAPI,
	countAPI: databaseUserCountAPI,
	params: {
		wheres: buildWheres(globalSelectedServerId?.value || null),
		preloads: [
			{
				table: "Server"
			}
		]
	}
})

const {
	list,
	pages,
	total,
	curPage,
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

const handleDelete = (id: number,serverId: number,username: string) => {
	databaseUserDeleteAPI({ id,serverId,username: username }).then((res: any) => {
		if (isSucc(res.code)) {
			getData()
			MsgSuccess(res.msg)
		}
	})
}

const handleRemark = (row: any) => {
	databaseUserRemarkAPI({ id: row.id, remark: row.remark }).then((res: any) => {
		if (isSucc(res.code)) {
			getData()
			MsgSuccess(res.msg)
		}
	})
}



onMounted(() => {
	emitter.on("database-user:refresh", () => {
		getData()
	})
})

onUnmounted(() => {
	emitter.off("database-user:refresh")
})
</script>

<template>
  <n-data-table
    striped
    remote
    :scroll-x="1800"
    :loading="loading"
    :columns="columns"
    :data="list"
    :row-key="(row) => row.name"
    v-model:page="page"
    v-model:pageSize="pageSize"
    :pagination="{
			page: page,
			pageCount: pages,
			pageSize: pageSize,
			itemCount: total,
			showQuickJumper: true,
			showSizePicker: true,
			pageSizes: [20, 50, 100, 200]
		}"
  />
  <update-user-modal
    v-model:show="updateModal"
    v-model:id="updateID"
    v-model:server-id="updateServerId"
    v-model:username="updateUsername"
    v-model:host="updateHost"
  />
</template>
