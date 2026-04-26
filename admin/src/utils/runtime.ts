export interface RuntimeLike {
	runtimeKind?: string
	runtimeMode?: string
	runUser?: string
}

export interface RuntimeLabelOptions {
	kindFallback?: string
	rootlessLabel?: string
	rootfulLabel?: string
	defaultModeLabel?: string
	userFallback?: string
	runtimePrefix?: string
	runUserPrefix?: string
}

const defaultOptions: Required<RuntimeLabelOptions> = {
	kindFallback: "Runtime",
	rootlessLabel: "rootless",
	rootfulLabel: "rootful",
	defaultModeLabel: "default",
	userFallback: "镜像默认",
	runtimePrefix: "运行时：",
	runUserPrefix: "用户："
}

function withDefaults(options?: RuntimeLabelOptions) {
	return { ...defaultOptions, ...(options || {}) }
}

export function getRuntimeKindLabel(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions) {
	const opts = withDefaults(options)
	const kind = String(item?.runtimeKind || "").toLowerCase()
	if (kind === "podman") return "Podman"
	if (kind === "docker") return "Docker"
	return opts.kindFallback
}

export function getRuntimeModeLabel(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions) {
	const opts = withDefaults(options)
	switch (String(item?.runtimeMode || "").toLowerCase()) {
		case "rootless":
			return opts.rootlessLabel
		case "rootful":
			return opts.rootfulLabel
		default:
			return opts.defaultModeLabel
	}
}

export function getRunUserLabel(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions) {
	const opts = withDefaults(options)
	return item?.runUser || opts.userFallback
}

export function buildRuntimeBadgeText(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions) {
	return `${getRuntimeKindLabel(item, options)}/${getRuntimeModeLabel(item, options)}`
}

export function buildRuntimeDetailText(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions & { prefix?: string }) {
	const opts = withDefaults(options)
	const parts = [
		options?.prefix || "",
		`${opts.runtimePrefix}${buildRuntimeBadgeText(item, opts)}`,
		`${opts.runUserPrefix}${getRunUserLabel(item, opts)}`
	].filter(Boolean)
	return parts.join(" · ")
}

export function buildRuntimeSummaryText(item: RuntimeLike | null | undefined, options?: RuntimeLabelOptions) {
	const opts = withDefaults(options)
	return `${buildRuntimeBadgeText(item, opts)} / ${opts.runUserPrefix}${getRunUserLabel(item, opts)}`
}
