import type { ComputedRef, Ref } from "vue"
import { useIntervalFn } from "@vueuse/core"
import { getAITasks } from "@/api/modules/code"
import type { AITask } from "@/api/interface/code"

export function useCodeTaskPolling(
	projectId: ComputedRef<number>,
	tasks: Ref<AITask[]>,
	onError: (error: unknown) => void
) {
	let requestPending = false
	const fetchTasks = async (silent = false) => {
		if (!projectId.value || requestPending) return
		const requestedProjectId = projectId.value
		requestPending = true
		try {
			const response = await getAITasks({ page: 1, limit: 50, projectId: requestedProjectId })
			if (response.code === 0 && projectId.value === requestedProjectId) tasks.value = response.data.items || []
		} catch (error) {
			if (!silent) onError(error)
		} finally {
			requestPending = false
		}
	}
	useIntervalFn(() => void fetchTasks(true), 3000)
	return { fetchTasks }
}
