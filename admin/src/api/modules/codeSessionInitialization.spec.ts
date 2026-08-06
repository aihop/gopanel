import { afterEach, describe, expect, it, vi } from "vitest"
import { waitForCodeSessionInitialization } from "./codeSessionInitialization"

afterEach(() => {
	vi.useRealTimers()
})

describe("waitForCodeSessionInitialization", () => {
	it("瞬时网络异常后继续轮询", async () => {
		vi.useFakeTimers()
		const load = vi.fn()
			.mockRejectedValueOnce(new Error("network unavailable"))
			.mockResolvedValueOnce({ id: 1, status: "active", currentStage: "idle" })
		const waiting = waitForCodeSessionInitialization(load, {
			failed: "initialization failed",
			timedOut: "initialization timed out"
		})

		await vi.advanceTimersByTimeAsync(2000)

		await expect(waiting).resolves.toBeUndefined()
		expect(load).toHaveBeenCalledTimes(2)
	})

	it("后端明确失败时仍立即报错", async () => {
		vi.useFakeTimers()
		const waiting = waitForCodeSessionInitialization(
			async () => ({ id: 1, status: "failed", currentStage: "initialization_failed", initializationError: "git failed" }),
			{ failed: "initialization failed", timedOut: "initialization timed out" }
		)

		const assertion = expect(waiting).rejects.toThrow("git failed")
		await vi.advanceTimersByTimeAsync(1000)
		await assertion
	})
})
