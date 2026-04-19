import { useThemeStore } from "@/store/theme"

export function useThemeSwitch() {
	const themeStore = useThemeStore()

	return {
		toggle: () => {
			themeStore.toggleTheme()
		}
	}
}
