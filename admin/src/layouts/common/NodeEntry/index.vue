<template>
	<!--
		侧栏底部的「多节点」独立入口。
		刻意不放在 Navbar 的菜单数组里：多节点是"换一台机器操作"的场景切换，
		和网站/数据库那些业务菜单不是同一层级，混在列表里会被当成又一个功能页。
		这里固定在菜单下方（不随菜单滚动），悬停出节点状态浮层，点进多节点页。
	-->
	<div v-if="canManageNodes" class="node-entry" :class="{ collapsed }">
		<n-popover
			trigger="hover"
			placement="right-end"
			:show-arrow="false"
			:delay="120"
			:disabled="!nodeStore.hasNodes"
			raw
			class="node-entry-popover"
		>
			<template #trigger>
				<div
					class="entry-row flex cursor-pointer items-center rounded-lg"
					:class="{ 'is-active': isNodeRoute }"
					@click="goNodePage"
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
			</template>

			<div class="node-flyout">
				<div class="flyout-head flex items-center justify-between">
					<span class="font-medium">{{ t("menu.node") }}</span>
					<span class="opacity-60">{{ t("node.entry.count", { count: nodeStore.list.length }) }}</span>
				</div>

				<div class="flyout-body">
					<div
						v-for="node of nodeStore.list"
						:key="node.id"
						class="flyout-item flex cursor-pointer items-center gap-2"
						@click="openWorkspace(node.id)"
					>
						<Icon :name="DotIcon" :size="9" :class="levelDotClass[nodeLevel(node)]" class="shrink-0" />
						<span class="item-name truncate">{{ node.name }}</span>
						<span class="item-state ml-auto shrink-0 truncate">{{ nodeStateText(node) }}</span>
					</div>
				</div>

				<div class="flyout-foot flex cursor-pointer items-center justify-between" @click="goNodePage">
					<span>{{ t("node.entry.enter") }}</span>
					<Icon :name="ArrowIcon" :size="14" />
				</div>
			</div>
		</n-popover>
	</div>
</template>

<script lang="ts" setup>
import type { NodeItem } from "@/api/modules/node"
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import NodeStore from "@/store/modules/node"
import { NBadge, NPopover } from "naive-ui"
import { computed, onMounted } from "vue"
import { useRoute, useRouter } from "vue-router"
import { levelDotClass, nodeLevel, statusText, warningText } from "../NodeRail/nodeDisplay"

const { collapsed = false } = defineProps<{
	collapsed?: boolean
}>()

const NodeIcon = "mdi:server-network-outline"
const DotIcon = "mdi:circle"
const ArrowIcon = "mdi:arrow-right"

const route = useRoute()
const router = useRouter()
const nodeStore = NodeStore()
const authStore = useAuthStore()

// 多节点已不在主菜单里，Navbar 在这个页面上不会高亮任何项（selectedKey 置空），
// 所以选中态得由这个入口自己表达
const isNodeRoute = computed(() => route.path === "/node" || route.path.startsWith("/node/"))

// 与 NodeRail 一致：节点接口是 admin-only，子管理员拉取只会拿到 401
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

/** 有告警就显示第一条告警，否则显示在线状态 */
function nodeStateText(node: NodeItem): string {
	const warnings = node.warnings || []
	if (warnings.length > 0) {
		const danger = warnings.find(item => item.level === "danger")
		return warningText(danger || warnings[0])
	}
	return statusText(node.status)
}

function goNodePage() {
	router.push({ name: "Node-Index" })
}

/** 点浮层里的某台机器：直接开该节点的工作区抽屉，不用先进列表页再找 */
function openWorkspace(id: number) {
	nodeStore.openDrawer(id)
}

onMounted(() => {
	// NodeRail 常驻并负责轮询；这里只在还没读过时补一次，
	// 保证即使细条被隐藏（railHidden）角标也有数
	if (canManageNodes.value && !nodeStore.loaded) {
		nodeStore.fetchList()
	}
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

.node-flyout {
	min-width: 240px;
	max-width: 300px;
	overflow: hidden;
	border: 1px solid var(--border-color);
	border-radius: 12px;
	background-color: var(--bg-body-color, var(--bg-default-color));
	box-shadow: 0 12px 32px rgb(15 23 42 / 14%);
	font-size: 13px;

	.flyout-head {
		padding: 10px 12px;
		border-bottom: 1px solid rgba(var(--border-color-rgb) / 0.7);
	}

	.flyout-body {
		max-height: 260px;
		overflow-y: auto;
		padding: 4px 0;
	}

	.flyout-item {
		padding: 7px 12px;

		&:hover {
			background-color: var(--bg-secondary-color);
		}

		.item-state {
			max-width: 130px;
			font-size: 12px;
			opacity: 0.7;
		}
	}

	.flyout-foot {
		padding: 9px 12px;
		border-top: 1px solid rgba(var(--border-color-rgb) / 0.7);
		color: var(--primary-color);

		&:hover {
			background-color: rgba(var(--primary-color-rgb) / 0.1);
		}
	}
}
</style>
