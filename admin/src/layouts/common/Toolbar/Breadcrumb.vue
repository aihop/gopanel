<template>
  <n-breadcrumb class="breadcrumb">
    <n-breadcrumb-item @click="goto({ path: '/' })">
      <Icon
        :size="16"
        :name="HomeIcon"
      />
    </n-breadcrumb-item>
    <TransitionGroup name="anim">
      <n-breadcrumb-item
        v-for="(item, index) of items"
        :key="item.key"
        :class="`index-${index}`"
        @click="goto({ path: item.path })"
      >
        {{ item.name }}
      </n-breadcrumb-item>
    </TransitionGroup>
  </n-breadcrumb>
</template>

<script lang="ts" setup>
import type { RouteLocationNormalizedLoaded } from "vue-router"
import Icon from "@/components/common/Icon.vue"
import _isEqual from "lodash/isEqual"
import { NBreadcrumb, NBreadcrumbItem } from "naive-ui"
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"

interface Page {
	name: string
	path: string
	key: string
}

const HomeIcon = "fluent:home-24-regular"
const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const items = ref<Page[]>([])

function goto(page: Partial<Page>) {
	if (page.name && page.name !== route.name) {
		router.push({ name: page.name })
	}
	if (page.path && page.path !== route.path) {
		router.push({ path: page.path })
	}
}

function resolveRouteTitle(matchedRoute: RouteLocationNormalizedLoaded["matched"][number]) {
	if (typeof matchedRoute.meta?.titleKey === "string" && matchedRoute.meta.titleKey) {
		return t(matchedRoute.meta.titleKey)
	}
	if (typeof matchedRoute.meta?.title === "string") {
		return matchedRoute.meta.title
	}
	return ""
}

function checkRoute(route: RouteLocationNormalizedLoaded) {
	const newItems: Page[] = []
	if (route.matched && route.matched.length > 0) {
		for (let i = 0; i < route.matched.length; i++) {
			const matchedRoute = route.matched[i]
			const title = resolveRouteTitle(matchedRoute)
			if (title) {
				newItems.push({
					name: title,
					path: matchedRoute.path,
					key: `breadcrumb-${i}-${matchedRoute.path}`
				})
			}
		}
	}

	if (!_isEqual(items.value, newItems)) {
		items.value = newItems
	}
}

watch([() => route.fullPath, () => locale.value], () => {
	checkRoute(router.currentRoute.value)
}, { immediate: true })
</script>

<style lang="scss" scoped>
.breadcrumb {
	.anim-move,
	.anim-enter-active {
		transition: all 0.5s var(--bezier-ease);

		@for $i from 0 through 10 {
			&.index-#{$i} {
				transition-delay: $i * 0.1s;
			}
		}
	}

	.anim-leave-active {
		display: none;
	}

	.anim-enter-from {
		opacity: 0;
		transform: translateX(-5px);
	}
}
</style>
