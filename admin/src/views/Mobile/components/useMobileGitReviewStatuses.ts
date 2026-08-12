import { ref } from "vue"
import type { CodeGitScope, CodeGitStatus } from "@/api/interface/codeGit"

const scopes: CodeGitScope[] = ["workspace", "result"]

export function useMobileGitReviewStatuses(
	loadStatus: (sessionId: number, scope: CodeGitScope) => Promise<CodeGitStatus>,
	loadFailedMessage: () => string
) {
	const statuses = ref<Record<CodeGitScope, CodeGitStatus | null>>({ workspace: null, result: null })
	const loading = ref<Record<CodeGitScope, boolean>>({ workspace: false, result: false })
	const refreshing = ref<Record<CodeGitScope, boolean>>({ workspace: false, result: false })
	const errors = ref<Record<CodeGitScope, string>>({ workspace: "", result: "" })
	const sequences: Record<CodeGitScope, number> = { workspace: 0, result: 0 }

	const load = async (sessionId: number, scope: CodeGitScope, silent = false) => {
		if (!sessionId || loading.value[scope] || refreshing.value[scope]) return
		const sequence = ++sequences[scope]
		if (silent) refreshing.value[scope] = true
		else loading.value[scope] = true
		try {
			const response = await loadStatus(sessionId, scope)
			if (sequence !== sequences[scope]) return
			statuses.value[scope] = response
			errors.value[scope] = ""
		} catch (error) {
			if (sequence === sequences[scope]) {
				errors.value[scope] = error instanceof Error ? error.message : loadFailedMessage()
			}
		} finally {
			if (sequence === sequences[scope]) {
				loading.value[scope] = false
				refreshing.value[scope] = false
			}
		}
	}

	const invalidate = (scope: CodeGitScope) => {
		sequences[scope]++
		statuses.value[scope] = null
		loading.value[scope] = false
		refreshing.value[scope] = false
		errors.value[scope] = ""
	}
	const replace = (scope: CodeGitScope, status: CodeGitStatus) => {
		invalidate(scope)
		statuses.value[scope] = status
	}

	const reset = () => {
		for (const scope of scopes) invalidate(scope)
	}

	return { statuses, loading, refreshing, errors, load, invalidate, replace, reset }
}
