import type { MenuMixedOption } from "naive-ui/es/menu/src/interface"
import { renderIcon } from "@/utils"
import { h } from "vue"
import { RouterLink } from "vue-router"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"

// 主菜单图标
const ReportIcon = "mdi:file-chart-outline"

// 子菜单图标
const OverviewIcon = "mdi:view-dashboard-outline"
const RechargeIcon = "mdi:cash-plus"
const CardApplyIcon = "mdi:credit-card-plus-outline"
const CardListIcon = "mdi:format-list-bulleted"
const BatchOperationsIcon = "mdi:cards-outline"
const CardRecordsIcon = "mdi:playlist-check"
const TransactionIcon = "mdi:swap-horizontal"
const ConsumptionReportIcon = "mdi:chart-line"

export default function getItems(_: { mode: "vertical" | "horizontal"; collapsed: boolean }): MenuMixedOption[] {
	const authStore = useAuthStore()
	const userMenus = authStore.userMenus || []
	const isSuperAdmin = authStore.role === "SUPER" || authStore.role === "ADMIN" || userMenus.includes("ALL")

	// 判断某个主菜单/路由是否允许显示
	const hasPermission = (key: string) => {
		if (isSuperAdmin) return true
		return userMenus.includes(key)
	}

	const allItems: MenuMixedOption[] = [
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Dashboard-Index"
						}
					},
					{ default: () => t("menu.dashboard") }
				),
			key: "dashboard",
			icon: renderIcon("mdi:view-dashboard-outline")
		},
				{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "AIAgent-Index"
						}
					},
					{ default: () => t("menu.ai_tools") }
				),
			key: "ai",
			icon: renderIcon("mdi:robot-outline")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Website-Index"
						}
					},
					{ default: () => t("menu.website") }
				),
			key: "website",
			icon: renderIcon("mdi:web")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "SSL-Index"
						}
					},
					{ default: () => t("menu.ssl") }
				),
			key: "ssl",
			icon: renderIcon("mdi:shield-check-outline")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Database-Index"
						}
					},
					{ default: () => t("menu.database") }
				),
			key: "database",
			icon: renderIcon("mdi:database")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Container-Index"
						}
					},
					{ default: () => t("menu.container") }
				),
			key: "container",
			icon: renderIcon("mdi:docker")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Pipeline-Index"
						}
					},
					{ default: () => t("menu.pipeline") }
				),
			key: "pipeline",
			icon: renderIcon("mdi:source-merge")
		},
		{
			label: () =>
				h(
					RouterLink,
					{
						to: {
							name: "Apps-Index"
						}
					},
					{ default: () => t("menu.apps") }
				),
			key: "apps",
			icon: renderIcon("mdi:apps")
		},
		{
			label: t("menu.host"),
			key: "host",
			icon: renderIcon("mdi:server-outline"),
			children: [
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Host-Files"
								}
							},
							{ default: () => t("menu.files") }
						),
					key: "Host-Files"
				},
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Host-Firewall"
								}
							},
							{ default: () => t("menu.firewall") }
						),
					key: "Host-Firewall"
				},
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Host-Process"
								}
							},
							{ default: () => t("menu.processManage") }
						),
					key: "Host-Process"
				},
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Toolbox-Daemon"
								}
							},
							{ default: () => t("menu.daemon") }
						),
					key: "Toolbox-Daemon"
				},
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Host-Monitor"
								}
							},
							{ default: () => t("menu.monitor") }
						),
					key: "Host-Monitor"
				}
			]
		},
		{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Logs"
								}
							},
							{ default: () => t("menu.logs") }
						),
					key: "logs",
					icon: renderIcon("ion:document-text-outline")
				},
				{
					label: () =>
						h(
							RouterLink,
							{
								to: {
									name: "Setting"
								}
							},
							{ default: () => t("menu.setting") }
						),
					key: "setting",
					icon: renderIcon("ion:settings-outline")
				}
	]

	// 过滤有权限的菜单
	const filterMenu = (items: MenuMixedOption[]): MenuMixedOption[] => {
		return items.reduce((result: MenuMixedOption[], item) => {
			// 如果是超级管理员，或者父级本身就有权限，或者它是没有 children 的项且有权限
			const itemHasPerm = hasPermission(item.key as string)
			
			if (item.children) {
				// 递归过滤子菜单
				const filteredChildren = filterMenu(item.children as MenuMixedOption[])
				// 如果父级有权限，或者它有至少一个有权限的子菜单，则保留该父级
				if (itemHasPerm || filteredChildren.length > 0) {
					result.push({
						...item,
						children: filteredChildren.length > 0 ? filteredChildren : undefined
					})
				}
			} else {
				// 对于没有 children 的项，直接判断权限
				if (itemHasPerm) {
					result.push(item)
				}
			}
			return result
		}, [])
	}

	return filterMenu(allItems)
}
