import { acceptHMRUpdate, defineStore } from "pinia"

const STORAGE_PREFIX = "gopanel_code_immersive"

function storageKey(userId: number | null) {
	return `${STORAGE_PREFIX}_${userId ?? "default"}`
}

export const useCodeImmersiveStore = defineStore("codeImmersive", {
	state: () => ({
		isImmersive: false,
		userId: undefined as number | null | undefined,
	}),
	actions: {
		restoreForUser(userId: number | null) {
			if (this.userId === userId) return
			this.userId = userId
			this.isImmersive = localStorage.getItem(storageKey(userId)) === "1"
		},
		setImmersive(value: boolean) {
			this.isImmersive = value
			localStorage.setItem(storageKey(this.userId), value ? "1" : "0")
		},
		toggleImmersive() {
			this.setImmersive(!this.isImmersive)
		},
		exitImmersive() {
			if (this.isImmersive) this.setImmersive(false)
		},
	},
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useCodeImmersiveStore, import.meta.hot))
}
