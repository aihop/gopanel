import { useAuthStore } from "@/store/auth"
import { useCodeImmersiveStore } from "@/store/codeImmersive"
import { useEventListener, useMediaQuery } from "@vueuse/core"
import { computed, watch } from "vue"

export function useCodeImmersiveMode(enabled: () => boolean = () => true) {
	const authStore = useAuthStore()
	const codeImmersiveStore = useCodeImmersiveStore()
	const isDesktop = useMediaQuery("(min-width: 1000px)")
	const isImmersive = computed(() => enabled() && isDesktop.value && codeImmersiveStore.isImmersive)

	watch(
		() => authStore.user?.id ?? null,
		userId => codeImmersiveStore.restoreForUser(userId),
		{ immediate: true },
	)

	useEventListener(window, "keydown", event => {
		if (event.key === "Escape" && isImmersive.value) codeImmersiveStore.exitImmersive()
	})

	return {
		isDesktop,
		isImmersive,
		toggleImmersive: () => codeImmersiveStore.toggleImmersive(),
	}
}
