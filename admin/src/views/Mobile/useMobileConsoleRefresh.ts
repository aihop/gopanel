import { onBeforeUnmount, type Ref } from "vue"
import type { MobileNode } from "@/api/modules/mobile"
import type { MobileConsoleTab } from "./useMobileConsoleRoute"

export function useMobileConsoleRefresh(options: {
	activeTab: Ref<MobileConsoleTab>
	selectedNode: Ref<MobileNode | null>
	selectedSessionId: Ref<number>
	loadOverview: (silent?: boolean) => Promise<void>
	loadNodes: (silent?: boolean) => Promise<void>
	loadSessionState: (silent?: boolean) => Promise<void>
}) {
	let nodeRefreshTicks = 0
	let refreshTimer: ReturnType<typeof setInterval> | null = null

	function startRefresh() {
		refreshTimer = setInterval(() => {
			if (options.activeTab.value === "overview") {
				if (options.selectedNode.value?.isLocal) void options.loadOverview(true)
				nodeRefreshTicks++
				if (nodeRefreshTicks >= 5) {
					nodeRefreshTicks = 0
					void options.loadNodes(true)
					if (!options.selectedNode.value?.isLocal) void options.loadOverview(true)
				}
			} else if (options.activeTab.value === "code" && options.selectedSessionId.value) {
				void options.loadSessionState(true)
			}
		}, 2000)
	}

	onBeforeUnmount(() => {
		if (refreshTimer) clearInterval(refreshTimer)
	})

	return startRefresh
}
