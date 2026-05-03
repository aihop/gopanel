import type { Website } from "@/api/interface/website"
import type { App } from "@/api/interface/apps"
import type { Pipeline } from "@/api/interface/pipeline"
import { buildRuntimeDetailText, type RuntimeLabelOptions } from "@/utils/runtime"

type WebsiteRuntimeFields = {
	runtimeKind?: string
	runtimeMode?: string
	runUser?: string
}

type WebsiteProtocolLike = {
	protocol?: string
}

type WebsiteIpv6Like = {
	IPV6?: boolean
	ipv6?: boolean
}

type WebsiteBindingLike = Pick<
	Website.WebsiteDTO,
	"codeSource" | "appInstallId" | "pipelineId" | "appName" | "runtimeKind" | "runtimeMode" | "runUser"
>

const websiteSourceLabelMap: Record<string, string> = {
	git: "自定义镜像",
	pipeline: "流水线",
	app_store: "应用商店",
	upload: "代码上传"
}

type AppInstallBindingLike = Pick<App.AppInstalledInfo, "name" | "runtimeKind" | "runtimeMode" | "runUser">
type PipelineBindingLike = Pick<Pipeline.ResPipeline, "name" | "runtimeKind" | "runtimeMode" | "runUser">

interface WebsiteBindingMaps {
	appInstallMap?: Record<number, AppInstallBindingLike>
	pipelineMap?: Record<number, PipelineBindingLike>
}

interface WebsiteBindingOptions extends RuntimeLabelOptions {
	sourcePrefix?: string
	includeSourceInDetail?: boolean
}

export function hasWebsiteRuntimeMeta(row: WebsiteBindingLike | null | undefined) {
	return !!(row?.runtimeKind || row?.runtimeMode || row?.runUser)
}

export function needsWebsiteBindingLookup(row: WebsiteBindingLike | null | undefined) {
	return !!row && !!(row.appInstallId || row.pipelineId) && !hasWebsiteRuntimeMeta(row)
}

export function isImageWebsiteSource(row: Pick<Website.WebsiteDTO, "codeSource"> | null | undefined) {
	return row?.codeSource === "git"
}

export function normalizeWebsiteProtocol(protocol?: string) {
	const value = String(protocol || "").trim().toUpperCase()
	if (value === "HTTP" || value === "HTTPS") {
		return value
	}
	return ""
}

export function isHttpsWebsiteProtocol(row: WebsiteProtocolLike | null | undefined) {
	return normalizeWebsiteProtocol(row?.protocol) === "HTTPS"
}

export function getWebsiteIpv6Value(row: WebsiteIpv6Like | null | undefined) {
	return !!(row?.IPV6 ?? row?.ipv6)
}

export function getWebsiteSourceLabel(codeSource?: string) {
	if (!codeSource) return ""
	return websiteSourceLabelMap[codeSource] || codeSource
}

export function resolveWebsiteBindingMeta(
	row: WebsiteBindingLike | null | undefined,
	maps: WebsiteBindingMaps,
	options?: WebsiteBindingOptions
) {
	if (!row) return null

	const includeSourceInDetail = options?.includeSourceInDetail ?? true
	const sourcePrefix = options?.sourcePrefix || ""

	if (row.codeSource === "app_store" && row.appInstallId) {
		const item = maps.appInstallMap?.[row.appInstallId]
		const name = row.appName || item?.name || `应用 #${row.appInstallId}`
		const source = "应用商店"
		const prefix = includeSourceInDetail ? `${sourcePrefix}${source} · ${name}` : name
		if (hasWebsiteRuntimeMeta(row)) {
			const runtimeRow: WebsiteRuntimeFields = row
			return {
				source,
				name,
				detail: buildRuntimeDetailText(runtimeRow, {
					prefix,
					kindFallback: "Runtime",
					userFallback: "镜像默认",
					...(options || {})
				})
			}
		}
		if (!item) {
			return {
				source,
				name,
				detail: includeSourceInDetail ? `${sourcePrefix}${source} · ${name}` : name
			}
		}
		return {
			source,
			name,
			detail: buildRuntimeDetailText(item, {
				prefix,
				kindFallback: "Runtime",
				userFallback: "镜像默认",
				...(options || {})
			})
		}
	}

	if (row.codeSource === "pipeline" && row.pipelineId) {
		const item = maps.pipelineMap?.[row.pipelineId]
		const name = item?.name || `流水线 #${row.pipelineId}`
		const source = "流水线"
		const prefix = includeSourceInDetail ? `${sourcePrefix}${source} · ${name}` : name
		if (hasWebsiteRuntimeMeta(row)) {
			const runtimeRow: WebsiteRuntimeFields = row
			return {
				source,
				name,
				detail: buildRuntimeDetailText(runtimeRow, {
					prefix,
					kindFallback: "Runtime",
					userFallback: "镜像默认",
					...(options || {})
				})
			}
		}
		if (!item) {
			return {
				source,
				name,
				detail: includeSourceInDetail ? `${sourcePrefix}${source} · ${name}` : name
			}
		}
		return {
			source,
			name,
			detail: buildRuntimeDetailText(item, {
				prefix,
				kindFallback: "Runtime",
				userFallback: "镜像默认",
				...(options || {})
			})
		}
	}

	return null
}
