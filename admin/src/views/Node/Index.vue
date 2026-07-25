<script setup lang="ts">
import type { NodeItem } from "@/api/modules/node"
import { nodeDeleteAPI, nodeProbeAPI } from "@/api/modules/node"
import Icon from "@/components/common/Icon.vue"
import CommonPage from "@/components/page/Common.vue"
import { t } from "@/i18n"
import { levelDotClass, nodeLevel, statusText, warningText } from "@/layouts/common/NodeRail/nodeDisplay"
import NodeStore from "@/store/modules/node"
import { useDialog, useMessage } from "naive-ui"
import { h, onMounted, ref } from "vue"
import LocalAccessCard from "./components/LocalAccessCard.vue"
import NodeFormModal from "./components/NodeFormModal.vue"

const message = useMessage()
const dialog = useDialog()
const nodeStore = NodeStore()

const modalVisible = ref(false)
const editingNode = ref<NodeItem | null>(null)
const probingId = ref(0)

const columns = [
	{
		title: () => t("node.column.name"),
		key: "name",
		render: (row: NodeItem) =>
			h("div", { class: "flex items-center gap-1" }, [
				h(Icon, { name: "mdi:circle", size: 9, class: levelDotClass[nodeLevel(row)] }),
				h("span", { class: "font-medium" }, row.name),
				row.isProd ? h("span", { class: "text-xs text-red-500" }, `· ${t("node.prod")}`) : null
			])
	},
	{ title: () => t("node.column.addr"), key: "addr" },
	{
		title: () => t("node.column.status"),
		key: "status",
		render: (row: NodeItem) =>
			h("div", {}, [
				h("div", {}, statusText(row.status)),
				row.statusMsg ? h("div", { class: "text-xs opacity-60 break-all" }, row.statusMsg) : null
			])
	},
	{
		title: () => t("node.column.warnings"),
		key: "warnings",
		render: (row: NodeItem) => {
			if (!row.warnings?.length) return h("span", { class: "opacity-40" }, "—")
			return h(
				"div",
				{ class: "flex flex-col gap-0.5" },
				row.warnings.map(warning =>
					h(
						"div",
						{ class: `text-xs ${warning.level === "danger" ? "text-red-500" : "text-amber-500"}` },
						warningText(warning)
					)
				)
			)
		}
	},
	{ title: () => t("node.column.version"), key: "version", render: (row: NodeItem) => row.version || "—" },
	{
		title: () => t("commons.table.operate"),
		key: "actions",
		render: (row: NodeItem) =>
			h("div", { class: "flex gap-2" }, [
				h(
					"a",
					{ class: "cursor-pointer text-primary", onClick: () => probe(row) },
					probingId.value === row.id ? t("node.probing") : t("node.probe")
				),
				h("a", { class: "cursor-pointer text-primary", onClick: () => openEdit(row) }, t("commons.button.edit")),
				h("a", { class: "cursor-pointer text-red-500", onClick: () => confirmDelete(row) }, t("commons.button.delete"))
			])
	}
]

async function probe(row: NodeItem) {
	probingId.value = row.id
	try {
		await nodeProbeAPI(row.id)
		message.success(t("node.probeDone"))
	} catch {
		// 失败提示由 axios 拦截器统一弹出（api/index.ts 对 FAIL 码已 MsgError + reject），
		// 这里再弹一次会让同一个错误出现两遍
	} finally {
		probingId.value = 0
		// 成败都要刷新：失败时状态列和告警要变成最新的失败原因
		await nodeStore.fetchList()
	}
}

function openCreate() {
	editingNode.value = null
	modalVisible.value = true
}

function openEdit(row: NodeItem) {
	editingNode.value = row
	modalVisible.value = true
}

function confirmDelete(row: NodeItem) {
	dialog.warning({
		title: t("commons.button.confirm"),
		content: t("node.deleteConfirm", { name: row.name }),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			try {
				await nodeDeleteAPI({ id: row.id })
				message.success(t("commons.msg.deleteSuccess"))
				await nodeStore.fetchList()
			} catch {
				// 错误提示由拦截器统一处理
			}
		}
	})
}

async function onSaved() {
	modalVisible.value = false
	await nodeStore.fetchList()
}

onMounted(() => {
	nodeStore.fetchList()
})
</script>

<template>
	<CommonPage>
		<LocalAccessCard class="mb-4" />

		<div class="mb-4 flex items-center justify-between gap-2">
			<div class="text-sm opacity-70">{{ t("node.listHint") }}</div>
			<div class="flex gap-2">
				<n-button :loading="nodeStore.loading" @click="nodeStore.refreshNow()">
					{{ t("node.refreshAll") }}
				</n-button>
				<n-button type="primary" @click="openCreate">{{ t("node.addNode") }}</n-button>
			</div>
		</div>

		<n-alert v-if="nodeStore.error" type="error" :show-icon="false" class="mb-4">
			{{ nodeStore.error }}
		</n-alert>

		<n-data-table
			striped
			:loading="nodeStore.loading"
			:columns="columns"
			:data="nodeStore.list"
			:row-key="(row: NodeItem) => row.id"
		/>

		<NodeFormModal v-model:show="modalVisible" :node="editingNode" @saved="onSaved" />
	</CommonPage>
</template>
