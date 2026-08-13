import type { Backup } from "../api/interface/backup"
import { computeSize } from "../utils/util"
import { formatTime } from "../utils/date"
import { h } from "vue"
import { NButton, type DataTableColumns } from "naive-ui"

type Translate = (key: string) => string
type BackupAction = (row: Backup.RecordInfo) => void

export function createBackupPagination(page: number, pageSize: number, total: number) {
	return {
		page,
		pageSize,
		pageCount: Math.max(1, Math.ceil((total || 0) / pageSize)),
		showSizePicker: true,
		pageSizes: [10, 20, 50, 100],
		showQuickJumper: true,
		itemCount: total
	}
}

export function createBackupColumns(
	t: Translate,
	onDelete: BackupAction,
	onRecover: BackupAction,
	onDownload: BackupAction,
	onLoadSize: BackupAction
): DataTableColumns<any> {
	return [
		{ type: "selection" as const, width: 48 },
		{ title: t("commons.table.name"), key: "fileName", ellipsis: true },
		{
			title: t("file.size"),
			key: "size",
			width: 100,
			render(row: any) {
				if (row.hasLoad) return row.size ? computeSize(row.size) : "-"
				return h(
					NButton,
					{
						text: true,
						type: "primary",
						size: "tiny",
						loading: row.sizeLoading,
						onClick: () => onLoadSize(row)
					},
					{ default: () => t("file.calculate") }
				)
			}
		},
		{
			title: t("database.source"),
			key: "backupType",
			width: 150,
			render: (row: any) => (row.source ? t("setting." + row.source) : "")
		},
		{
			title: t("commons.table.date"),
			key: "createdAt",
			width: 180,
			render: (row: any) => formatTime(row.createdAt)
		},
		{
			title: t("commons.table.operate"),
			key: "actions",
			width: 240,
			fixed: "right",
			render(row: Backup.RecordInfo) {
				const action = (label: string, handler: BackupAction) =>
					h(NButton, { size: "small", onClick: () => handler(row) }, { default: () => t(label) })
				return h("div", { style: "display:flex; gap:8px;" }, [
					action("commons.button.delete", onDelete),
					action("commons.button.recover", onRecover),
					action("commons.button.download", onDownload)
				])
			}
		}
	]
}
