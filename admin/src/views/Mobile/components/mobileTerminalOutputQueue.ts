export interface MobileTerminalWriter {
	buffer: { active: { baseY: number; viewportY: number } }
	write: (data: string, callback?: () => void) => void
	reset: () => void
	scrollToBottom: () => void
}

export interface MobileTerminalOutput {
	data: string
	sequence?: number
	forceBottom?: boolean
	resetBefore?: boolean
}

interface MobileTerminalOutputQueueOptions {
	onRendered: (sequence: number) => void
	onOverflow: () => void
	schedule?: (callback: FrameRequestCallback) => number
	cancel?: (handle: number) => void
	maxBatchChars?: number
	maxPendingChars?: number
}

interface QueuedTerminalOutput extends MobileTerminalOutput {
	data: string
}

const terminalBatchChars = 16 * 1024
const terminalPendingChars = 2 * 1024 * 1024

export class MobileTerminalOutputQueue {
	private readonly items: QueuedTerminalOutput[] = []
	private readonly scheduleFrame: (callback: FrameRequestCallback) => number
	private readonly cancelFrame: (handle: number) => void
	private readonly maxBatchChars: number
	private readonly maxPendingChars: number
	private head = 0
	private frame: number | null = null
	private pendingChars = 0
	private writing = false
	private paused = false
	private disposed = false
	private generation = 0
	private resumeTimer: ReturnType<typeof setTimeout> | null = null
	private touchElement: HTMLElement | null = null

	constructor(
		private readonly terminal: MobileTerminalWriter,
		private readonly options: MobileTerminalOutputQueueOptions,
	) {
		this.scheduleFrame = options.schedule || (callback => window.requestAnimationFrame(callback))
		this.cancelFrame = options.cancel || (handle => window.cancelAnimationFrame(handle))
		this.maxBatchChars = options.maxBatchChars || terminalBatchChars
		this.maxPendingChars = options.maxPendingChars || terminalPendingChars
	}

	enqueue(output: MobileTerminalOutput) {
		if (this.disposed) return false
		if (this.pendingChars + output.data.length > this.maxPendingChars) {
			this.clear()
			this.options.onOverflow()
			return false
		}
		if (!output.data) {
			this.items.push({ ...output, data: "" })
		} else {
			for (let start = 0; start < output.data.length; ) {
				let end = Math.min(start + this.maxBatchChars, output.data.length)
				if (end < output.data.length && isHighSurrogate(output.data.charCodeAt(end - 1))) end--
				const isFirst = start === 0
				const isLast = end === output.data.length
				this.items.push({
					data: output.data.slice(start, end),
					sequence: isLast ? output.sequence : undefined,
					forceBottom: isLast && output.forceBottom,
					resetBefore: isFirst && output.resetBefore,
				})
				start = end
			}
		}
		this.pendingChars += output.data.length
		this.schedule()
		return true
	}

	clear() {
		this.generation++
		this.items.length = 0
		this.head = 0
		this.pendingChars = 0
		if (this.frame !== null) this.cancelFrame(this.frame)
		this.frame = null
	}

	bindTouchScrolling(element: HTMLElement) {
		this.unbindTouchScrolling()
		this.touchElement = element
		element.addEventListener("touchstart", this.pauseForScroll, { passive: true })
		element.addEventListener("touchend", this.resumeAfterScroll, { passive: true })
		element.addEventListener("touchcancel", this.resumeAfterScroll, { passive: true })
	}

	dispose() {
		this.disposed = true
		this.clear()
		this.unbindTouchScrolling()
	}

	private readonly pauseForScroll = () => {
		this.paused = true
		if (this.resumeTimer) clearTimeout(this.resumeTimer)
		this.resumeTimer = null
	}

	private readonly resumeAfterScroll = () => {
		if (this.resumeTimer) clearTimeout(this.resumeTimer)
		this.resumeTimer = setTimeout(() => {
			this.resumeTimer = null
			this.paused = false
			this.schedule()
		}, 120)
	}

	private unbindTouchScrolling() {
		if (this.resumeTimer) clearTimeout(this.resumeTimer)
		this.resumeTimer = null
		if (!this.touchElement) return
		this.touchElement.removeEventListener("touchstart", this.pauseForScroll)
		this.touchElement.removeEventListener("touchend", this.resumeAfterScroll)
		this.touchElement.removeEventListener("touchcancel", this.resumeAfterScroll)
		this.touchElement = null
	}

	private schedule() {
		if (this.disposed || this.paused || this.writing || this.frame !== null || !this.hasItems()) return
		this.frame = this.scheduleFrame(() => {
			this.frame = null
			this.flush()
		})
	}

	private flush() {
		if (this.disposed || this.paused || this.writing || !this.hasItems()) return
		const batch: QueuedTerminalOutput[] = []
		let batchChars = 0
		while (this.hasItems()) {
			const next = this.items[this.head]
			if (batch.length && (next.resetBefore || batchChars + next.data.length > this.maxBatchChars)) break
			batch.push(next)
			this.head++
			batchChars += next.data.length
			this.pendingChars -= next.data.length
			if (batchChars >= this.maxBatchChars) break
		}
		this.compactItems()
		const generation = this.generation
		const shouldFollow = batch.some(item => item.forceBottom) || this.isAtBottom()
		const resetBefore = batch.some(item => item.resetBefore)
		const sequence = batch.reduce((latest, item) => Math.max(latest, item.sequence || 0), 0)
		const data = batch.map(item => item.data).join("")
		if (resetBefore) this.terminal.reset()
		this.writing = true
		this.terminal.write(data, () => {
			this.writing = false
			if (generation === this.generation && !this.disposed) {
				if (shouldFollow) this.terminal.scrollToBottom()
				if (sequence > 0) this.options.onRendered(sequence)
			}
			this.schedule()
		})
	}

	private isAtBottom() {
		const buffer = this.terminal.buffer.active
		return buffer.baseY - buffer.viewportY <= 1
	}

	private hasItems() {
		return this.head < this.items.length
	}

	private compactItems() {
		if (this.head < 256 || this.head * 2 < this.items.length) return
		this.items.splice(0, this.head)
		this.head = 0
	}
}

function isHighSurrogate(code: number) {
	return code >= 0xd800 && code <= 0xdbff
}
