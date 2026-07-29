<template>
	<!--
		侧栏底部的「多节点」独立入口。
		刻意不放在 Navbar 的菜单数组里：多节点是"换一台机器操作"的场景切换，
		和网站/数据库那些业务菜单不是同一层级，混在列表里会被当成又一个功能页。
		这里固定在菜单下方（不随菜单滚动），点击后打开节点抽屉。
	-->
	<div v-if="canManageNodes" class="node-entry" :class="{ collapsed }">
		<div
			class="entry-row flex cursor-pointer items-center rounded-lg"
			:class="{ 'is-active': isNodeRoute || nodeStore.drawerVisible }"
			@click="nodeStore.openDrawer()"
		>
			<Icon :name="NodeIcon" :size="18" class="entry-icon shrink-0" />
			<span v-if="!collapsed" class="entry-label grow truncate">{{ t("menu.node") }}</span>
			<n-badge
				v-if="nodeStore.hasNodes"
				:value="badgeValue"
				:type="badgeType"
				:show-zero="false"
				class="shrink-0"
			/>
		</div>
	</div>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import NodeStore from "@/store/modules/node"
import { NBadge } from "naive-ui"
import { computed, onBeforeUnmount, onMounted } from "vue"
import { useRoute } from "vue-router"

const { collapsed = false } = defineProps<{
	collapsed?: boolean
}>()

const NodeIcon = "mdi:server-network-outline"

const route = useRoute()
const nodeStore = NodeStore()
const authStore = useAuthStore()

// 多节点已不在主菜单里，Navbar 在这个页面上不会高亮任何项（selectedKey 置空），
// 所以选中态得由这个入口自己表达
const isNodeRoute = computed(() => route.path === "/node" || route.path.startsWith("/node/"))

// 节点接口是 admin-only，子管理员拉取只会拿到 401
const canManageNodes = computed(
	() => authStore.role === "SUPER" || authStore.role === "ADMIN" || (authStore.userMenus || []).includes("ALL")
)

// 角标只报"需要关注的台数"，全部正常时报总数，避免一个没事的角标天天飘红
const badgeValue = computed(() => {
	if (nodeStore.dangerCount > 0) return nodeStore.dangerCount
	if (nodeStore.warnCount > 0) return nodeStore.warnCount
	return nodeStore.list.length
})

const badgeType = computed<"error" | "warning" | "default">(() => {
	if (nodeStore.dangerCount > 0) return "error"
	if (nodeStore.warnCount > 0) return "warning"
	return "default"
})

let timer = 0

onMounted(() => {
	if (!canManageNodes.value) return
	if (!nodeStore.loaded) nodeStore.fetchList()
	timer = nodeStore.startAutoRefresh()
})

onBeforeUnmount(() => {
	if (timer) window.clearInterval(timer)
})
</script>

<style lang="scss" scoped>
.node-entry {
	border-top: 1px solid rgba(var(--border-color-rgb) / 0.7);
	padding: 8px;

	.entry-row {
		gap: 10px;
		padding: 9px 12px;
		color: var(--fg-secondary-color);
		transition:
			background-color 0.2s var(--bezier-ease),
			color 0.2s var(--bezier-ease);

		&:hover {
			background-color: rgba(var(--primary-color-rgb) / 0.1);
			color: var(--primary-color);
		}

		&.is-active {
			background-color: rgba(var(--primary-color-rgb) / 0.14);
			color: var(--primary-color);
			font-weight: 500;
		}
	}

	.entry-label {
		font-size: 14px;
	}

	&.collapsed .entry-row {
		justify-content: center;
		padding-left: 0;
		padding-right: 0;
	}
}
</style>
