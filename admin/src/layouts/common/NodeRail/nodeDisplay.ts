import type { NodeItem, NodeWarning } from "@/api/modules/node"
import { t } from "@/i18n"

/** 节点状态级别。danger 优先于 warn，warn 优先于正常 */
export type NodeLevel = "online" | "warn" | "danger" | "offline" | "unknown"

export function nodeLevel(node: NodeItem): NodeLevel {
	if (node.status === "offline" || node.status === "unauthorized") return "offline"
	if (node.status === "unknown") return "unknown"
	if (node.warnings?.some(w => w.level === "danger")) return "danger"
	if (node.warnings?.length > 0) return "warn"
	return "online"
}

/** Tailwind 文本色类，用于节点状态圆点 */
export const levelDotClass: Record<NodeLevel, string> = {
	online: "text-green-500",
	warn: "text-amber-500",
	danger: "text-red-500",
	offline: "text-slate-400",
	unknown: "text-slate-300"
}

/** Naive UI tag 类型，用于抽屉里的状态标签 */
export const levelTagType: Record<NodeLevel, "success" | "warning" | "error" | "default"> = {
	online: "success",
	warn: "warning",
	danger: "error",
	offline: "default",
	unknown: "default"
}

/** 把后端返回的告警折算成展示文案。阈值判断在后端，这里只做 i18n */
export function warningText(warning: NodeWarning): string {
	switch (warning.type) {
		case "offline":
			return warning.value > 0
				? t("node.warning.offlineFor", { hours: Math.floor(warning.value) })
				: t("node.warning.offline")
		case "unauthorized":
			return t("node.warning.unauthorized")
		case "disk":
			return t("node.warning.disk", { percent: warning.value.toFixed(0) })
		case "cert":
			// 负数代表证书已经过期
			return warning.value < 0
				? t("node.warning.certExpired", { days: Math.abs(Math.floor(warning.value)) })
				: t("node.warning.certExpiring", { days: Math.floor(warning.value) })
		case "container":
			return t("node.warning.container", { count: Math.floor(warning.value) })
		default:
			return ""
	}
}

export function statusText(status: NodeItem["status"]): string {
	switch (status) {
		case "online":
			return t("node.status.online")
		case "offline":
			return t("node.status.offline")
		case "unauthorized":
			return t("node.status.unauthorized")
		default:
			return t("node.status.unknown")
	}
}

/** 水位条颜色：跟后端阈值保持一致（85 warn / 90 danger） */
export function usageColor(percent: number): string {
	if (percent >= 90) return "#ef4444"
	if (percent >= 85) return "#f59e0b"
	return "#18a058"
}

/**
 * 拼出直接打开该节点面板的地址。
 * 节点开了安全入口时必须走 /{entrance}——这条路径会命中节点的 authorizeEntrance
 * 并写入入口 cookie，否则会被挡在"请从安全入口登录"页面。
 */
export function nodePanelUrl(node: Pick<NodeItem, "addr" | "entrance">): string {
	const addr = (node.addr || "").replace(/\/+$/, "")
	const entrance = (node.entrance || "").trim().replace(/^\/+/, "")
	return entrance ? `${addr}/${entrance}` : addr
}

/** 在新标签打开节点自己的面板。noopener 避免被打开页拿到 window.opener */
export function openNodePanel(node: Pick<NodeItem, "addr" | "entrance">): void {
	window.open(nodePanelUrl(node), "_blank", "noopener,noreferrer")
}

export function formatBytes(bytes: number): string {
	if (!bytes || bytes <= 0) return "0"
	const units = ["B", "KB", "MB", "GB", "TB"]
	let value = bytes
	let index = 0
	while (value >= 1024 && index < units.length - 1) {
		value /= 1024
		index++
	}
	return `${value.toFixed(value >= 100 || index === 0 ? 0 : 1)}${units[index]}`
}
