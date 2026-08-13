import type { FormType } from "@/components/auth/types.d"
import { Layout } from "@/types/theme.d"
import { checkAuth } from "./guard/auth"
import { createRouter, createWebHistory } from "vue-router"
import { t } from "@/i18n"

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL != "" ? import.meta.env.BASE_URL : "/"),
	routes: [
		{
			path: "/",
			redirect: "/dashboard/index"
		},
		{
			path: "/profile",
			name: "Profile",
			component: () => import("@/views/Profile.vue"),
			meta: { title: t("user.profile"), titleKey: "user.profile", auth: true, roles: "all" }
		},
		{
			path: "/dashboard",
			name: "Dashboard",
			redirect: "/dashboard/index",
			meta: {
				title: t("menu.dashboard"),
				titleKey: "menu.dashboard",
				auth: true,
				roles: "all"
			},
			children: [
				{
					path: "index",
					name: "Dashboard-Index",
					component: () => import("@/views/Dashboard/Index.vue"),
					meta: { title: t("menu.dashboardHelper"), titleKey: "menu.dashboardHelper", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/website",
			name: "Website",
			redirect: "/website/index",
			meta: { title: t("menu.website"), titleKey: "menu.website", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Website-Index",
					component: () => import("@/views/Website/Index.vue"),
					meta: { title: t("menu.website"), titleKey: "menu.website", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/ssl",
			name: "SSL",
			redirect: "/ssl/index",
			meta: { title: t("menu.ssl"), titleKey: "menu.ssl", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "SSL-Index",
					component: () => import("@/views/Website/SSL.vue"),
					meta: { title: t("ssl.sslManagement"), titleKey: "ssl.sslManagement", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/code",
			name: "Code",
			redirect: "/code/index",
			meta: { title: t("menu.code"), titleKey: "menu.code", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Code-Index",
					component: () => import("@/views/Code/Index.vue"),
					meta: { title: t("code.workspace"), titleKey: "code.workspace", auth: true, roles: "all" }
				},
				{
					path: "project/:id",
					name: "Code-Project",
					component: () => import("@/views/Code/Workspace.vue"),
					meta: { title: t("code.projectWorkspace"), titleKey: "code.projectWorkspace", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/pipeline",
			name: "Pipeline",
			redirect: "/pipeline/index",
			meta: { title: "流水线", titleKey: "menu.pipeline", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Pipeline-Index",
					component: () => import("@/views/Pipeline/Index.vue"),
					meta: { title: "CI/CD 流水线", titleKey: "pipeline.workspace", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/flow",
			name: "Flow",
			redirect: "/flow/index",
			meta: { title: t("menu.flow"), titleKey: "menu.flow", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Flow-Index",
					component: () => import("@/views/Flow/Index.vue"),
					meta: { title: t("menu.flow"), titleKey: "menu.flow", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/database",
			name: "Database",
			redirect: "/database/index",
			meta: { title: "数据库", titleKey: "menu.database", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Database-Index",
					component: () => import("@/views/Database/Index.vue"),
					meta: { title: "数据库列表", titleKey: "database.listPage", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/cronjob",
			redirect: "/host/cronjob"
		},
		{
			path: "/cronjob/index",
			redirect: "/host/cronjob"
		},
		{
			path: "/node",
			name: "Node",
			redirect: "/node/index",
			meta: { title: t("menu.node"), titleKey: "menu.node", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Node-Index",
					component: () => import("@/views/Node/Index.vue"),
					meta: { title: t("menu.node"), titleKey: "menu.node", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/apps",
			name: "Apps",
			redirect: "/apps/index",
			meta: { title: "网站", titleKey: "menu.website", auth: true, roles: "all" },
			children: [
				{
					path: "index",
					name: "Apps-Index",
					component: () => import("@/views/Apps/index.vue"),
					meta: { title: t("website.websiteList"), titleKey: "website.websiteList", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/container",
			name: "ContainerIndex",
			redirect: "/container/index",
			meta: {
				title: t("menu.container"),
				auth: true,
				roles: "all"
			},
			component: () => import("@/views/Container/index.vue"),
			children: [
				{
					path: "index",
					name: "Container-Index",
					component: () => import("@/views/Container/container/index.vue"),
					meta: { title: "Docker", auth: true, roles: "all" }
				},
				{
					path: "compose",
					name: "Container-Compose",
					component: () => import("@/views/Container/compose/index.vue"),
					meta: { title: "Docker Compose", auth: true, roles: "all" }
				},
				{
					path: "image",
					name: "Container-Image",
					component: () => import("@/views/Container/image/index.vue"),
					meta: { title: "Docker Image", auth: true, roles: "all" }
				},
				{
					path: "network",
					name: "Container-Network",
					component: () => import("@/views/Container/network/index.vue"),
					meta: { title: "Docker Network", auth: true, roles: "all" }
				},
				{
					path: "volume",
					name: "Container-Volume",
					component: () => import("@/views/Container/volume/index.vue"),
					meta: { title: "Docker Volume", auth: true, roles: "all" }
				},
				{
					path: "repo",
					name: "Container-repo",
					component: () => import("@/views/Container/repo/index.vue"),
					meta: { title: "Docker repo", auth: true, roles: "all" }
				},
				{
					path: "template",
					name: "Container-Template",
					component: () => import("@/views/Container/template/index.vue"),
					meta: { title: "Docker Template", auth: true, roles: "all" }
				},
				{
					path: "setting",
					name: "Container-Setting",
					component: () => import("@/views/Container/setting/index.vue"),
					meta: { title: "Docker Setting", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/host",
			name: "Host",
			redirect: "/host/process",
			meta: {
				title: "主机",
				auth: true,
				roles: "all"
			},
			children: [
				{
					path: "terminal",
					name: "Host-Terminal",
					component: () => import("@/views/Host/terminal/index.vue"),
					meta: { title: t("menu.terminal"), titleKey: "menu.terminal", auth: true, roles: "all", keepAlive: true }
				},
				{
					path: "monitor",
					name: "Host-Monitor",
					component: () => import("@/views/Host/monitor.vue"),
					meta: { title: "监控", auth: true, roles: "all" }
				},
				{
					path: "files",
					name: "Host-Files",
					component: () => import("@/views/Host/files.vue"),
					meta: { title: "文件", auth: true, roles: "all" }
				},
				{
					path: "disk",
					name: "Host-Disk",
					component: () => import("@/views/Host/disk.vue"),
					meta: { title: "磁盘管理", auth: true, roles: "all" }
				},
				{
					path: "firewall",
					name: "Host-Firewall",
					component: () => import("@/views/Host/firewall.vue"),
					meta: { title: "防火墙", auth: true, roles: "all" }
				},
				{
					path: "process",
					name: "Host-Process",
					component: () => import("@/views/Host/process.vue"),
					meta: { title: "进程管理", auth: true, roles: "all" }
				},
				{
					path: "security",
					name: "Host-Security",
					component: () => import("@/views/Host/security.vue"),
					meta: { title: "安全体检", auth: true, roles: "all" }
				},
				{
					path: "daemon",
					name: "Toolbox-Daemon",
					component: () => import("@/views/Host/Toolbox/Daemon.vue"),
					meta: { title: "进程守护", auth: false, roles: "all" }
				},
				{
					path: "cronjob",
					name: "Cronjob-Index",
					component: () => import("@/views/Cronjob/Index.vue"),
					meta: { title: "计划任务", titleKey: "menu.cronjob", auth: true, roles: "all" }
				}
			]
		},
		{
			path: "/setting",
			name: "Setting",
			component: () => import("@/views/Setting/Setting.vue"),
			meta: {
				title: "系统设置",
				auth: true,
				roles: "all"
			}
		},
		{
			path: "/logs",
			name: "Logs",
			component: () => import("@/views/Log/index.vue"),
			meta: {
				title: "日志中心",
				auth: true,
				roles: "all"
			}
		},
		{
			path: "/mobile",
			name: "MobileConsole",
			component: () => import("@/views/Mobile/Console.vue"),
			meta: {
				title: "GoPanel",
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				checkAuth: false,
				skipPin: true
			}
		},
		{
			path: "/mobile/auth",
			name: "MobileAuth",
			component: () => import("@/views/Mobile/Auth.vue"),
			meta: {
				title: "GoPanel",
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				checkAuth: false,
				skipPin: true
			}
		},
		{
			path: "/login",
			name: "Login",
			component: () => import("@/views/Auth/Login.vue"),
			meta: {
				title: "Login",
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				checkAuth: true,
				skipPin: true
			}
		},
		{
			path: "/forgot-password",
			name: "ForgotPassword",
			component: () => import("@/views/Auth/Login.vue"),
			props: { formType: "forgotpassword" as FormType },
			meta: {
				title: "Forgot Password",
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				checkAuth: true,
				skipPin: true
			}
		},
		{
			path: "/logout",
			name: "Logout",
			redirect: "/login"
		},
		{
			path: "/no-permission",
			name: "NoPermission",
			component: () => import("@/views/NoPermission.vue"),
			meta: {
				title: "No Permission",
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				checkAuth: true,
				skipPin: true
			}
		},
		{
			path: "/:pathMatch(.*)*",
			name: "NotFound",
			component: () => import("@/views/NotFound.vue"),
			meta: {
				theme: { layout: Layout.Blank, boxed: { enabled: false }, padded: { enabled: false } },
				skipPin: true
			}
		}
	]
})

const viewportMeta = document.querySelector<HTMLMetaElement>('meta[name="viewport"]')
const defaultViewport = viewportMeta?.content || "initial-scale=1, minimum-scale=1, width=device-width, height=device-height"

router.afterEach(route => {
	if (!viewportMeta) return
	viewportMeta.content = route.path === "/mobile" || route.path.startsWith("/mobile/")
		? "width=device-width, initial-scale=1, minimum-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover"
		: defaultViewport
})

checkAuth(router)

export default router
