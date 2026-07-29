import type { NodeItem } from "@/api/modules/node"
import { nodeListAPI, nodeRefreshAPI } from "@/api/modules/node"
import piniaPersistConfig from "@/config/pinia-persist"
import { defineStore } from "pinia"

/** 列表自动刷新间隔。后端 cron 每分钟采集一轮，前端跟着这个节奏读库即可，不需要更密 */
const AUTO_REFRESH_MS = 60_000
let listRequest: Promise<void> | null = null

interface NodeState {
	list: NodeItem[]
	loading: boolean
	/** 首次加载是否已完成，用于区分"空列表"和"还没读过" */
	loaded: boolean
	error: string
	drawerVisible: boolean
	/** 从细条点进来时要高亮的节点，0 表示没有指定 */
	focusId: number
	/** 细条是否隐藏。默认常驻——细条上的状态灯就是这个功能的价值所在 */
	railHidden: boolean
}

const NodeStore = defineStore("NodeState", {
	state: (): NodeState => ({
		list: [],
		loading: false,
		loaded: false,
		error: "",
		drawerVisible: false,
		focusId: 0,
		railHidden: false
	}),
	getters: {
		/** 有节点才需要展示细条，一台都没配时不占用横向空间 */
		hasNodes(state): boolean {
			return state.list.length > 0
		},
		offlineCount(state): number {
			return state.list.filter(item => item.status === "offline" || item.status === "unauthorized").length
		},
		/** 存在 danger 级告警的节点数，用于细条上的角标 */
		dangerCount(state): number {
			return state.list.filter(item => item.warnings?.some(w => w.level === "danger")).length
		},
		warnCount(state): number {
			return state.list.filter(
				item => !item.warnings?.some(w => w.level === "danger") && item.warnings?.length > 0
			).length
		}
	},
	actions: {
		async fetchList() {
			if (listRequest) return listRequest

			const request = (async () => {
				this.loading = true
				this.error = ""
				try {
					const res = await nodeListAPI()
					this.list = res.data || []
				} catch (e: any) {
					this.error = e?.message || "加载节点列表失败"
				} finally {
					this.loading = false
					this.loaded = true
				}
			})()
			listRequest = request
			try {
				await request
			} finally {
				if (listRequest === request) listRequest = null
			}
		},
		/** 手动刷新：让后端立刻采集一轮再返回，比等 cron 更快看到结果 */
		async refreshNow() {
			this.loading = true
			this.error = ""
			try {
				const res = await nodeRefreshAPI()
				this.list = res.data || []
			} catch (e: any) {
				this.error = e?.message || "刷新节点状态失败"
			} finally {
				this.loading = false
				this.loaded = true
			}
		},
		openDrawer(focusId = 0) {
			this.focusId = focusId
			this.drawerVisible = true
		},
		closeDrawer() {
			this.drawerVisible = false
			this.focusId = 0
		},
		toggleDrawer() {
			if (this.drawerVisible) {
				this.closeDrawer()
			} else {
				this.openDrawer()
			}
		},
		setRailHidden(hidden: boolean) {
			this.railHidden = hidden
		},
		startAutoRefresh(): number {
			return window.setInterval(() => {
				// 抽屉打开时用户正在看，仍然刷新；只在请求进行中跳过，避免堆叠
				if (!this.loading) {
					this.fetchList()
				}
			}, AUTO_REFRESH_MS)
		}
	},
	// 只持久化用户的显示偏好。节点列表和 loading 是服务端状态，缓存到 localStorage
	// 会让刷新页面后先闪一遍过期数据（比如已经恢复的节点仍显示离线）
	persist: {
		...piniaPersistConfig("NodeState"),
		pick: ["railHidden"]
	}
})

export default NodeStore
