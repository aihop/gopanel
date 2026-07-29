import type { NavigationGuardNext, RouteLocationNormalized, Router } from "vue-router"
import { useAuthStore } from "@/store/auth"
import GlobalStore from "@/store/modules/global"

export function checkAuth(router: Router) {
	router.beforeEach(authCheck)
}

// 注意：这里曾经在「进入面板」时自动触发 gp-agent 更新（AgentAutoUpdateAPI），
// 等于一运行就升级，用户无法选择时机。现已改为在
// 主机 - 工具箱 - 守护进程 页面手动点「更新 gp-agent」触发，见 useDaemonAgentStatus.updateAgent。

async function authCheck(
	to: RouteLocationNormalized,
	from: RouteLocationNormalized,
	next: NavigationGuardNext
): Promise<string | false | void> {
	const authStore = useAuthStore()
	const globalStore = GlobalStore()
	const routeEntrance = typeof to.query.entrance === "string" ? to.query.entrance.trim() : ""

	if (routeEntrance) {
		globalStore.setEntrance(routeEntrance)
	}

	// 手机控制台始终使用独立设备令牌，不能被桌面账号状态或菜单权限影响。
	if (to.path === "/mobile" || to.path.startsWith("/mobile/")) {
		return next()
	}

	// 第二步：检查是否已登录
	if (authStore.isLogged) {
		// 登录后再次访问「安全入口」路径本身：前端没有对应路由，会命中 NotFound 显示 404。
		// 此时直接进首页（入口的使命是放行进入面板，登录后不该再停在入口路径）。
		const entranceVal = (globalStore.entrance || "").replace(/^\/+/, "").replace(/\/+$/, "")
		if (entranceVal && to.name === "NotFound" && to.path.replace(/\/+$/, "") === "/" + entranceVal) {
			return next({ path: "/dashboard/index" })
		}
		// --- 新增：菜单权限拦截 ---
		// 检查路由对应的顶层模块 key，例如 /code/xxx 对应 code，/website/xxx 对应 website
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
