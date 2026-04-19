import type { NavigationGuardNext, RouteLocationNormalized, Router } from "vue-router"
import { useAuthStore } from "@/store/auth"

export function checkAuth(router: Router) {
	router.beforeEach(authCheck)
}

// 检查 entrance cookie 是否存在且有效
function checkEntranceCookie(): boolean {
	try {
		const entranceCookie = document.cookie
			.split("; ")
			.find(row => row.startsWith("Entrance="))
			?.split("=")[1]

		if (!entranceCookie) {
			return false
		}

		// 尝试解码 base64，如果成功说明 cookie 有效
		atob(entranceCookie)
		return true
	} catch (error) {
		console.warn("Invalid entrance cookie:", error)
		return false
	}
}

async function authCheck(
	to: RouteLocationNormalized,
	from: RouteLocationNormalized,
	next: NavigationGuardNext
): Promise<string | false | void> {
	const authStore = useAuthStore()

	// 第一步：检查 entrance cookie（优先级高）
	const hasValidEntranceCookie = checkEntranceCookie()

	// 如果没有有效的 entrance cookie
	if (!hasValidEntranceCookie) {
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
