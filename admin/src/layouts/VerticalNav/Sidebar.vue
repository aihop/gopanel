<template>
	<aside
		id="app-sidebar"
		class="sidebar flex flex-col"
		:class="{ collapsed: sidebarCollapsed, opened: !sidebarCollapsed }"
	>
		<div
			ref="sidebar"
			class="sidebar-wrap sidebar-panel my-3 flex grow flex-col overflow-hidden transition-all"
			:class="sidebarClosed ? 'mx-0 rounded-lg' : 'mx-3 rounded-[28px]'"
		>
			<div :class="{ 'px-7': !sidebarClosed, 'px-2': sidebarClosed }" class="transition-all">
				<SidebarHeader :logo-mini="sidebarClosed" />
			</div>
			<!-- grow + min-h-0：让菜单区吃掉剩余高度并可滚动，多节点入口才会真的贴在底部 -->
			<n-scrollbar class="min-h-0 grow">
				<div :class="{ 'px-2': !sidebarClosed }" class="transition-all">
					<Navbar :collapsed="sidebarClosed" />
				</div>
			</n-scrollbar>
			<!-- 放在 n-scrollbar 外面：多节点入口要固定在底部，不跟着菜单滚动 -->
			<NodeEntry :collapsed="sidebarClosed" />
			<!-- <div class="p-2">
				<SidebarFooter :collapsed="sidebarClosed" />
			</div> -->
		</div>
	</aside>
</template>

<script lang="ts" setup>
import Navbar from "@/layouts/common/Navbar"
import NodeEntry from "@/layouts/common/NodeEntry/index.vue"
import { useThemeStore } from "@/store/theme"
import { isMobile } from "@/utils"
import { onClickOutside, useElementHover } from "@vueuse/core"
import { NScrollbar } from "naive-ui"
import { computed, onMounted, ref, watch } from "vue"
// import SidebarFooter from "./SidebarFooter.vue"
import SidebarHeader from "./SidebarHeader.vue"

const themeStore = useThemeStore()
const sidebar = ref(null)
const sidebarHovered = useElementHover(sidebar)
const sidebarCollapsed = computed(() => themeStore.sidebar.collapsed)
const sidebarClosed = computed(() => !sidebarHovered.value && sidebarCollapsed.value)

function clickListener() {
	if (sidebar.value) {
		onClickOutside(sidebar, e => {
			if (!sidebarCollapsed.value) {
				e.stopPropagation()
				themeStore.closeSidebar()
			}
		})
	}
}

onMounted(() => {
	watch(
		sidebarCollapsed,
		val => {
			if (val) {
				if (isMobile()) {
					sidebarHovered.value = false
				}
			}
		},
		{
			immediate: true
		}
	)

	if (window.innerWidth <= 700) {
		clickListener()
	}
})
</script>

<style lang="scss" scoped>
@import "./variables";

.sidebar {
	position: fixed;
	z-index: 4;
	top: 0;
	left: 0;
	width: var(--sidebar-open-width);
	height: 100vh;
	height: 100svh;
	overflow-x: hidden;
	overflow-y: auto;
	transition:
		width var(--sidebar-anim-ease) var(--sidebar-anim-duration),
		box-shadow var(--sidebar-anim-ease) var(--sidebar-anim-duration),
		color 0.3s var(--bezier-ease) 0s,
		background-color 0.3s var(--bezier-ease) 0s;

	.sidebar-wrap {
		overflow: hidden;
	}

	.sidebar-panel {
		border: 1px solid rgba(var(--border-color-rgb) / 0.7);
		background: rgba(var(--bg-sidebar-color-rgb) / 0.92);
	}

	:deep(.n-scrollbar-rail) {
		opacity: 0.15;
	}

	&.collapsed {
		width: var(--sidebar-close-width);

		&:hover {
			width: var(--sidebar-open-width);
			box-shadow: 0px 24px 56px 0px rgba(37, 99, 235, 0.14);
		}
	}

	@media (max-width: $sidebar-bp) {
		z-index: -1;
		transition: all 0.3s var(--bezier-ease) 0s;
		transform: translateX(-100%);

		&.opened {
			z-index: 2100;
			transform: translateX(0);
			box-shadow: 0px 24px 56px 0px rgba(15, 23, 42, 0.18);
		}
	}
}

.direction-rtl {
	.sidebar {
		left: unset;
		right: 0;

		@media (max-width: $sidebar-bp) {
			transform: translateX(100%);

			&.opened {
				transform: translateX(0%);
			}
		}
	}
}
</style>
