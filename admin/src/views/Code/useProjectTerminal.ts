import { ref, type Ref } from "vue"
import { openCodeProjectTerminal } from "@/api/modules/code"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"

interface ProjectTerminalMessages {
	created: string
	failed: string
	unavailable: string
}

export function useProjectTerminal(
	projectId: Readonly<Ref<number>>,
	sessionId: Readonly<Ref<number | null>>,
	onActivate: (session: HostTerminalSession) => void,
	onSuccess: (message: string) => void,
	onError: (message: string) => void,
	messages: ProjectTerminalMessages
) {
	const opening = ref(false)

	const open = async () => {
		if (opening.value) return
		if (!projectId.value) {
			onError(messages.unavailable)
			return
		}
		opening.value = true
		try {
			const response = await openCodeProjectTerminal(projectId.value, sessionId.value ?? undefined)
			onActivate(response.data)
			onSuccess(messages.created)
		} catch (error) {
			onError(error instanceof Error ? error.message : messages.failed)
		} finally {
			opening.value = false
		}
	}

	return { opening, open }
}
