export interface TerminalCellPosition {
	column: number
	row: number
}

export interface TerminalTouchSelectionTarget {
	readonly cols: number
	readonly rows: number
	readonly buffer: { readonly active: { readonly viewportY: number } }
	select(column: number, row: number, length: number): void
}

export function terminalCellFromPoint(
	clientX: number,
	clientY: number,
	bounds: Pick<DOMRect, "left" | "top" | "width" | "height">,
	columns: number,
	rows: number,
	viewportY: number,
): TerminalCellPosition {
	const column = Math.floor(((clientX - bounds.left) / bounds.width) * columns)
	const viewportRow = Math.floor(((clientY - bounds.top) / bounds.height) * rows)
	return {
		column: Math.max(0, Math.min(columns - 1, column)),
		row: viewportY + Math.max(0, Math.min(rows - 1, viewportRow)),
	}
}

export function terminalSelectionRange(anchor: TerminalCellPosition, current: TerminalCellPosition, columns: number) {
	const anchorOffset = anchor.row * columns + anchor.column
	const currentOffset = current.row * columns + current.column
	const startOffset = Math.min(anchorOffset, currentOffset)
	return {
		column: startOffset % columns,
		row: Math.floor(startOffset / columns),
		length: Math.abs(currentOffset - anchorOffset) + 1,
	}
}

export class MobileTerminalTouchSelection {
	private timer: ReturnType<typeof setTimeout> | null = null
	private anchor: TerminalCellPosition | null = null
	private startX = 0
	private startY = 0
	private selecting = false

	constructor(
		private readonly terminal: TerminalTouchSelectionTarget,
		private readonly element: HTMLElement,
		private readonly longPressDelay = 450,
	) {
		element.addEventListener("touchstart", this.onTouchStart, { passive: true })
		element.addEventListener("touchmove", this.onTouchMove, { passive: false })
		element.addEventListener("touchend", this.onTouchEnd, { passive: false })
		element.addEventListener("touchcancel", this.onTouchEnd, { passive: false })
		element.addEventListener("contextmenu", this.preventContextMenu)
	}

	dispose() {
		this.cancelTimer()
		this.element.removeEventListener("touchstart", this.onTouchStart)
		this.element.removeEventListener("touchmove", this.onTouchMove)
		this.element.removeEventListener("touchend", this.onTouchEnd)
		this.element.removeEventListener("touchcancel", this.onTouchEnd)
		this.element.removeEventListener("contextmenu", this.preventContextMenu)
	}

	private readonly onTouchStart = (event: TouchEvent) => {
		if (event.touches.length !== 1) return
		const touch = event.touches[0]
		this.startX = touch.clientX
		this.startY = touch.clientY
		this.cancelTimer()
		this.timer = setTimeout(() => {
			this.anchor = this.cellAt(this.startX, this.startY)
			this.selecting = true
			this.terminal.select(this.anchor.column, this.anchor.row, 1)
		}, this.longPressDelay)
	}

	private readonly onTouchMove = (event: TouchEvent) => {
		if (event.touches.length !== 1) return
		const touch = event.touches[0]
		if (!this.selecting || !this.anchor) {
			if (Math.hypot(touch.clientX - this.startX, touch.clientY - this.startY) > 10) this.cancelTimer()
			return
		}
		event.preventDefault()
		const range = terminalSelectionRange(this.anchor, this.cellAt(touch.clientX, touch.clientY), this.terminal.cols)
		this.terminal.select(range.column, range.row, range.length)
	}

	private readonly onTouchEnd = (event: TouchEvent) => {
		this.cancelTimer()
		if (this.selecting) event.preventDefault()
		this.selecting = false
		this.anchor = null
	}

	private readonly preventContextMenu = (event: Event) => {
		if (window.matchMedia("(pointer: coarse)").matches) event.preventDefault()
	}

	private cellAt(clientX: number, clientY: number) {
		return terminalCellFromPoint(
			clientX,
			clientY,
			this.element.getBoundingClientRect(),
			this.terminal.cols,
			this.terminal.rows,
			this.terminal.buffer.active.viewportY,
		)
	}

	private cancelTimer() {
		if (this.timer) clearTimeout(this.timer)
		this.timer = null
	}
}
