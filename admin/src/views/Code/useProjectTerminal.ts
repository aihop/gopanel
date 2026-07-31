import { computed, ref, type Ref } from "vue"
import { createCodeSession } from "@/api/modules/code"
import type { CodeSession } from "@/api/interface/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"

interface ProjectTerminalMessages {
	title: string
	created: string
	failed: string
}

export function useProjectTerminal(
	projectId: Readonly<Ref<number>>,
	tasks: Readonly<Ref<CodeTaskListItem[]>>,
	activeExecutor: Ref<string>,
	onActivateTask: (task: CodeTaskListItem) => void,
	onActivateSession: (session: CodeSession) => void,
	onSuccess: (message: string) => void,
	onError: (message: string) => void,
	messages: ProjectTerminalMessages
) {
	const opening = ref(false)
	const isTerminalSession = computed(() => activeExecutor.value === "terminal")

	const open = async () => {
		if (!projectId.value || opening.value) return
		const existing = tasks.value.find(task => task.agentName === "terminal" && task.sessionId > 0)
		if (existing) {
			onActivateTask(existing)
			return
		}
		opening.value = true
		try {
			const response = await createCodeSession({
				title: messages.title,
				workDir: "",
				projectId: projectId.value,
				executorId: "terminal",
				approvalPolicy: "full_auto",
				isolated: false,
				includeUncommitted: false
			})
			onActivateSession(response.data)
			onSuccess(messages.created)
		} catch (error) {
			onError(error instanceof Error ? error.message : messages.failed)
		} finally {
			opening.value = false
		}
	}

	return { opening, isTerminalSession, open }
}
