export type CodeProjectDropPosition = "before" | "after"

export function reconcileCodeProjectOrder(projectIds: number[], savedOrder: number[]) {
	const currentIds = new Set(projectIds)
	const savedIds = new Set(savedOrder)
	const newIds = projectIds.filter(id => !savedIds.has(id))
	const existingIds = savedOrder.filter(id => currentIds.has(id))
	return [...newIds, ...existingIds]
}

export function sortCodeProjectsByOrder<T extends { id: number }>(projects: T[], order: number[]) {
	const positions = new Map(order.map((id, index) => [id, index]))
	return [...projects].sort((left, right) => {
		const leftPosition = positions.get(left.id) ?? Number.MAX_SAFE_INTEGER
		const rightPosition = positions.get(right.id) ?? Number.MAX_SAFE_INTEGER
		return leftPosition - rightPosition
	})
}

export function moveCodeProject(
	order: number[],
	projectId: number,
	targetProjectId: number,
	position: CodeProjectDropPosition,
) {
	if (projectId === targetProjectId || !order.includes(projectId) || !order.includes(targetProjectId)) return order
	const next = order.filter(id => id !== projectId)
	const targetIndex = next.indexOf(targetProjectId)
	next.splice(targetIndex + (position === "after" ? 1 : 0), 0, projectId)
	return next
}

export function loadCodeProjectOrder(storageKey: string) {
	try {
		const value = JSON.parse(localStorage.getItem(storageKey) || "[]")
		return Array.isArray(value) ? value.filter(id => Number.isInteger(id) && id > 0) : []
	} catch {
		return []
	}
}

export function saveCodeProjectOrder(storageKey: string, order: number[]) {
	try {
		localStorage.setItem(storageKey, JSON.stringify(order))
	} catch {
		void 0
	}
}

export function codeProjectOrderStorageKey(userId?: number) {
	return `code-dashboard-project-order:${userId || "current"}`
}
