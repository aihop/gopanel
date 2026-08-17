import { createPinia, setActivePinia } from "pinia"
import { beforeEach, describe, expect, it } from "vitest"
import { useCodeImmersiveStore } from "./codeImmersive"

describe("useCodeImmersiveStore", () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		localStorage.clear()
	})

	it("按用户恢复并保存沉浸模式", () => {
		localStorage.setItem("gopanel_code_immersive_12", "1")
		const store = useCodeImmersiveStore()

		store.restoreForUser(12)
		expect(store.isImmersive).toBe(true)

		store.exitImmersive()
		expect(store.isImmersive).toBe(false)
		expect(localStorage.getItem("gopanel_code_immersive_12")).toBe("0")
	})

	it("切换用户时隔离沉浸偏好", () => {
		localStorage.setItem("gopanel_code_immersive_1", "1")
		const store = useCodeImmersiveStore()

		store.restoreForUser(1)
		expect(store.isImmersive).toBe(true)

		store.restoreForUser(2)
		expect(store.isImmersive).toBe(false)
		store.toggleImmersive()
		expect(localStorage.getItem("gopanel_code_immersive_2")).toBe("1")
	})
})
