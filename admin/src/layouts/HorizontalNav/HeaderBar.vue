<template>
	<div class="header-bar px-4 pb-3 sm:px-6">
		<div
			class="header-panel flex min-h-[var(--header-bar-height)] items-center gap-3 rounded-[24px] px-4 shadow-[0_10px_30px_rgba(37,99,235,0.06)] backdrop-blur-xl sm:px-5"
		>
			<Logo :mini="false" :dark="isDark" class="anim-wrap shrink-0" />
			<n-scrollbar class="grow" x-scrollable>
				<Navbar :collapsed="false" mode="horizontal" />
			</n-scrollbar>
		</div>
	</div>
</template>

<script lang="ts" setup>
import Logo from "@/layouts/common/Logo.vue"
import Navbar from "@/layouts/common/Navbar"
import { useThemeStore } from "@/store/theme"
import { NScrollbar } from "naive-ui"
import { computed } from "vue"

const themeStore = useThemeStore()
const isDark = computed<boolean>(() => themeStore.isThemeDark)
</script>

<style lang="scss" scoped>
@import "./variables";

.header-bar {
	background: linear-gradient(
		180deg,
		rgba(var(--bg-sidebar-color-rgb) / 0.52) 0%,
		rgba(var(--bg-sidebar-color-rgb) / 0) 100%
	);

	.header-panel {
		border: 1px solid rgba(var(--border-color-rgb) / 0.68);
		background: rgba(var(--bg-sidebar-color-rgb) / 0.88);
		box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
	}

	:deep(.n-scrollbar-rail) {
		opacity: 0.15;
	}

	.nav {
		padding-top: 8px;
	}

	@media (max-width: $sidebar-bp) {
		display: none;
	}
}

.direction-rtl {
	.header-bar {
		direction: rtl;
	}
}
</style>
