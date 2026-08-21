<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage, NEmpty, NIcon, NButton, NDataTable, NInput, NAlert, NPopconfirm } from "naive-ui"
import { execDBManagerSqlAPI } from "@/api/modules/database"
import { renderIcon } from "@/utils"
import DatabaseWorkspaceHeader from "./DatabaseWorkspaceHeader.vue"

const props = defineProps<{
	selectedServerId: number | null
	selectedDatabase: string | null
	selectedTable: string | null
	serverOptions: any[]
}>()

const message = useMessage()
const { t } = useI18n()
const sqlQuery = ref("")
const executingSql = ref(false)
const sqlResult = ref<any>(null)
const sqlHistory = ref<Array<{ title: string; sql: string }>>([])

const historyStorageKey = computed(() => {
	return props.selectedServerId && props.selectedDatabase
		? `sql_history_${props.selectedServerId}_${props.selectedDatabase}`
		: null
})

const loadHistory = () => {
	const key = historyStorageKey.value
	if (!key) {
		sqlHistory.value = []
		return
	}
	try {
		const raw = localStorage.getItem(key)
		sqlHistory.value = raw ? JSON.parse(raw) : []
	} catch {
		sqlHistory.value = []
	}
}

const saveHistory = () => {
	const key = historyStorageKey.value
	if (!key) return
	try {
		localStorage.setItem(key, JSON.stringify(sqlHistory.value))
	} catch {
		/* quota exceeded - ignore */
	}
}

const clearHistory = () => {
	const key = historyStorageKey.value
	sqlHistory.value = []
	if (key) localStorage.removeItem(key)
}

const downloadFile = (content: string, filename: string) => {
	const blob = new Blob([content], { type: "text/plain;charset=utf-8" })
	const url = URL.createObjectURL(blob)
	const a = document.createElement("a")
	a.href = url
	a.download = filename
	a.click()
	URL.revokeObjectURL(url)
}

const exportSqlResultAsCSV = () => {
	if (!sqlResult.value || sqlResult.value.type !== "query") return
	const cols = sqlResult.value.columns || []
	const rows = sqlResult.value.rows || []
	const csvRows: string[] = []
	csvRows.push(cols.map((c: string) => `"${c.replace(/"/g, '""')}"`).join(","))
	for (const row of rows) {
		csvRows.push(
			cols
				.map((c: string) => {
					const v = row[c]
					if (v === null || v === undefined) return ""
					return `"${String(v).replace(/"/g, '""')}"`
				})
				.join(",")
		)
	}
	downloadFile("\ufeff" + csvRows.join("\n"), `sql_result_${Date.now()}.csv`)
}

onMounted(loadHistory)

const selectedServerLabel = computed(() => {
	return props.selectedServerId ? props.serverOptions.find(s => s.value === props.selectedServerId)?.label || "" : ""
})

const quickSqlTemplates = computed(() => {
	if (!props.selectedTable) return []

	const tableName = props.selectedTable
	return [
		{
			label: t("database.browseTop20"),
			sql: `SELECT * FROM \`${tableName}\` LIMIT 20;`
		},
		{
			label: t("database.countTotalRows"),
			sql: `SELECT COUNT(*) AS total_count FROM \`${tableName}\`;`
		},
		{
			label: t("database.orderByPkDesc"),
			sql: `SELECT * FROM \`${tableName}\` ORDER BY 1 DESC LIMIT 20;`
		},
		{
			label: t("database.insertTemplate"),
			sql: `INSERT INTO \`${tableName}\` ()\nVALUES ();`
		},
		{
			label: t("database.updateTemplate"),
			sql: `UPDATE \`${tableName}\`\nSET \nWHERE ;`
		},
		{
			label: t("database.deleteTemplate"),
			sql: `DELETE FROM \`${tableName}\`\nWHERE ;`
		}
	]
})

const sqlResultColumns = computed(() => {
	if (sqlResult.value && sqlResult.value.type === "query" && sqlResult.value.columns) {
		return sqlResult.value.columns.map((col: string) => ({
			title: col,
			key: col,
			ellipsis: { tooltip: true as const }
		}))
	}
	return []
})

const sqlResultSummary = computed(() => {
	if (!sqlResult.value) return []
	if (sqlResult.value.type === "query") {
		return [
			`${sqlResult.value.rows?.length || 0} ${t("database.rowsUnit")}`,
			`${sqlResult.value.columns?.length || 0} ${t("database.columnsUnit")}`,
			t("database.queryResult")
		]
	}
	return [`${sqlResult.value.affected || 0} ${t("database.rowsAffectedUnit")}`, t("database.execStatement")]
})

watch(
	() => [props.selectedServerId, props.selectedDatabase],
	() => {
		sqlResult.value = null
		loadHistory()
	},
	{ immediate: true }
)

const applyQuickSql = (sql: string) => {
	sqlQuery.value = sql
}

const pushSqlHistory = (sql: string) => {
	const normalized = sql.trim()
	if (!normalized) return
	const nextTitle = normalized.split("\n")[0].slice(0, 48)
	sqlHistory.value = [
		{ title: nextTitle, sql: normalized },
		...sqlHistory.value.filter(item => item.sql !== normalized)
	].slice(0, 6)
	saveHistory()
}

const executeSql = async () => {
	if (!props.selectedServerId || !props.selectedDatabase) {
		message.warning(t("database.selectServerAndDatabaseFirst"))
		return
	}
	if (!sqlQuery.value.trim()) return

	executingSql.value = true
	sqlResult.value = null

	try {
		const res = await execDBManagerSqlAPI({
			serverId: props.selectedServerId,
			databaseName: props.selectedDatabase,
			sql: sqlQuery.value
		})

		if (res.code === 0) {
			sqlResult.value = res.data
			pushSqlHistory(sqlQuery.value)
			if (res.data.type === "query") {
				message.success(t("database.querySuccess", { count: res.data.rows?.length || 0 }))
			} else {
				message.success(t("database.execSuccess", { count: res.data.affected || 0 }))
			}
		} else {
			message.error(res.msg || res.message || t("database.sqlExecuteFailed"))
		}
	} catch (error: any) {
		message.error(error?.message || t("database.sqlExecuteFailed"))
	} finally {
		executingSql.value = false
	}
}
</script>

<template>
	<div class="flex flex-1 flex-col overflow-hidden">
		<DatabaseWorkspaceHeader
			:server-label="selectedServerLabel"
			:database-name="selectedDatabase"
			:table-name="selectedTable"
			:title="selectedTable ? `${selectedTable} (SQL)` : t('database.sqlWorkspaceTitle')"
			icon="mdi:console"
		>
			<template #summary>
				<div class="hidden items-center gap-2 text-[11px] text-slate-500 xl:flex">
					<span class="rounded bg-slate-100 px-2 py-1">
						{{ t("database.quickTemplatesCount", { count: quickSqlTemplates.length }) }}
					</span>
					<span v-if="sqlResultSummary.length > 0" class="rounded bg-blue-50 px-2 py-1 text-blue-600">
						{{ sqlResultSummary.join(" · ") }}
					</span>
				</div>
			</template>
		</DatabaseWorkspaceHeader>
		<div
			v-if="quickSqlTemplates.length > 0"
			class="flex flex-wrap gap-2 border-b border-slate-200 bg-white px-2 py-2"
		>
			<n-button
				v-for="item in quickSqlTemplates"
				:key="item.label"
				size="tiny"
				quaternary
				@click="applyQuickSql(item.sql)"
			>
				{{ item.label }}
			</n-button>
		</div>
		<div class="relative flex h-48 flex-col border-b border-slate-200 bg-white p-2">
			<div class="mb-1 flex items-center gap-1 text-xs font-semibold text-slate-600">
				<n-icon :component="renderIcon('mdi:console')" />
				{{
					t("database.executeSqlIn", {
						database: selectedDatabase,
						tablePart: selectedTable ? ` / ${t("database.table")} ${selectedTable}` : ""
					})
				}}
			</div>
			<n-input
				v-model:value="sqlQuery"
				type="textarea"
				:placeholder="
					selectedTable
						? t('database.sqlInputPlaceholderWithTable', { table: selectedTable })
						: t('database.sqlInputPlaceholder')
				"
				class="flex-1 border border-slate-300 font-mono text-xs"
				@keydown.ctrl.enter="executeSql"
				@keydown.meta.enter="executeSql"
			/>
			<div class="mt-2 flex justify-end gap-2">
				<n-button size="small" @click="sqlQuery = ''">{{ t("database.clear") }}</n-button>
				<n-button
					type="primary"
					size="small"
					@click="executeSql"
					:loading="executingSql"
					:disabled="!sqlQuery.trim() || !selectedDatabase"
				>
					{{ t("database.executeWithShortcut") }}
				</n-button>
			</div>
		</div>
		<div class="flex min-h-0 flex-1 flex-col overflow-auto bg-[#f0f0f0] p-2">
			<div v-if="sqlHistory.length > 0" class="mb-2 border border-slate-200 bg-white">
				<div
					class="flex items-center justify-between border-b border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700"
				>
					<span>{{ t("database.recentExecutions") }}</span>
					<n-popconfirm @positive-click="clearHistory">
						<template #trigger>
							<n-button size="tiny" text type="error">{{ t("database.clear") }}</n-button>
						</template>
						{{ t("database.clearHistoryConfirm") }}
					</n-popconfirm>
				</div>
				<div class="flex flex-wrap gap-2 p-2">
					<n-button
						v-for="item in sqlHistory"
						:key="item.sql"
						size="tiny"
						quaternary
						@click="applyQuickSql(item.sql)"
					>
						{{ item.title }}
					</n-button>
				</div>
			</div>
			<n-empty v-if="!sqlResult" :description="t('database.resultWillShowHere')" class="mt-10" />
			<template v-else>
				<n-alert v-if="sqlResult.type === 'exec'" type="success" :title="t('database.execSuccessTitle')">
					{{ t("database.affectedRows") }}: {{ sqlResult.affected }}
				</n-alert>
				<div v-else class="flex min-h-0 flex-1 flex-col border border-slate-200 bg-white">
					<div class="flex justify-end gap-2 border-b border-slate-200 bg-[#f8f9fa] p-1.5">
						<n-button size="tiny" @click="exportSqlResultAsCSV">
							<template #icon><n-icon :component="renderIcon('mdi:file-delimited-outline')" /></template>
							{{ t("database.exportCsv") }}
						</n-button>
					</div>
					<n-data-table
						:columns="sqlResultColumns"
						:data="sqlResult.rows"
						:bordered="true"
						size="small"
						:scroll-x="sqlResultColumns.length * 120"
						class="min-h-0 flex-1 text-xs"
						flex-height
					/>
				</div>
			</template>
		</div>
	</div>
</template>
