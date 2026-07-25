import type { NavigationGuardNext, RouteLocationNormalized, Router } from "vue-router"
import { useAuthStore } from "@/store/auth"
import GlobalStore from "@/store/modules/global"
import { AgentAutoUpdateAPI } from "@/api/modules/agent"

export function checkAuth(router: Router) {
	router.beforeEach(authCheck)
}

// 进入面板后触发一次 gp-agent 自动更新（每次页面加载一次；后端还有防重入+节流）。
// 刻意不在后端启动时触发——开发模式频繁重启会反复更新导致崩溃。
let agentAutoUpdateTriggered = false
function maybeTriggerAgentAutoUpdate() {
	if (agentAutoUpdateTriggered) return
	agentAutoUpdateTriggered = true
	AgentAutoUpdateAPI().catch(() => {})
}

function readEntranceCookie(): string {
	try {
		const entranceCookie = document.cookie
			.split("; ")
			.find(row => row.startsWith("Entrance="))
			?.split("=")[1]

		if (!entranceCookie) {
			return ""
		}

		return atob(entranceCookie)
	} catch (error) {
		console.warn("Invalid entrance cookie:", error)
		return ""
	}
}

async function authCheck(
	to: RouteLocationNormalized,
	from: RouteLocationNormalized,
	next: NavigationGuardNext
): Promise<string | false | void> {
	const authStore = useAuthStore()
	const globalStore = GlobalStore()
	const routeEntrance = typeof to.query.entrance === "string" ? to.query.entrance.trim() : ""
	const cookieEntrance = readEntranceCookie()

	if (routeEntrance) {
		globalStore.setEntrance(routeEntrance)
	} else if (cookieEntrance) {
		globalStore.setEntrance(cookieEntrance)
	}

	const hasValidEntrance = !!routeEntrance || !!cookieEntrance

	if (!hasValidEntrance) {
		// 清除可能存在的登录状态
		if (authStore.isLogged) {
			authStore.setLogout()
		}

		// 如果当前要访问的就是登录页面，允许访问
		if (to.path === "/login") {
			return next()
		}

		// 否则重定向到登录页面
		return next({ path: "/login" })
	}

	// 第二步：检查是否已登录
	if (authStore.isLogged) {
		// 登录后再次访问「安全入口」路径本身：前端没有对应路由，会命中 NotFound 显示 404。
		// 此时直接进首页（入口的使命是放行进入面板，登录后不该再停在入口路径）。
		const entranceVal = (globalStore.entrance || cookieEntrance || "").replace(/^\/+/, "").replace(/\/+$/, "")
		if (entranceVal && to.name === "NotFound" && to.path.replace(/\/+$/, "") === "/" + entranceVal) {
			return next({ path: "/dashboard/index" })
		}
		// 已登录 = 真正进入了面板，触发一次 gp-agent 自动更新（非阻塞）
		maybeTriggerAgentAutoUpdate()
		// --- 新增：菜单权限拦截 ---
		// 检查路由对应的顶层模块 key，例如 /ai/xxx 对应 ai，/website/xxx 对应 website
		const pathParts = to.path.split("/").filter(Boolean)
		if (pathParts.length > 0 && to.path !== "/profile" && to.path !== "/setting" && to.path !== "/dashboard/index") {
			const topModule = pathParts[0]
			const userMenus = authStore.userMenus || []
			const isSuperAdmin = authStore.role === "SUPER" || authStore.role === "ADMIN" || userMenus.includes("ALL")
			
			// 对于一些系统公共路径，直接放行
			const publicModules = ["login", "logout", "profile", "no-permission"]
			
			if (!isSuperAdmin && !publicModules.includes(topModule)) {
				// 首先检查具体的子路由名称是否有权限 (比如 Website-Index)
				const routeName = to.name as string
				const hasSubMenuPerm = routeName && userMenus.includes(routeName)
				
				// 如果当前访问的模块不在用户的 menus 权限列表中，且没有具体子路由的权限
				if (!userMenus.includes(topModule) && !hasSubMenuPerm) {
					return next({ path: "/no-permission" })
				}
			}
		}

		return next()
	}

	// 第三步：没有 auth 时，检查是否是登录或安装页面
	if (to.path === "/login") {
		return next()
	}

	return next({ path: "/login" })
}
