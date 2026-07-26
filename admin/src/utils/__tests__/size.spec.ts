import { describe, expect, it, vi } from "vitest"

// util.ts 顶层就 useClipboard()，而 vue-clipboard3 的 CJS 产物在 vitest 的 ESM
// 环境里加载不了（exports is not defined）。这里只测纯函数，把它 mock 掉即可。
vi.mock("vue-clipboard3", () => ({
	default: () => ({ toClipboard: async () => undefined })
}))

const { computeSize, computeSizeFromByte, computeSizeFromKB, computeSizeFromMB, splitSize } = await import("../util")

const KB = 1024
const MB = 1024 ** 2
const GB = 1024 ** 3
const TB = 1024 ** 4

describe("computeSizeFromByte", () => {
	it("各量级都按 1024 进制换算", () => {
		expect(computeSizeFromByte(512)).toBe("512 B")
		expect(computeSizeFromByte(2 * KB)).toBe("2.00 KB")
		expect(computeSizeFromByte(3.5 * MB)).toBe("3.50 MB")
		expect(computeSizeFromByte(3.5 * GB)).toBe("3.50 GB")
		expect(computeSizeFromByte(2 * TB)).toBe("2.00 TB")
	})

	// 磁盘管理页把 3.58 GB 的 git pack 显示成 3666.09 GB：
	// GB 分支除的是 1024**2 而不是 1024**3，整整放大了 1024 倍
	it("GB 不再放大 1024 倍", () => {
		const bytes = 3843819304 // ≈ 3.58 GiB，就是那个 pack 文件
		expect(computeSizeFromByte(bytes)).toBe("3.58 GB")
		expect(computeSizeFromByte(bytes)).not.toContain("3666")
	})

	it("TB 不再缩小 1024 倍", () => {
		expect(computeSizeFromByte(1.5 * TB)).toBe("1.50 TB")
	})

	it("单位边界处切换正确", () => {
		expect(computeSizeFromByte(KB - 1)).toContain("B")
		expect(computeSizeFromByte(KB)).toBe("1.00 KB")
		expect(computeSizeFromByte(MB)).toBe("1.00 MB")
		expect(computeSizeFromByte(GB)).toBe("1.00 GB")
		expect(computeSizeFromByte(TB)).toBe("1.00 TB")
	})

	// 和面板里用得最多的 computeSize（23 处）保持同一套换算，避免两个函数各说一套
	it("与 computeSize 的量级一致", () => {
		for (const bytes of [4 * KB, 7 * MB, 3.58 * GB, 2.5 * TB]) {
			expect(computeSizeFromByte(bytes).split(" ")[1]).toBe(computeSize(bytes).split(" ")[1])
			const a = Number(computeSizeFromByte(bytes).split(" ")[0])
			const b = Number(computeSize(bytes).split(" ")[0])
			expect(Math.abs(a - b)).toBeLessThan(0.01)
		}
	})
})

describe("computeSizeFromMB", () => {
	it("MB → GB → TB 换算正确", () => {
		expect(computeSizeFromMB(512)).toBe("512 MB")
		expect(computeSizeFromMB(2048)).toBe("2.00 GB")
		// 原来 TB 分支除的是 1024**3，2 TB 会显示成 0.00 TB
		expect(computeSizeFromMB(2 * 1024 * 1024)).toBe("2.00 TB")
	})
})

describe("已有的正确实现不受影响", () => {
	it("computeSizeFromKB", () => {
		expect(computeSizeFromKB(2048)).toBe("2.00 MB")
		expect(computeSizeFromKB(3 * MB)).toBe("3.00 GB")
		expect(computeSizeFromKB(2 * GB)).toBe("2.00 TB")
	})

	it("computeSize / splitSize", () => {
		expect(computeSize(3.58 * GB)).toBe("3.58 GB")
		expect(splitSize(3.58 * GB)).toEqual({ size: 3.58, unit: "GB" })
	})
})
