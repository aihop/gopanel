import type { Website } from "@/api/interface/website"
import { buildRuntimeDetailText, type RuntimeLabelOptions } from "@/utils/runtime"

type WebsiteBindingLike = Pick<
	Website.WebsiteDTO,
	"codeSource" | "appInstallId" | "pipelineId" | "appName" | "runtimeKind" | "runtimeMode" | "runUser"
> & Record<string, any>

interface WebsiteBindingMaps {
	appInstallMap?: Record<number, any>
	pipelineMap?: Record<number, any>
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
			return {
				source,
				name,
				detail: buildRuntimeDetailText(row, {
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
			return {
				source,
				name,
				detail: buildRuntimeDetailText(row, {
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
