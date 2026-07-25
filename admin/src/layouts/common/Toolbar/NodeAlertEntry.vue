<template>
	<!--
		顶栏告警入口：只在真的有节点异常时出现。
		存在的理由是细条可以被用户隐藏（railHidden），隐藏后就完全没有感知通道了——
		这个入口保证"有节点出问题"这件事在任何布局状态下都能看到。
	-->
	<n-tooltip v-if="visible" placement="bottom">
		<template #trigger>
			<n-badge :value="alertCount" :type="badgeType" :offset="[-2, 4]">
				<div class="toolbar-trigger cursor-pointer rounded-2xl px-3 py-2" @click="nodeStore.openDrawer()">
					<Icon :size="20" name="mdi:server-network-outline" :class="iconClass" />
				</div>
			</n-badge>
		</template>
		{{ tooltipText }}
	</n-tooltip>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import NodeStore from "@/store/modules/node"
import { computed } from "vue"

const nodeStore = NodeStore()
const authStore = useAuthStore()

const canManageNodes = computed(
	() => authStore.role === "SUPER" || authStore.role === "ADMIN" || (authStore.userMenus || []).includes("ALL")
)

const alertCount = computed(() => nodeStore.dangerCount + nodeStore.warnCount)

// 一切正常时不显示，避免顶栏多一个常亮但没信息量的图标
const visible = computed(() => canManageNodes.value && nodeStore.hasNodes && alertCount.value > 0)

const badgeType = computed(() => (nodeStore.dangerCount > 0 ? "error" : "warning"))
const iconClass = computed(() => (nodeStore.dangerCount > 0 ? "text-red-500" : "text-amber-500"))

const tooltipText = computed(() => {
	if (nodeStore.offlineCount > 0) {
		return t("node.alert.offlineAndWarn", { offline: nodeStore.offlineCount, total: alertCount.value })
	}
	return t("node.alert.warnOnly", { total: alertCount.value })
})
</script>
