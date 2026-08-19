export function insertTerminalSymbol(draft: string, symbol: string, start: number, end: number) {
	const selectionStart = Math.max(0, Math.min(start, draft.length))
	const selectionEnd = Math.max(selectionStart, Math.min(end, draft.length))
	return {
		value: `${draft.slice(0, selectionStart)}${symbol}${draft.slice(selectionEnd)}`,
		cursor: selectionStart + symbol.length,
	}
}

export class MobileTerminalInputFallback {
	private composing = false
	private compositionEnding = false
	private terminalData: string[] = []
	private timers = new Set<ReturnType<typeof setTimeout>>()
	private compositionTimer: ReturnType<typeof setTimeout> | null = null

	recordTerminalData(data: string) {
		this.terminalData.push(data)
		this.schedule(() => {
			const index = this.terminalData.indexOf(data)
			if (index >= 0) this.terminalData.splice(index, 1)
		}, 50)
	}

	startComposition() {
		if (this.compositionTimer) {
			clearTimeout(this.compositionTimer)
			this.timers.delete(this.compositionTimer)
			this.compositionTimer = null
		}
		this.composing = true
		this.compositionEnding = false
	}

	endComposition() {
		this.composing = false
		this.compositionEnding = true
		if (this.compositionTimer) {
			clearTimeout(this.compositionTimer)
			this.timers.delete(this.compositionTimer)
		}
		this.compositionTimer = this.schedule(() => {
			this.compositionEnding = false
			this.compositionTimer = null
		}, 0)
	}

	queueInput(
		event: Pick<InputEvent, "data" | "inputType" | "isComposing">,
		send: (data: string) => void,
	) {
		if (
			event.inputType !== "insertText" ||
			!event.data ||
			event.isComposing ||
			this.composing ||
			this.compositionEnding
		)
			return
		const data = event.data
		this.schedule(() => {
			const index = this.terminalData.indexOf(data)
			if (index >= 0) {
				this.terminalData.splice(index, 1)
				return
			}
			send(data)
		})
	}

	dispose() {
		for (const timer of this.timers) clearTimeout(timer)
		this.timers.clear()
		this.compositionTimer = null
		this.terminalData = []
	}

	private schedule(callback: () => void, delay = 0) {
		const timer = setTimeout(() => {
			this.timers.delete(timer)
			callback()
		}, delay)
		this.timers.add(timer)
		return timer
	}
}
