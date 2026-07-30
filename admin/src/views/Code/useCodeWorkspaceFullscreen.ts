import { computed, ref } from "vue"
import { useEventListener } from "@vueuse/core"

export function useCodeWorkspaceFullscreen(t: (key: string) => string) {
	const isWorkspaceFullscreen = ref(false)
	const fullscreenLabel = computed(() =>
		t(isWorkspaceFullscreen.value ? "code.exitWorkspaceFullscreen" : "code.enterWorkspaceFullscreen")
	)
	const toggleWorkspaceFullscreen = () => (isWorkspaceFullscreen.value = !isWorkspaceFullscreen.value)
	useEventListener(window, "keydown", event => {
		if (event.key === "Escape") isWorkspaceFullscreen.value = false
	})
	return { isWorkspaceFullscreen, fullscreenLabel, toggleWorkspaceFullscreen }
}
