<template>
	<nav class="nav" :class="[{ collapsed }, mode]">
		<n-menu
			ref="menu"
			v-model:value="selectedKey"
			:options="menuOptions"
			:collapsed="collapsed"
			:mode
			:indent="18"
			:collapsed-width="collapsedWidth"
			:dropdown-props="{
				scrollable: true,
				menuProps: () => ({
					class: 'main-nav'
				})
			}"
			:default-expanded-keys="['card']"
			:expanded-keys="expandedKeys"
			@update:expanded-keys="handleUpdateExpandedKeys"
		/>
	</nav>
</template>

<script lang="ts" setup>
import type { MenuInst } from "naive-ui"
import type { MenuMixedOption } from "naive-ui/es/menu/src/interface"
import type { RouteRecordNormalized } from "vue-router"
import { useThemeStore } from "@/store/theme"
import _uniq from "lodash/uniq"
import { NMenu } from "naive-ui"
import { computed, onBeforeMount, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import getItems from "./items"

const { mode = "vertical", collapsed = false } = defineProps<{
	mode?: "vertical" | "horizontal"
	collapsed?: boolean
}>()

const route = useRoute()
const router = useRouter()
const selectedKey = ref<string | null>(null)
const menu = ref<MenuInst | null>(null)
const expandedKeys = ref<string[] | undefined>(undefined)

const themeStore = useThemeStore()

const menuOptions = computed<MenuMixedOption[]>(() => getItems({ mode, collapsed }))
const collapsedWidth = computed<number>(() => themeStore.sidebar.closeWidth)
const sidebarCollapsed = computed<boolean>(() => themeStore.sidebar.collapsed)

function collectMenuKeys(options: MenuMixedOption[], keys = new Set<string>()) {
	for (const option of options) {
		if (typeof option.key === "string") {
			keys.add(option.key)
		}
		if (Array.isArray(option.children) && option.children.length > 0) {
			collectMenuKeys(option.children as MenuMixedOption[], keys)
		}
	}
	return keys
}

function resolveRouteMenuKey(name: string, menuKeys: Set<string>) {
	const candidates = [name, name.toLowerCase(), name.split("-")[0]?.toLowerCase()].filter(Boolean) as string[]
	return candidates.find(candidate => menuKeys.has(candidate)) ?? null
}

function setMenuKey(matched: RouteRecordNormalized[]) {
	const menuKeys = collectMenuKeys(menuOptions.value)
	for (let i = matched.length - 1; i >= 0; i--) {
		const match = matched[i]
		if (match.name && typeof match.name === "string") {
			const resolvedKey = resolveRouteMenuKey(match.name.toString(), menuKeys)
			if (resolvedKey) {
				selectedKey.value = resolvedKey
				menu.value?.showOption(selectedKey.value)
				return
			}
		}
	}
	selectedKey.value = null
}

function syncExpandedKeys(matched: RouteRecordNormalized[]) {
	const parentKeys = matched
		.map(match => (typeof match.name === "string" ? match.name.toString() : ""))
		.map(name => name.split("-")[0]?.toLowerCase())
		.filter(key => key && key !== selectedKey.value)
	expandedKeys.value = parentKeys.length ? _uniq(parentKeys) : undefined
}

onBeforeMount(() => {
	setMenuKey(route.matched)
	syncExpandedKeys(route.matched)

	router.afterEach(route => {
		if (route?.matched?.length) {
			setMenuKey(route.matched)
			syncExpandedKeys(route.matched)

			if (window.innerWidth <= 700 && !sidebarCollapsed.value) {
				themeStore.closeSidebar()
			}
		}
	})
})

watch(
	() => menuOptions.value,
	() => {
		setMenuKey(route.matched)
		syncExpandedKeys(route.matched)
	},
	{ deep: true }
)

// handler to simulate the accordion behavior in a specific submenu
function handleUpdateExpandedKeys(value: string[]) {
	const submenu = "components"

	if (value?.length && value.includes(submenu)) {
		const lastKey = value.pop()
		if (lastKey) {
			expandedKeys.value = _uniq([submenu, lastKey])
		}
	} else {
		expandedKeys.value = undefined
	}
}
</script>

<style lang="scss" scoped>
.nav {
	&.collapsed {
		pointer-events: none;
	}

	:deep() {
		.n-menu {
			--n-item-color-active: rgba(37, 99, 235, 0.08);
			--n-item-color-active-hover: rgba(37, 99, 235, 0.12);
			--n-item-color-hover: rgba(148, 163, 184, 0.08);
			--n-item-text-color-active: rgb(37, 99, 235);
			--n-item-text-color-active-hover: rgb(37, 99, 235);
			--n-item-icon-color-active: rgb(37, 99, 235);
			--n-item-icon-color-active-hover: rgb(37, 99, 235);
			--n-arrow-color-active: rgb(37, 99, 235);
			--n-border-radius: 14px;
		}

		.n-menu-item-content,
		.n-submenu-children .n-menu-item-content,
		.n-submenu .n-submenu-label {
			border-radius: 14px;
			margin: 2px 0;
			font-weight: 500;
		}

		.n-menu-item-group {
			.n-menu-item-group-title {
				white-space: nowrap;
				overflow: hidden;
				text-overflow: ellipsis;
			}
		}

		// .n-submenu-children {
		// 	--dash-width: 8px;
		// 	--dash-height: 2px;
		// 	--dash-offset: 29px;

		// 	position: relative;

		// 	&::before {
		// 		content: "";
		// 		display: block;
		// 		background-color: var(--border-color);
		// 		width: var(--dash-height);
		// 		position: absolute;
		// 		top: 0px;
		// 		bottom: 20px;
		// 		left: var(--dash-offset);
		// 	}

		// 	.n-menu-item-content {
		// 		&::after {
		// 			content: "";
		// 			display: block;
		// 			background-color: var(--border-color);
		// 			width: var(--dash-width);
		// 			height: var(--dash-height);
		// 			position: absolute;
		// 			z-index: -1;
		// 			top: calc(50% - calc(ar(--dash-height) / 2));
		// 			left: calc(var(--dash-offset) + var(--dash-height));
		// 		}
		// 	}

		// 	.n-menu-item-group {
		// 		.n-menu-item-group-title {
		// 			padding-left: 44px !important;
		// 		}
		// 	}

		// 	.n-submenu-children {
		// 		&::before {
		// 			display: none;
		// 		}
		// 		.n-menu-item-content {
		// 			&::after {
		// 				width: calc(var(--dash-width) * 3);
		// 				background: repeating-linear-gradient(
		// 					90deg,
		// 					var(--border-color) 0px,
		// 					var(--border-color) 5px,
		// 					transparent 5px,
		// 					transparent 8px
		// 				);
		// 			}
		// 		}
		// 	}
		// }

		.n-menu--horizontal {
			.n-menu-item-content {
				.n-menu-item-content-header {
					overflow: initial;
				}
			}
		}
	}
}

.direction-rtl {
	.nav {
		:deep() {
			.n-submenu-children {
				--dash-offset: 25px;
			}
		}
	}
}
</style>
