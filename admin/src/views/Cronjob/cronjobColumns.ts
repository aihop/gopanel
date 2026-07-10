import { h } from "vue"
import { NButton, NSwitch, NTag, type DataTableColumns } from "naive-ui"

export const typeLabels: Record<string, string> = {
	shell: "执行 Shell 脚本",
	db_backup: "数据库备份",
	log_clean: "清理面板日志",
	ssl_renew: "SSL 证书续期"
}

const statusLabels: Record<string, { text: string; type: "success" | "error" | "warning" | "default" }> = {
	Running: { text: "执行中", type: "warning" },
	Success: { text: "成功", type: "success" },
	Failed: { text: "失败", type: "error" }
}

const weekLabels = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"]

// 把常见预设周期生成的 cron 表达式翻译成人话，非预设格式（比如手写的自定义表达式）就直接显示原文
export const describeSpec = (spec: string): string => {
	const parts = spec.trim().split(/\s+/)
	if (parts.length !== 5) return spec
	const [minute, hour, day, month, week] = parts
	const hm = (h: string, m: string) => `${h.padStart(2, "0")}:${m.padStart(2, "0")}`
	if (hour === "*" && day === "*" && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		return `每小时第 ${minute} 分钟`
	}
	if (/^\d+$/.test(hour) && day === "*" && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		return `每天 ${hm(hour, minute)}`
	}
	if (/^\d+$/.test(hour) && day === "*" && month === "*" && /^\d+$/.test(week) && /^\d+$/.test(minute)) {
		return `每${weekLabels[Number(week) % 7]} ${hm(hour, minute)}`
	}
	if (/^\d+$/.test(hour) && /^\d+$/.test(day) && month === "*" && week === "*" && /^\d+$/.test(minute)) {
		return `每月 ${day} 日 ${hm(hour, minute)}`
	}
	return spec
}

interface CronjobColumnOptions {
	openEdit: (row: any) => void
	openRecords: (row: any) => void
	handleRun: (row: any) => void
	handleDelete: (row: any) => void
	handleToggleStatus: (row: any, enabled: boolean) => void
}

export const createCronjobColumns = (options: CronjobColumnOptions): DataTableColumns<any> => [
	{ title: "名称", key: "name" },
	{
		title: "类型",
		key: "type",
		render: (row: any) => h(NTag, { size: "small", bordered: false }, { default: () => typeLabels[row.type] || row.type })
	},
	{
		title: "周期",
		key: "spec",
		render: (row: any) => describeSpec(row.spec)
	},
	{
		title: "状态",
		key: "status",
		render: (row: any) =>
			h(NSwitch, {
				value: row.status === "Enable",
				"onUpdate:value": (val: boolean) => options.handleToggleStatus(row, val)
			})
	},
	{
		title: "最近执行",
		key: "lastRecord",
		width: 220,
		render: (row: any) => {
			if (!row.lastRecord) return h("span", { class: "text-gray-400" }, "从未执行")
			const info = statusLabels[row.lastRecord.status] || { text: row.lastRecord.status, type: "default" as const }
			return h("div", { class: "flex items-center gap-2" }, [
				h("span", {}, row.lastRecord.startTime?.replace("T", " ").slice(0, 19) || "-"),
				h(NTag, { size: "small", type: info.type, bordered: false }, { default: () => info.text })
			])
		}
	},
	{
		title: "操作",
		key: "actions",
		align: "center" as const,
		fixed: "right",
		render: (row: any) =>
			h("div", { class: "flex items-center justify-center gap-2" }, [
				h(NButton, { text: true, type: "primary", onClick: () => options.openEdit(row) }, { default: () => "编辑" }),
				h(NButton, { text: true, type: "primary", onClick: () => options.handleRun(row) }, { default: () => "立即执行" }),
				h(NButton, { text: true, type: "primary", onClick: () => options.openRecords(row) }, { default: () => "执行记录" }),
				h(NButton, { text: true, type: "error", onClick: () => options.handleDelete(row) }, { default: () => "删除" })
			])
	}
]
