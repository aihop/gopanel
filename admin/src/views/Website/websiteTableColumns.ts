import { h } from "vue"
import { NButton, NSpace, NTag, type DataTableColumns } from "naive-ui"
import type { App } from "@/api/interface/apps"
import type { Website } from "@/api/interface/website"
import { formatTime } from "@/utils/date"
import { t } from "@/i18n"
import {
	getWebsiteSourceLabel,
	isHttpsWebsiteProtocol,
	normalizeWebsiteProtocol,
	resolveWebsiteBindingMeta
} from "@/utils/websiteRuntime"

export type WebsiteTableRow = Website.WebsiteDTO & {
	status?: string | boolean
}

interface WebsiteTableColumnOptions {
	appInstallMap: () => Record<number, App.AppInstalledInfo>
	httpServerRunning: () => boolean
	diagnosticText: (key: string, params?: Record<string, number>) => string
	onAccessLog: (row: WebsiteTableRow) => void
	onErrorLog: (row: WebsiteTableRow) => void
	onSecurity: (row: WebsiteTableRow) => void
	onDiagnostic: (row: WebsiteTableRow) => void
	onUpdate: (row: WebsiteTableRow) => void
	onDeploy: (row: WebsiteTableRow) => void
	onDelete: (row: WebsiteTableRow) => void
}

function normalizeDomainList(row: WebsiteTableRow) {
	if (Array.isArray(row.domains)) {
		return row.domains.filter(Boolean)
	}
	if (typeof row.otherDomains === "string") {
		return row.otherDomains.split(",").map(item => item.trim()).filter(Boolean)
	}
	return []
}

function websiteTypeLabel(type: string, text: WebsiteTableColumnOptions["diagnosticText"]) {
	if (type === "proxy") return text("websiteDiagnostic.tableReverseProxy")
	if (type === "web_app") return text("websiteDiagnostic.tableWebApp")
	return text("websiteDiagnostic.tableStatic")
}

function websiteTypeTag(type: string): "info" | "warning" | "success" {
	if (type === "proxy") return "info"
	if (type === "web_app") return "warning"
	return "success"
}

function websiteStatus(status: unknown, httpServerRunning: boolean, text: WebsiteTableColumnOptions["diagnosticText"]) {
	if (typeof status === "boolean") return status ? text("websiteDiagnostic.tableRunning") : text("websiteDiagnostic.tableStopped")
	if (typeof status === "string") {
		const normalized = status.trim().toLowerCase()
		if (["running", "enable", "enabled", "online", "success", "active"].includes(normalized)) return text("websiteDiagnostic.tableRunning")
		if (["stopped", "stop", "disabled", "disable", "offline", "inactive"].includes(normalized)) return text("websiteDiagnostic.tableStopped")
		return status
	}
	return httpServerRunning ? text("websiteDiagnostic.tableConnected") : text("websiteDiagnostic.tablePendingCheck")
}

function websiteStatusTag(status: unknown, httpServerRunning: boolean, text: WebsiteTableColumnOptions["diagnosticText"]): "success" | "warning" | "default" {
	const statusText = websiteStatus(status, httpServerRunning, text)
	if (statusText === text("websiteDiagnostic.tableRunning") || statusText === text("websiteDiagnostic.tableConnected")) return "success"
	if (statusText === text("websiteDiagnostic.tableStopped") || statusText === text("websiteDiagnostic.tablePendingCheck")) return "warning"
	return "default"
}

function securitySummary(row: WebsiteTableRow, text: WebsiteTableColumnOptions["diagnosticText"]) {
	const tags: string[] = []
	if (row.antiCrawler) tags.push(text("websiteDiagnostic.tableAntiCrawler"))
	if (row.antiLeech) tags.push(text("websiteDiagnostic.tableAntiLeech"))
	if (row.rateLimitMode === "normal") tags.push(text("websiteDiagnostic.tableNormalRateLimit"))
	if (row.rateLimitMode === "strict") tags.push(text("websiteDiagnostic.tableStrictRateLimit"))
	if (row.wafEnable) tags.push(text("websiteDiagnostic.tableWaf"))
	if (row.blockSensitive) tags.push(text("websiteDiagnostic.tableSensitiveProtection"))
	return tags
}

export function resolveWebsiteRowBindingMeta(
	row: WebsiteTableRow,
	appInstallMap: Record<number, App.AppInstalledInfo>,
	text: WebsiteTableColumnOptions["diagnosticText"]
) {
	return resolveWebsiteBindingMeta(
		row,
		{ appInstallMap },
		{
			includeSourceInDetail: false,
			kindFallback: text("websiteDiagnostic.runtimeFallback"),
			userFallback: text("websiteDiagnostic.imageUserFallback"),
			runtimePrefix: "",
			runUserPrefix: text("websiteDiagnostic.runUserPrefix")
		}
	)
}

function backendSummary(row: WebsiteTableRow) {
	if (row.type !== "proxy" && row.type !== "web_app") return ""
	const upstreams = Array.isArray(row.upstreams) ? row.upstreams.filter(item => item?.address) : []
	if (upstreams.length > 0) {
		const enabledCount = upstreams.filter(item => item.enabled !== false).length
		const preview = upstreams.slice(0, 2).map(item => item.address).join(", ")
		return `${t("website.backends")}: ${upstreams.length} · ${t("website.backendsEnabled", { count: enabledCount })} · ${preview}`
	}
	if (row.proxy) return `${t("website.backends")}: ${row.proxy}`
	return t("website.backendNotConfigured")
}

export function createWebsiteTableColumns(options: WebsiteTableColumnOptions): DataTableColumns<WebsiteTableRow> {
	return [
		{
			title: t("website.primaryDomain"), key: "primaryDomain",
			render(row) {
				const protocol = normalizeWebsiteProtocol(row.protocol) || "HTTP"
				return h("div", { class: "flex flex-col space-y-1" }, [
					h("a", {
						href: protocol === "HTTP" ? `http://${row.primaryDomain}` : `https://${row.primaryDomain}`,
						target: "_blank", class: "text-base font-semibold fg-base-100"
					}, row.primaryDomain),
					h("div", { class: "flex flex-wrap gap-2 pt-1" }, [
						h(NTag, { size: "small", round: true, bordered: false, type: isHttpsWebsiteProtocol(row) ? "success" : "default" }, { default: () => protocol }),
						row.defaultServer ? h(NTag, { size: "small", round: true, bordered: false, type: "warning" }, { default: () => options.diagnosticText("websiteDiagnostic.tableDefaultSite") }) : null
					].filter(Boolean))
				])
			}
		},
		{
			title: options.diagnosticText("websiteDiagnostic.tableSubdomains"), key: "otherDomains",
			render(row) {
				const domains = normalizeDomainList(row)
				return h("div", { class: "flex flex-wrap gap-2" }, domains.length
					? domains.map(item => h(NTag, { size: "small", round: true, bordered: false }, { default: () => item }))
					: [h("span", { class: "text-sm text-slate-400" }, options.diagnosticText("websiteDiagnostic.tableNoDomains"))])
			}
		},
		{
			title: options.diagnosticText("websiteDiagnostic.tableType"), key: "type",
			render(row) {
				const tags = [h(NTag, { round: true, bordered: false, type: websiteTypeTag(row.type) }, { default: () => websiteTypeLabel(row.type, options.diagnosticText) })]
				if (row.type === "web_app" && row.codeSource) {
					tags.push(h(NTag, { type: "default", size: "small", bordered: false, style: { marginLeft: "4px" } }, { default: () => getWebsiteSourceLabel(row.codeSource) }))
				}
				const bindingMeta = resolveWebsiteRowBindingMeta(row, options.appInstallMap(), options.diagnosticText)
				return h("div", { class: "flex flex-col gap-2" }, [
					h("div", { class: "flex items-center flex-wrap gap-1" }, tags),
					bindingMeta ? h("div", { class: "text-xs text-slate-500" }, `${bindingMeta.source} · ${bindingMeta.detail}`) : null,
					h("div", { class: "text-xs text-slate-500" }, backendSummary(row))
				])
			}
		},
		{
			title: options.diagnosticText("websiteDiagnostic.tableStatus"), key: "status",
			render(row) {
				return h(NTag, { round: true, bordered: false, type: websiteStatusTag(row.status, options.httpServerRunning(), options.diagnosticText) }, {
					default: () => websiteStatus(row.status, options.httpServerRunning(), options.diagnosticText)
				})
			}
		},
		{
			title: options.diagnosticText("websiteDiagnostic.tableSecurity"), key: "security",
			render(row) {
				const tags = securitySummary(row, options.diagnosticText)
				return h("div", { class: "flex flex-wrap gap-2" }, tags.length
					? tags.slice(0, 3).map(item => h(NTag, { size: "small", round: true, bordered: false, type: "success" }, { default: () => item }))
					: [h("span", { class: "text-sm text-slate-400" }, options.diagnosticText("websiteDiagnostic.tableNotEnabled"))])
			}
		},
		{
			title: options.diagnosticText("websiteDiagnostic.title"), key: "diagnostic",
			render(row) {
				const diagnostic = row.diagnostic
				if (!diagnostic?.configured) return h(NTag, { bordered: false }, { default: () => options.diagnosticText("websiteDiagnostic.unconfigured") })
				if (!diagnostic.enabled) return h(NTag, { bordered: false, type: "warning" }, { default: () => options.diagnosticText("websiteDiagnostic.disabled") })
				return h("div", { class: "flex flex-col gap-1" }, [
					h(NTag, { bordered: false, type: "success" }, {
						default: () => options.diagnosticText("websiteDiagnostic.monitoring", { sources: diagnostic.sourceCount, contents: diagnostic.contentCount })
					}),
					diagnostic.codeProjectId ? h("span", { class: "text-xs text-slate-500" }, [
						options.diagnosticText("websiteDiagnostic.linkedCode"),
						diagnostic.autoAnalysis ? ` · ${options.diagnosticText("websiteDiagnostic.autoEnabled")}` : ""
					]) : null,
					h("span", { class: "text-xs text-slate-500" }, options.diagnosticText("websiteDiagnostic.issueSummary", {
						open: diagnostic.openCount || 0, reopened: diagnostic.reopenedCount || 0, processing: diagnostic.processingCount || 0
					}))
				])
			}
		},
		{
			title: t("commons.table.createdAt"), key: "updatedAt",
			render: row => h("span", { class: "text-sm text-slate-500" }, formatTime(row.updatedAt))
		},
		{
			title: t("commons.table.operate"), key: "actions",
			render(row) {
				const buttons = [
					h(NButton, { text: true, type: "info", onClick: () => options.onAccessLog(row) }, { default: () => options.diagnosticText("websiteDiagnostic.actionAccessLog") }),
					h(NButton, { text: true, type: "error", onClick: () => options.onErrorLog(row) }, { default: () => options.diagnosticText("websiteDiagnostic.actionErrorLog") }),
					h(NButton, { text: true, type: "warning", onClick: () => options.onSecurity(row) }, { default: () => options.diagnosticText("websiteDiagnostic.tableSecurity") }),
					h(NButton, { text: true, type: "success", onClick: () => options.onDiagnostic(row) }, { default: () => options.diagnosticText("websiteDiagnostic.title") }),
					h(NButton, { text: true, type: "primary", onClick: () => options.onUpdate(row) }, { default: () => t("commons.button.set") })
				]
				if (row.type === "web_app" || row.type === "static") {
					buttons.splice(1, 0, h(NButton, { text: true, type: "success", onClick: () => options.onDeploy(row) }, { default: () => options.diagnosticText("websiteDiagnostic.actionDeploy") }))
				}
				buttons.push(h(NButton, { text: true, type: "error", onClick: () => options.onDelete(row) }, { default: () => options.diagnosticText("websiteDiagnostic.actionDelete") }))
				return h(NSpace, { size: 8 }, { default: () => buttons })
			}
		}
	]
}
