import type { CodeTaskListItem } from "@/api/interface/codeTasks"

export type CodeDashboardBucket = "attention" | "active" | "doneToday"

const ACTIVE_TASK_STATUSES = ["running", "queued", "delivering"]
const ACTIVE_DELIVERY_STATUSES = ["queued", "running"]
const FINISHED_TASK_STATUSES = ["completed", "failed", "cancelled"]
const BROKEN_DELIVERY_STATUSES = ["failed", "partial", "conflict"]

/** 任务用 updatedAt 排序；老数据没有这个字段时退回 createdAt。 */
export function codeTaskTimestamp(task: CodeTaskListItem) {
	return task.updatedAt || task.createdAt
}

export function isSameLocalDay(value: string, reference: Date) {
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) return false
	return (
		date.getFullYear() === reference.getFullYear() &&
		date.getMonth() === reference.getMonth() &&
		date.getDate() === reference.getDate()
	)
}

export function isDeliveringTask(task: CodeTaskListItem) {
	return task.status === "delivering" || ACTIVE_DELIVERY_STATUSES.includes(task.summary.deliveryStatus || "")
}

export function codeDashboardFocusStatus(task: CodeTaskListItem) {
	if (task.status === "pending_approval") return "pending_approval"
	if (isDeliveringTask(task)) return "delivering"
	return task.status
}

export function focusCodeDashboardTasks(tasks: CodeTaskListItem[]) {
	const priority = { pending_approval: 0, running: 1, delivering: 2, queued: 3 }
	return tasks
		.filter(task => ["pending_approval", "running", "queued", "delivering"].includes(codeDashboardFocusStatus(task)))
		.sort((left, right) => {
			const statusDifference = priority[codeDashboardFocusStatus(left) as keyof typeof priority]
				- priority[codeDashboardFocusStatus(right) as keyof typeof priority]
			return statusDifference || right.id - left.id
		})
}

/**
 * 一个任务只落一个桶，优先级：待我处理 > 运行中 > 今日完成。
 * 交付冲突/失败排在最前面，因为它卡住的是整条交付队列，不处理后面都跑不动。
 */
export function codeTaskBucket(task: CodeTaskListItem, now: Date): CodeDashboardBucket | null {
	if (task.status === "pending_approval") return "attention"
	if (BROKEN_DELIVERY_STATUSES.includes(task.summary.deliveryStatus || "")) return "attention"
	if (task.status === "failed" && isSameLocalDay(codeTaskTimestamp(task), now)) return "attention"
	if (ACTIVE_TASK_STATUSES.includes(task.status) || isDeliveringTask(task)) return "active"
	if (FINISHED_TASK_STATUSES.includes(task.status) && isSameLocalDay(codeTaskTimestamp(task), now)) {
		return "doneToday"
	}
	return null
}

export function groupCodeDashboardTasks(tasks: CodeTaskListItem[], now: Date) {
	const buckets: Record<CodeDashboardBucket, CodeTaskListItem[]> = { attention: [], active: [], doneToday: [] }
	for (const task of tasks) {
		const bucket = codeTaskBucket(task, now)
		if (bucket) buckets[bucket].push(task)
	}
	return {
		...buckets,
		deliveringCount: tasks.filter(isDeliveringTask).length,
	}
}

/**
 * 列表的显示顺序：按任务 id 倒序，新建的在上面。
 *
 * 刻意不按状态或 updated_at 排。任务状态每几秒就在变，
 * 一按状态排，你正要点的那一行会在指针底下跳走 —— 列表是用来点的，位置必须是稳的。
 * 后端那套「活跃任务优先」的排序只用来决定取哪 50 条（避免跑了几天的任务被挤出首页），
 * 取回来之后由这里重新排成固定顺序，两件事分开。
 */
export function sortCodeTasksStably(tasks: CodeTaskListItem[]) {
	return [...tasks].sort((left, right) => right.id - left.id)
}
