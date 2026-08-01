import { describe, expect, it, vi } from "vitest"
import { MobileTerminalOutputQueue, type MobileTerminalWriter } from "./mobileTerminalOutputQueue"

function createQueue() {
	const frames: FrameRequestCallback[] = []
	const writes: string[] = []
	const rendered: number[] = []
	const terminal: MobileTerminalWriter = {
		buffer: { active: { baseY: 20, viewportY: 20 } },
		write(data, callback) {
			writes.push(data)
			callback?.()
		},
		reset: vi.fn(),
		scrollToBottom: vi.fn(),
	}
	const queue = new MobileTerminalOutputQueue(terminal, {
		onRendered: sequence => rendered.push(sequence),
		onOverflow: vi.fn(),
		schedule: callback => {
			frames.push(callback)
			return frames.length
		},
		cancel: vi.fn(),
	})
	return { queue, terminal, frames, writes, rendered }
}

describe("MobileTerminalOutputQueue", () => {
	it("batches output and acknowledges only the latest rendered sequence", () => {
		const subject = createQueue()
		subject.queue.enqueue({ data: "one", sequence: 1 })
		subject.queue.enqueue({ data: "two", sequence: 2 })
		subject.frames.shift()?.(0)
		expect(subject.writes).toEqual(["onetwo"])
		expect(subject.rendered).toEqual([2])
		expect(subject.terminal.scrollToBottom).toHaveBeenCalledTimes(1)
	})

	it("preserves the viewport while the user reads older output", () => {
		const subject = createQueue()
		subject.terminal.buffer.active.viewportY = 10
		subject.queue.enqueue({ data: "new", sequence: 1 })
		subject.frames.shift()?.(0)
		expect(subject.terminal.scrollToBottom).not.toHaveBeenCalled()
	})

	it("resets before writing a replacement baseline", () => {
		const subject = createQueue()
		subject.queue.enqueue({ data: "baseline", sequence: 7, resetBefore: true, forceBottom: true })
		subject.frames.shift()?.(0)
		expect(subject.terminal.reset).toHaveBeenCalledTimes(1)
		expect(subject.rendered).toEqual([7])
	})

	it("does not split or drop surrogate pairs at a batch boundary", () => {
		const subject = createQueue()
		const data = `${"x".repeat(3)}😀tail`
		const queue = new MobileTerminalOutputQueue(subject.terminal, {
			onRendered: sequence => subject.rendered.push(sequence),
			onOverflow: vi.fn(),
			maxBatchChars: 4,
			schedule: callback => {
				subject.frames.push(callback)
				return subject.frames.length
			},
			cancel: vi.fn(),
		})
		queue.enqueue({ data, sequence: 8 })
		while (subject.frames.length) subject.frames.shift()?.(0)
		expect(subject.writes.join("")).toBe(data)
		expect(subject.rendered).toEqual([8])
	})

	it("drops excessive pending output and requests a resync", () => {
		const subject = createQueue()
		const onOverflow = vi.fn()
		const queue = new MobileTerminalOutputQueue(subject.terminal, {
			onRendered: vi.fn(),
			onOverflow,
			maxPendingChars: 4,
			schedule: callback => {
				subject.frames.push(callback)
				return subject.frames.length
			},
			cancel: vi.fn(),
		})
		expect(queue.enqueue({ data: "12345", sequence: 1 })).toBe(false)
		expect(onOverflow).toHaveBeenCalledTimes(1)
		expect(subject.writes).toEqual([])
	})
})
