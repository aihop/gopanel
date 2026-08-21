<script setup lang="ts">
import { NButton, NFlex, NInput, NPopconfirm, NTag } from "naive-ui"

import { formatTime } from "@/utils/date"
import DatabaseUserPasswordCell from "./DatabaseUserPasswordCell.vue"
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

const { t } = useI18n()

const updateModal = ref(false)
const updateID = ref(0)
const updateServerId = ref(0)
const updateUsername = ref("")
const updateHost = ref("")
const page = ref(1)

const rowIdentity = (row: any) => `${row.serverId || 0}|${row.username || ""}|${row.host || ""}`

const accessScopeLabel = (row: any) => {
	switch (row.accessScope) {
		case "all":
			return t("database.remoteAccess")
		case "local":
			return t("database.localOnly")
		case "specific":
			return t("database.specificHost")
		case "server":
			return t("database.serverAccessPolicy")
		default:
			return t("database.notApplicable")
	}
}

const accessScopeTagType = (row: any) => {
	switch (row.accessScope) {
		case "all":
			return "success"
		case "local":
			return "warning"
		case "specific":
			return "info"
		case "server":
			return "default"
		default:
			return "default"
	}
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
		title: t("database.hostAccessPolicy"),
		key: "host",
		width: 190,
		render(row: any) {
			if (row.accessScope === "server") {
				return h(NTag, { type: "info" }, { default: () => accessScopeLabel(row) })
			}
			return h(NFlex, { align: "center", size: 6 }, {
				default: () => [
					h(NTag, null, { default: () => row.host || t("None") }),
					h(NTag, { type: accessScopeTagType(row) }, { default: () => accessScopeLabel(row) })
				]
			})
		}
	},
	{
		title: t("database.password"),
		key: "password",
		width: 230,
		render(row: any) {
			return h(DatabaseUserPasswordCell, {
				id: row.id || 0,
				serverId: row.serverId,
				username: row.username,
				host: row.host || "",
				managed: !!row.passwordManaged
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
		title: t("database.passwordStatus"),
		key: "status",
		width: 120,
		render(row: any) {
			if (row.status === 0) {
				return h(NTag, { type: "warning" }, { default: () => t("database.passwordUnverified") })
			}
			return h(
				NTag,
				{ type: row.status === 20 ? "success" : "error" },
				{ default: () => (row.status === 20 ? t("database.passwordVerified") : t("database.passwordVerificationFailed")) }
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
						onPositiveClick: () => handleDelete(row.id, row.serverId, row.username, row.host)
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

const handleDelete = (id: number, serverId: number, username: string, host: string) => {
	databaseUserDeleteAPI({ id, serverId, username, host }).then((res: any) => {
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
  <n-alert
    type="info"
    class="mb-3"
  >
    {{ $t("database.databaseAccountAccessTips") }}
  </n-alert>
  <n-data-table
    striped
    remote
    :scroll-x="2200"
    :loading="loading"
    :columns="columns"
    :data="list"
    :row-key="rowIdentity"
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
