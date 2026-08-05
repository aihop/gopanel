import { onBeforeMount, ref } from "vue"

export function useNav(list: string[]) {
	const activeNav = ref("")
	const navInit = () => {
		if (!list || !list.length) return
		const ids = "#" + list.join(",#")
		const doms = document.querySelectorAll(ids)
	}
	onBeforeMount(() => {
		if (!list || !list.length) return
	})
	return {
		activeNav,
		navInit
	}
}
