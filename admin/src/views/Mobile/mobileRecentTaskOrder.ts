import type { CodeTaskListItem } from "@/api/interface/codeTasks"

const RUNNING_TASK_STATUSES = new Set(["running", "delivering", "executing"])
const ATTENTION_TASK_STATUSES = new Set(["pending_approval", "awaiting_approval"])
const QUEUED_TASK_STATUSES = new Set(["active", "queued", "interactive", "instruction_queued"])

export function mobileTaskDisplayStatus(task: CodeTaskListItem) {
	return task.status === "active" ? task.summary.stage || task.status : task.status
}

export function isMobileTaskActive(task: CodeTaskListItem) {
	return mobileTaskActivePriority(task) > 0
}

export function mobileTaskActivePriority(task: CodeTaskListItem) {
	const status = mobileTaskDisplayStatus(task)
	if (RUNNING_TASK_STATUSES.has(status) || task.summary.deliveryStatus === "running") return 3
	if (ATTENTION_TASK_STATUSES.has(status)) return 2
	if (QUEUED_TASK_STATUSES.has(status) || task.summary.deliveryStatus === "queued") return 1
	return 0
}

function activityTimestamp(task: CodeTaskListItem) {
	const value = task.summary.lastActivityAt || task.updatedAt || task.createdAt
	const timestamp = Date.parse(value)
	return Number.isNaN(timestamp) ? 0 : timestamp
}

export function sortMobileRecentTasks(tasks: CodeTaskListItem[]) {
	return [...tasks].sort((left, right) => {
		const activityOrder = mobileTaskActivePriority(right) - mobileTaskActivePriority(left)
		if (activityOrder !== 0) return activityOrder
		return activityTimestamp(right) - activityTimestamp(left) || right.id - left.id
	})
}
