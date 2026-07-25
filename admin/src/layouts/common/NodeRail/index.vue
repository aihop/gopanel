<template>
	<!--
		常驻细条：宽度恒定 48px，永不折叠。
		宽度恒定是刻意的——布局宽度一旦会变，xterm 就需要 refit（Terminal.vue 只监听 window resize，
		容器宽度变化不会触发），所以这里用"细条常驻 + 抽屉浮层"而不是可折叠侧栏。
	-->
	<div v-if="visible" class="node-rail flex flex-col items-center gap-1 border-l py-2">
		<n-tooltip placement="left">
			<template #trigger>
				<n-button quaternary size="tiny" class="mb-1" @click="nodeStore.toggleDrawer()">
					<template #icon>
						<Icon :name="ExpandIcon" :size="16" />
					</template>
				</n-button>
			</template>
			{{ t("node.rail.expand") }}
		</n-tooltip>

		<n-tooltip v-for="node of nodeStore.list" :key="node.id" placement="left">
			<template #trigger>
				<div
					class="node-dot flex cursor-pointer flex-col items-center justify-center rounded"
					:class="{ 'is-prod': node.isProd }"
					@click="nodeStore.openDrawer(node.id)"
					@dblclick="openNodePanel(node)"
				>
					<Icon :name="DotIcon" :size="10" :class="levelDotClass[nodeLevel(node)]" />
					<span class="initials">{{ nodeInitials(node.name) }}</span>
				</div>
			</template>
			<div class="text-xs">
				<div class="font-medium">{{ node.name }}</div>
				<div class="opacity-75">{{ statusText(node.status) }}</div>
				<div class="opacity-60">{{ t("node.rail.dblclickHint") }}</div>
				<div v-for="(warning, index) of node.warnings" :key="index" class="opacity-90">
					{{ warningText(warning) }}
				</div>
			</div>
		</n-tooltip>

		<div class="grow" />

		<n-tooltip placement="left">
			<template #trigger>
				<n-button quaternary size="tiny" @click="nodeStore.setRailHidden(true)">
					<template #icon>
						<Icon :name="HideIcon" :size="14" />
					</template>
				</n-button>
			</template>
			{{ t("node.rail.hide") }}
		</n-tooltip>
	</div>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import NodeStore from "@/store/modules/node"
import { computed, onBeforeUnmount, onMounted } from "vue"
import { levelDotClass, nodeInitials, nodeLevel, openNodePanel, statusText, warningText } from "./nodeDisplay"

const ExpandIcon = "mdi:chevron-left"
const HideIcon = "mdi:eye-off-outline"
const DotIcon = "mdi:circle"

const nodeStore = NodeStore()
const authStore = useAuthStore()

// 节点接口是 admin-only，子管理员拉取只会拿到 401，没必要每分钟撞一次
const canManageNodes = computed(
	() => authStore.role === "SUPER" || authStore.role === "ADMIN" || (authStore.userMenus || []).includes("ALL")
)

// 一台节点都没配时不占用横向空间；用户主动隐藏后也不显示
const visible = computed(() => canManageNodes.value && nodeStore.hasNodes && !nodeStore.railHidden)

let timer = 0

onMounted(() => {
	if (!canManageNodes.value) return
	nodeStore.fetchList()
	timer = nodeStore.startAutoRefresh()
})

onBeforeUnmount(() => {
	if (timer) {
		window.clearInterval(timer)
		timer = 0
	}
})
</script>

<style lang="scss" scoped>
.node-rail {
	width: 48px;
	flex: 0 0 48px;
	border-color: var(--border-color);
	background-color: var(--bg-sidebar-color);
	overflow-y: auto;
	overflow-x: hidden;
}

.node-dot {
	width: 36px;
	padding: 3px 0;

	&:hover {
		background-color: var(--bg-secondary-color);
	}

	.initials {
		font-size: 10px;
		line-height: 12px;
		opacity: 0.7;
	}

	&.is-prod .initials {
		color: var(--error-color);
		opacity: 1;
		font-weight: 600;
	}
}
</style>
