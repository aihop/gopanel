import type { CodeSessionInitialization } from "../interface/code"

interface InitializationMessages {
	failed: string
	timedOut: string
}

export async function waitForCodeSessionInitialization(
	load: () => Promise<CodeSessionInitialization>,
	messages: InitializationMessages
) {
	const deadline = Date.now() + 300_000
	while (Date.now() < deadline) {
		await new Promise(resolve => window.setTimeout(resolve, 1000))
		const state = await load()
		if (state.status === "active") return
		if (state.status === "failed") throw new Error(state.initializationError || messages.failed)
	}
	throw new Error(messages.timedOut)
}
