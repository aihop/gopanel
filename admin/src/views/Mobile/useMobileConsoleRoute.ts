import { ref, watch, type Ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import type { CodeSession, CodeSessionState } from "@/api/interface/code"

export type MobileConsoleTab = "overview" | "resources" | "code" | "settings"

interface MobileConsoleRouteOptions {
	isLocalNode: (nodeId: number) => boolean
	sessions: Ref<CodeSession[]>
	sessionState: Ref<CodeSessionState | null>
	loadSessions: (clearCurrent?: boolean) => Promise<void>
	loadSessionState: (silent?: boolean) => Promise<void>
}

export function useMobileConsoleRoute(options: MobileConsoleRouteOptions) {
	const route = useRoute()
	const router = useRouter()
	const routeTab = typeof route.query.tab === "string" ? route.query.tab : ""
	const activeTab = ref<MobileConsoleTab>(
		["overview", "resources", "code", "settings"].includes(routeTab) ? (routeTab as MobileConsoleTab) : "overview"
	)
	const selectedNodeId = ref(Number(route.query.nodeId || localStorage.getItem("gopanel-mobile-node-id") || 0))
	const selectedProjectId = ref<number | null>(
		Number(route.query.projectId || localStorage.getItem("gopanel-mobile-project-id")) || null
	)
	const selectedSessionId = ref(Number(route.query.sessionId) || 0)
	const selectedTaskId = ref(Number(route.query.taskId) || 0)
	const showSessionDetail = ref(Boolean(selectedSessionId.value) && route.query.view !== "list")

	function syncRoute(push = false) {
		const query = { ...route.query }
		query.tab = activeTab.value
		if (selectedNodeId.value) query.nodeId = String(selectedNodeId.value)
		else delete query.nodeId
		if (selectedProjectId.value) query.projectId = String(selectedProjectId.value)
		else delete query.projectId
		if (selectedSessionId.value && options.isLocalNode(selectedNodeId.value)) {
			query.sessionId = String(selectedSessionId.value)
			if (selectedTaskId.value) query.taskId = String(selectedTaskId.value)
			else delete query.taskId
			query.view = showSessionDetail.value ? "detail" : "list"
		} else {
			delete query.sessionId
			delete query.taskId
			delete query.view
		}
		void router[push ? "push" : "replace"]({ path: route.path, query })
	}

	watch(
		() => route.query,
		async query => {
			const tab = typeof query.tab === "string" ? query.tab : ""
			if (["overview", "resources", "code", "settings"].includes(tab)) {
				activeTab.value = tab as MobileConsoleTab
			}
			const nodeId = Number(query.nodeId) || 0
			const nodeChanged = nodeId !== selectedNodeId.value
			selectedNodeId.value = nodeId
			const projectId = Number(query.projectId) || selectedProjectId.value
			const projectChanged = projectId !== selectedProjectId.value
			selectedProjectId.value = projectId
			const sessionId = Number(query.sessionId) || 0
			const sessionChanged = sessionId !== selectedSessionId.value
			selectedSessionId.value = sessionId
			selectedTaskId.value = Number(query.taskId) || 0
			const detail = Boolean(sessionId) && query.view !== "list"
			const detailChanged = detail !== showSessionDetail.value
			showSessionDetail.value = detail
			if (!options.isLocalNode(nodeId)) {
				options.sessions.value = []
				options.sessionState.value = null
				return
			}
			if (activeTab.value !== "code") return
			if (nodeChanged || projectChanged) {
				await options.loadSessions(true)
			} else if ((sessionChanged || detailChanged) && selectedSessionId.value && showSessionDetail.value) {
				await options.loadSessionState()
			} else if (!selectedSessionId.value) options.sessionState.value = null
		},
		{ deep: true }
	)

	return {
		activeTab,
		selectedNodeId,
		selectedProjectId,
		selectedSessionId,
		selectedTaskId,
		showSessionDetail,
		syncRoute,
		router
	}
}
