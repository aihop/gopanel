const PROJECT_COLORS = [
	"#2563eb",
	"#7c3aed",
	"#db2777",
	"#dc2626",
	"#ea580c",
	"#ca8a04",
	"#16a34a",
	"#0d9488",
	"#0891b2",
	"#4f46e5"
] as const

export function codeProjectColor(projectId: number) {
	const normalizedId = Number.isFinite(projectId) ? Math.max(1, Math.trunc(Math.abs(projectId))) : 1
	return PROJECT_COLORS[(normalizedId - 1) % PROJECT_COLORS.length]
}
