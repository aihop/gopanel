<script setup lang="ts">
import { h, ref, watch } from "vue"
import { NModal, NDataTable, NButton, NTag, NSpace, NPopconfirm, useMessage, type DataTableColumns } from "naive-ui"
import { getPipelineRecords, runPipeline, stopPipeline, deletePipelineRecord, publishPipelineRelease } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import PipelineLogsModal from "./PipelineLogsModal.vue"
import { useAuthStore } from "@/store/auth"
import dayjs from "dayjs"
import { getRuntimeKindLabel, getRuntimeModeLabel, getRunUserLabel } from "@/utils/runtime"
import { renderChangelogCell } from "@/utils/changelog"
import { useI18n } from "vue-i18n"
import { t } from "@/i18n"
import { pipelineSourceMessages } from "./pipelineSourceMessages"
const props = defineProps<{ show: boolean; pipelineId: number }>()
const emit = defineEmits(["update:show"])

const message = useMessage()
const { t: sourceT } = useI18n({ messages: pipelineSourceMessages })
const authStore = useAuthStore()
const loading = ref(false)
const data = ref<Pipeline.ResRecord[]>([])
const isSubAdmin = authStore.user?.role === "SUB_ADMIN"

const handleCopy = async (text: string, successText: string) => {
	const value = String(text || "").trim()
	if (!value) {
		message.warning(t("pipeline.nothingToCopy"))
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
		message.error(t("pipeline.copyFailed"))
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
		message.success(t("pipeline.rerunTriggered", { version: currentRecordVersion.value }))

		if (res.data && res.data.recordId) {
			currentRecordId.value = res.data.recordId
		}

		fetchData()
	} catch (error: any) {
		void 0
	}
}

const handleRerun = async (row: Pipeline.ResRecord) => {
	try {
		const res = await runPipeline({ id: props.pipelineId, version: row.version })
		message.success(t("pipeline.rerunTriggered", { version: row.version }))

		if (res.data && res.data.recordId) {
			currentRecordId.value = res.data.recordId
			currentRecordVersion.value = row.version || ""
			logsModalShow.value = true
		}

		fetchData()
	} catch (error: any) {
		void 0
	}
}

const handleStop = async (row: Pipeline.ResRecord) => {
	try {
		await stopPipeline({ id: row.id })
		message.success(t("pipeline.stopSent"))
		fetchData()
	} catch (error: any) {
		void 0
	}
}

const handleDelete = async (row: Pipeline.ResRecord) => {
	try {
		await deletePipelineRecord({ id: row.id })
		message.success(t("pipeline.deleteSuccess"))
		if (data.value.length === 1 && pagination.value.page > 1) {
			pagination.value.page -= 1
		}
		fetchData()
	} catch (error: any) {
		void 0
	}
}

const handlePublishRelease = async (row: Pipeline.ResRecord) => {
	try {
		await publishPipelineRelease({ id: row.id })
		message.success(row.released ? t("pipeline.releaseExists") : t("pipeline.releaseGenerated"))
		fetchData()
	} catch (error: any) {
		void 0
	}
}

const getRecordResultTags = (row: Pipeline.ResRecord) => {
	const tags: Array<{ label: string; type: "success" | "warning" | "info" | "default" }> = []

	if (row.runnerHostPort) {
		tags.push({ label: t("pipeline.resultRunningInstance"), type: "success" })
	}
	if (row.archiveFile) {
		tags.push({ label: t("pipeline.resultArchive"), type: "info" })
	}
	if (row.imageTag) {
		tags.push({
			label: row.runnerHostPort ? t("pipeline.resultImageRef") : t("pipeline.resultScriptImage"),
			type: "warning"
		})
	}

	if (tags.length === 0) {
		tags.push({ label: t("pipeline.resultScript"), type: "default" })
	}

	return tags
}

const columns: DataTableColumns<Pipeline.ResRecord> = [
	{ title: "ID", key: "id", width: 60 },
	{
		title: t("commons.table.createdAt"),
		key: "createdAt",
		width: 150,
		ellipsis: { tooltip: true },
		render: (row: Pipeline.ResRecord) => (row.createdAt ? dayjs(row.createdAt).format("YYYY-MM-DD HH:mm") : "-")
	},
	{
		title: t("pipeline.version"),
		key: "version",
		width: 100,
		render: (row: Pipeline.ResRecord) => h(NTag, { type: "success", size: "small" }, { default: () => `v${row.version || "-"}` })
	},
	{
		title: sourceT("pipelineSource.recordSource"),
		key: "source",
		width: 150,
		ellipsis: { tooltip: true },
		render(row: Pipeline.ResRecord) {
			if (!row.codeProjectId) {
				return h(NTag, { size: "tiny", type: "default" }, { default: () => sourceT("pipelineSource.recordSourceGit") })
			}
			return h("div", { class: "flex flex-col gap-1" }, [
				h(NTag, { size: "tiny", type: "info" }, { default: () => sourceT("pipelineSource.recordSourceCode", { id: row.codeProjectId }) }),
				h(
					"span",
					{ class: "font-mono text-[10px] text-slate-400", title: row.sourceDigest || "" },
					row.sourceDigest ? row.sourceDigest.slice(0, 18) : "-"
				)
			])
		}
	},
	{
		title: "Commit",
		key: "commitHash",
		width: 140,
		ellipsis: { tooltip: true },
		render(row: Pipeline.ResRecord) {
			if (!row.commitHash) {
				return h("span", { class: "text-slate-400 text-xs" }, "-")
			}
			return h("div", { class: "flex items-center gap-1 text-xs min-w-0 overflow-hidden" }, [
				h("span", { class: "font-mono text-slate-700 truncate", title: row.commitHash }, row.commitHash.slice(0, 12)),
				h(
					"button",
					{
						type: "button",
						class: "shrink-0 text-[12px] leading-none text-blue-600 hover:text-blue-700 whitespace-nowrap",
						onClick: (event: MouseEvent) => {
							event.stopPropagation()
							void handleCopy(row.commitHash || "", t("pipeline.commitCopied"))
						}
					},
					t("commons.button.copy")
				)
			])
		}
	},
	{
		title: t("pipeline.changelog"),
		key: "changelog",
		minWidth: 220,
		render: (row: Pipeline.ResRecord) => renderChangelogCell(row.changelog)
	},
	{
		title: t("commons.table.status"),
		key: "status",
		width: 90,
		render(row: Pipeline.ResRecord) {
			let type: "default" | "info" | "success" | "warning" | "error" = "default"
			switch (row.status) {
				case "pending":
					type = "default"
					break
				case "preparing":
					type = "info"
					break
				case "cloning":
					type = "info"
					break
				case "building":
					type = "warning"
					break
				case "deploying":
					type = "info"
					break
				case "success":
					type = "success"
					break
				case "failed":
					type = "error"
					break
			}
			const statusLabel = row.status === "preparing" ? sourceT("pipelineSource.statusPreparing") : row.status
			return h("div", { class: "flex flex-col items-center py-0.5" }, [
				h(NTag, { type, size: "tiny" }, { default: () => statusLabel }),
				h(
					"span",
					{ class: "text-[11px] leading-4 mt-0.5", style: { color: row.released ? "#22c55e" : "#94a3b8" } },
					row.released ? t("pipeline.published") : t("pipeline.unpublished")
				)
			])
		}
	},
	{
		title: t("pipeline.resultType"),
		key: "resultType",
		width: 150,
		ellipsis: { tooltip: true },
		render(row: Pipeline.ResRecord) {
			const tags = getRecordResultTags(row)
			return h(
				"div",
				{ class: "flex gap-1 overflow-hidden" },
				tags.slice(0, 2).map(item => h(NTag, { size: "tiny", type: item.type }, { default: () => item.label }))
			)
		}
	},
	{
		title: t("pipeline.runtime"),
		key: "runtime",
		width: 260,
		ellipsis: { tooltip: true },
		render(row: Pipeline.ResRecord) {
			const children: any[] = [
				h("div", { class: "flex flex-wrap items-center gap-1" }, [
					h(
						NTag,
						{ size: "tiny", type: row.runtimeKind === "docker" ? "success" : "warning" },
						{
							default: () => getRuntimeKindLabel(row, { kindFallback: "?" })
						}
					),
					h(
						NTag,
						{ size: "tiny", type: row.runtimeMode === "rootless" ? "warning" : "default" },
						{
							default: () => getRuntimeModeLabel(row)
						}
					),
					h("span", { class: "text-[11px] text-slate-400" }, getRunUserLabel(row))
				])
			]
			if (row.runnerHostPort) {
				children.push(
					h("div", { class: "flex items-center gap-1 mt-0.5 truncate" }, [
						h("span", { class: "text-[11px] text-slate-400" }, t("pipeline.preview") + ":"),
						h("span", { class: "font-mono text-emerald-600 text-xs truncate" }, `127.0.0.1:${row.runnerHostPort}`)
					])
				)
			}
			return h("div", { class: "flex flex-col py-0.5 overflow-hidden" }, children)
		}
	},
	{ title: t("pipeline.errorMessage"), key: "errorMessage", ellipsis: true },
	{
		title: t("pipeline.actions"),
		key: "actions",
		width: 160,
		fixed: "right",
		render(row: Pipeline.ResRecord, index: number) {
			const isFirstRow = index === 0 && pagination.value.page === 1
			const isRunning = ["pending", "preparing", "cloning", "building", "deploying"].includes(row.status)

			const btns = [
				h(
					NButton,
					{
						size: "small",
						type: "primary",
						ghost: true,
						onClick: () => {
							currentRecordId.value = row.id
							currentRecordVersion.value = row.version || ""
							logsModalShow.value = true
						}
					},
					{ default: () => t("pipeline.logs") }
				)
			]

			if (isFirstRow) {
				if (isRunning) {
					btns.push(
						h(
							NPopconfirm,
							{
								onPositiveClick: () => handleStop(row),
								positiveText: t("pipeline.confirmStop")
							},
							{
								trigger: () =>
									h(
										NButton,
										{
											size: "small",
											type: "error",
											ghost: true
										},
										{ default: () => t("pipeline.forceStop") }
									),
								default: () => t("pipeline.forceStopConfirm")
							}
						)
					)
				} else {
					btns.push(
						h(
							NButton,
							{
								size: "small",
								type: "warning",
								ghost: true,
								onClick: () => handleRerun(row)
							},
							{ default: () => t("pipeline.rerun") }
						)
					)
				}
			}

			if (!isSubAdmin && row.status === "success") {
				btns.push(
					h(
						NButton,
						{
							size: "small",
							type: row.released ? "default" : "success",
							ghost: !row.released,
							disabled: !!row.released,
							onClick: () => handlePublishRelease(row)
						},
						{ default: () => (row.released ? t("pipeline.generated") : t("pipeline.generateRelease")) }
					)
				)
			}

			if (!isSubAdmin && !isRunning) {
				btns.push(
					h(
						NPopconfirm,
						{
							onPositiveClick: () => handleDelete(row),
							positiveText: t("pipeline.confirmDelete")
						},
						{
							trigger: () =>
								h(
									NButton,
									{
										size: "small",
										type: "error",
										ghost: true
									},
									{ default: () => t("pipeline.delete") }
								),
							default: () => t("pipeline.deleteRecordConfirm")
						}
					)
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
		void 0
	} finally {
		loading.value = false
	}
}

watch(
	() => props.show,
	newVal => {
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
		:title="t('pipeline.executionRecords')"
		style="width: 1080px"
		class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(15,23,42,0.12)] sm:w-[90%]"
		@update:show="val => emit('update:show', val)"
	>
		<div class="mb-4 text-sm text-slate-500">
			{{ t("pipeline.recordsHelper") }}
		</div>
		<div class="h-[500px] overflow-auto">
			<n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination" :bordered="false" />
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
