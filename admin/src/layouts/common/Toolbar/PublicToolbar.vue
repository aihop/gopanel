<template>
	<header class="toolbar" :class="{ boxed }">
		<div
			class="wrap flex items-center justify-between gap-3 rounded-[24px] px-4 shadow-[0_10px_30px_rgba(37,99,235,0.06)] backdrop-blur-xl sm:px-5"
		>
			<div class="logo-box flex cursor-pointer items-center gap-2 rounded-2xl px-3 py-2">
				<Logo />
			</div>

			<div class="toolbar-actions">
				<LocaleSwitch />
				<FullscreenSwitch />
				<ThemeSwitch />
				<Avatar :inner="false" />
			</div>
		</div>

		<BlurEffect />
	</header>
</template>

<script lang="ts" setup>
import { useLoadingBarSetup } from "@/composables/useLoadingBarSetup"
import Logo from "../Logo.vue"
import Avatar from "./Avatar.vue"
import BlurEffect from "./BlurEffect.vue"
import FullscreenSwitch from "./FullscreenSwitch.vue"
import LocaleSwitch from "./LocaleSwitch.vue"
import ThemeSwitch from "./ThemeSwitch.vue"

const { boxed } = defineProps<{
	boxed: boolean
}>()

useLoadingBarSetup()
</script>

<style lang="scss" scoped>
.toolbar {
	position: sticky;
	top: 0;
	left: 0;
	height: var(--toolbar-height);
	width: 100%;
	max-width: 100%;
	padding: 0 var(--view-padding);
	z-index: 3;
	overflow: visible;

	.wrap {
		height: var(--toolbar-height);
		width: 100%;
		max-width: 100%;
		position: relative;
		z-index: 1;
		border: 1px solid rgba(var(--border-color-rgb) / 0.68);
		background: rgba(var(--bg-sidebar-color-rgb) / 0.88);
		box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);

		@media (max-width: 850px) {
			.pinned-pages {
				display: none;
			}
		}

		@media (max-width: 700px) {
			justify-content: space-between;
			.breadcrumb {
				display: none;
			}
		}
	}

	.toolbar-actions {
		display: inline-flex;
		align-items: center;
		gap: 14px;
		padding: 6px;
		border-radius: 9999px;
		border: 1px solid rgba(var(--border-color-rgb) / 0.72);
		background: rgba(var(--bg-body-color-rgb) / 0.72);
		backdrop-filter: blur(12px);
		box-shadow: 0 4px 18px rgba(37, 99, 235, 0.08);
	}

	&.boxed {
		padding: 0;
		.wrap {
			padding: 0 var(--view-padding);
			max-width: var(--boxed-width);
			margin: 0 auto;
		}
	}

	&.gradient-bg-sidebar {
		background: linear-gradient(
			180deg,
			rgba(var(--bg-sidebar-color-rgb) / 0.72) 0%,
			rgba(var(--bg-sidebar-color-rgb) / 0) 100%
		);
	}
	&.gradient-bg-body {
		background: linear-gradient(
			180deg,
			rgba(var(--bg-sidebar-color-rgb) / 0.32) 0%,
			rgba(var(--bg-body-color-rgb) / 0) 100%
		);
	}
}

.direction-rtl {
	.toolbar {
		.wrap {
			.logo-box {
				.n-icon {
					transform: rotateY(180deg);
				}
			}
		}
	}
}
</style>
