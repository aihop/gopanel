import type { ComputedRef, Ref } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { getAITasks } from "@/api/modules/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"

export function useCodeTaskPolling(
	projectId: ComputedRef<number>,
	tasks: Ref<CodeTaskListItem[]>,
	taskTotal: Ref<number>,
	onError: (error: unknown) => void
) {
	let requestPending = false
	let gitRefreshPending = false
	let pollCount = 0
	const fetchTasks = async (silent = false, includeGit = true) => {
		if (!projectId.value) return
		if (requestPending) {
			if (includeGit) gitRefreshPending = true
			return
		}
		const requestedProjectId = projectId.value
		requestPending = true
		try {
			const response = await getAITasks({ page: 1, limit: 50, projectId: requestedProjectId, includeGit })
			if (response.code !== 0) throw new Error(response.message)
			if (projectId.value !== requestedProjectId) return
			const previousTasks = new Map(tasks.value.map(task => [task.id, task]))
			tasks.value = (response.data.items || []).map(task => {
				if (includeGit) return task
				const previous = previousTasks.get(task.id)
				return previous ? { ...task, summary: { ...task.summary, ...pickCodeTaskGitSummary(previous) } } : task
			})
			taskTotal.value = response.data.total || 0
		} catch (error) {
			if (!silent) onError(error)
		} finally {
			requestPending = false
			if (gitRefreshPending) {
				gitRefreshPending = false
				void fetchTasks(true, true)
			}
		}
	}
	useIntervalFn(() => {
		pollCount++
		void fetchTasks(true, pollCount % 10 === 0)
	}, 3000)
	return { fetchTasks }
}

function pickCodeTaskGitSummary(task: CodeTaskListItem) {
	return {
		gitStatus: task.summary.gitStatus,
		gitError: task.summary.gitError,
		deliveryStatus: task.summary.deliveryStatus,
		deliveryStage: task.summary.deliveryStage,
		deliveryProgress: task.summary.deliveryProgress,
		deliveryQueuePosition: task.summary.deliveryQueuePosition,
		deliveryAttempt: task.summary.deliveryAttempt,
		deliveryError: task.summary.deliveryError,
		branch: task.summary.branch,
		additions: task.summary.additions,
		deletions: task.summary.deletions,
		changedFiles: task.summary.changedFiles,
		hasDiff: task.summary.hasDiff
	}
}
