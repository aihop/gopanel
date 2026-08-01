<template>
  <div
    v-if="isDark && !mini"
    class="logo"
  >
    <img
      :alt="appBrand"
      :src="logoText"
      width="120"
      height="80"
    />
  </div>
  <div
    v-else-if="isLight && !mini"
    class="logo"
  >
    <img
      :alt="appBrand"
      :src="logoText"
      width="120"
      height="80"
    />
  </div>
  <div
    v-else-if="isDark && mini"
    class="logo"
  >
    <img
      :alt="appBrand"
      :src="logoIcon"
      width="120"
      height="80"
    />
  </div>
  <div
    v-else-if="isLight && mini"
    class="logo"
  >
    <img
      :alt="appBrand"
      :src="logoIcon"
      width="120"
      height="80"
    />
  </div>
</template>

<script lang="ts" setup>
import { useThemeStore } from "@/store/theme"
import { computed } from "vue"

import logoTextDefault from "@/assets/images/logo-text.svg?url"
import logoIconDefault from "@/assets/images/logo-icon.svg?url"
 
const {
	mini,
	dark,
	maxHeight = "27px"
} = defineProps<{
	mini?: boolean
	dark?: boolean
	maxHeight?: string
}>()
const themeStore = useThemeStore()
const isDark = computed<boolean>(() => dark ?? themeStore.isThemeDark)
const isLight = computed<boolean>(() => !dark || themeStore.isThemeLight)
const appBrand = import.meta.env.VITE_APP_BRAND || "GoPanel"
const logoText = computed(() => logoTextDefault)
const logoIcon = computed(() => logoIconDefault)
</script>

<style lang="scss" scoped>
.logo {
	height: 100%;
	display: flex;
	align-items: center;
	img {
		max-height: v-bind(maxHeight);
		display: block;
		height: 100%;
	}
}
</style>
