<template>
  <header
    class="toolbar px-4 pb-6 pt-4 sm:px-10"
    :class="{ boxed }"
  >
    <div class="wrap toolbar-panel flex items-center justify-end gap-3 rounded-[24px] px-4 shadow-[0_10px_30px_rgba(37,99,235,0.06)] backdrop-blur-xl sm:px-5">
      <div
        class="logo-box toolbar-trigger flex cursor-pointer items-center gap-2 rounded-2xl px-3 py-2 transition-colors"
        @click="openNav()"
      >
        <Logo mini />
        <Icon
          :size="20"
          name="carbon:chevron-right"
        />
      </div>

      <Breadcrumb class="grow" />

      <SystemUpdateEntry v-if="isLogged" />
      <Search />
      <MobileAdminSwitch />
      <LocaleSwitch />
      <FullscreenSwitch />
      <ThemeSwitch />
      <Avatar class="cursor-pointer" />
    </div>

    <BlurEffect />
  </header>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { useLoadingBarSetup } from "@/composables/useLoadingBarSetup"
import { useThemeStore } from "@/store/theme"
import Logo from "../Logo.vue"
import Avatar from "./Avatar.vue"
import BlurEffect from "./BlurEffect.vue"
import Breadcrumb from "./Breadcrumb.vue"
import FullscreenSwitch from "./FullscreenSwitch.vue"
import LocaleSwitch from "./LocaleSwitch.vue"
import MobileAdminSwitch from "./MobileAdminSwitch.vue"
import Search from "./Search.vue"
import SystemUpdateEntry from "./SystemUpdateEntry.vue"
import ThemeSwitch from "./ThemeSwitch.vue"
import { useAuthStore } from "@/store/auth"
import { computed } from "vue"

const { boxed } = defineProps<{
	boxed: boolean
}>()

const themeStore = useThemeStore()
const authStore = useAuthStore()
const isLogged = computed(() => authStore.isLogged)
const openNav = () => themeStore.openSidebar()

useLoadingBarSetup()
</script>

<style lang="scss" scoped>
.toolbar {
	position: sticky;
	top: 0;
	left: 0;
	width: 100%;
	max-width: 100%;
	z-index: 3;
	overflow: visible;

	.wrap {
		height: var(--toolbar-height);
		width: 100%;
		max-width: 100%;
		position: relative;
		z-index: 1;

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

	.toolbar-panel {
		border: 1px solid rgba(var(--border-color-rgb) / 0.68);
		background: rgba(var(--bg-sidebar-color-rgb) / 0.88);
		box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
	}

	.toolbar-trigger {
		border: 1px solid rgba(var(--border-color-rgb) / 0.72);
		background: rgba(var(--bg-body-color-rgb) / 0.72);
		color: var(--fg-secondary-color);

		&:hover {
			border-color: rgba(var(--primary-color-rgb) / 0.28);
			background: rgba(var(--primary-color-rgb) / 0.1);
			color: var(--primary-color);
		}
	}

	&.boxed {
		.wrap {
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
