<script setup lang="ts">
import type { NodeItem } from "@/api/modules/node"
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import {
	levelDotClass,
	nodeLevel,
	openNodePanel,
	statusText,
	usageColor,
	warningText
} from "@/layouts/common/NodeRail/nodeDisplay"
import { computed, ref, watch } from "vue"
import RemoteContainerPanel from "./RemoteContainerPanel.vue"

const props = defineProps<{
	show: boolean
	node: NodeItem | null
}>()

const emit = defineEmits<{ (e: "update:show", value: boolean): void }>()

const tab = ref("overview")

const visible = computed({
	get: () => props.show,
	set: (value: boolean) => emit("update:show", value)
})

const canControl = computed(() => !!props.node?.hasControlToken)
const isOnline = computed(() => props.node?.status === "online")

// 换节点时回到概览：容器面板停在上一台的数据上会让人误判操作对象
watch(
	() => props.node?.id,
	() => {
		tab.value = "overview"
	}
)

function clamp(percent?: number): number {
	if (!percent || percent < 0) return 0
	return percent > 100 ? 100 : percent
}
</script>

<template>
	<!--
		节点工作区：在主控里直接操作远程节点，请求走 /node-proxy/{id} 代理。
		用大号抽屉而不是 iframe——iframe 会遇到节点登录态、SameSite=Lax 的入口 cookie
		和 https 页面加载 http 节点的混合内容拦截，三个问题都绕不开。
	-->
	<n-drawer v-model:show="visible" :width="720" placement="right" :trap-focus="false">
		<n-drawer-content closable body-content-style="padding: 16px;">
			<template #header>
				<div v-if="node" class="flex min-w-0 items-center gap-2">
					<Icon name="mdi:circle" :size="10" :class="levelDotClass[nodeLevel(node)]" />
					<span class="truncate font-medium">{{ node.name }}</span>
					<n-tag v-if="node.isProd" size="tiny" type="error" :bordered="false">{{ t("node.prod") }}</n-tag>
					<n-tag size="tiny" :bordered="false">{{ statusText(node.status) }}</n-tag>
					<n-button quaternary size="tiny" @click="openNodePanel(node)">
						<template #icon>
							<Icon name="mdi:open-in-new" :size="13" />
						</template>
					</n-button>
				</div>
			</template>

			<div v-if="!node" class="opacity-60">{{ t("node.workspace.noNode") }}</div>

			<template v-else>
				<!-- 操作对象提示常驻：远程操作最大的风险是搞错机器 -->
				<n-alert type="info" :show-icon="false" class="mb-3 text-xs">
					{{ t("node.workspace.actingOn", { node: node.name, addr: node.addr }) }}
				</n-alert>

				<n-alert v-if="!isOnline" type="warning" :show-icon="false" class="mb-3 text-xs">
					{{ node.statusMsg || t("node.workspace.offline") }}
				</n-alert>

				<n-tabs v-model:value="tab" type="line" animated>
					<n-tab-pane name="overview" :tab="t('node.workspace.tabOverview')">
						<div v-if="isOnline" class="flex flex-col gap-3">
							<div class="grid grid-cols-2 gap-3">
								<div class="metric-card">
									<div class="metric-title">{{ t("node.metric.cpu") }}</div>
									<n-progress
										type="line"
										:percentage="clamp(node.summary.cpuPercent)"
										:color="usageColor(node.summary.cpuPercent)"
										:height="6"
									/>
									<div class="metric-sub">{{ node.summary.cpuTotal }} core</div>
								</div>
								<div class="metric-card">
									<div class="metric-title">{{ t("node.metric.memory") }}</div>
									<n-progress
										type="line"
										:percentage="clamp(node.summary.memPercent)"
										:color="usageColor(node.summary.memPercent)"
										:height="6"
									/>
									<div class="metric-sub">{{ node.summary.hostname }}</div>
								</div>
							</div>

							<n-descriptions bordered size="small" :column="2">
								<n-descriptions-item :label="t('node.workspace.os')">
									{{ node.summary.os || "—" }}
								</n-descriptions-item>
								<n-descriptions-item :label="t('node.column.version')">
									{{ node.version || "—" }}
								</n-descriptions-item>
								<n-descriptions-item :label="t('node.metric.container')">
									{{ node.summary.containerRunning }}/{{ node.summary.containerTotal }}
								</n-descriptions-item>
								<n-descriptions-item :label="t('node.metric.cert')">
									{{ node.summary.certTotal }}
								</n-descriptions-item>
							</n-descriptions>

							<div v-if="node.warnings?.length" class="flex flex-col gap-1">
								<div
									v-for="(warning, index) of node.warnings"
									:key="index"
									class="text-xs"
									:class="warning.level === 'danger' ? 'text-red-500' : 'text-amber-500'"
								>
									{{ warningText(warning) }}
								</div>
							</div>
						</div>
						<n-empty v-else :description="t('node.workspace.noSummary')" />
					</n-tab-pane>

					<n-tab-pane name="container" :tab="t('node.metric.container')" display-directive="if">
						<RemoteContainerPanel
							:node-id="node.id"
							:node-name="node.name"
							:can-control="canControl"
						/>
					</n-tab-pane>
				</n-tabs>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<style lang="scss" scoped>
.metric-card {
	border: 1px solid var(--border-color);
	border-radius: var(--border-radius);
	padding: 8px 10px;

	.metric-title {
		font-size: 12px;
		opacity: 0.7;
		margin-bottom: 6px;
	}

	.metric-sub {
		font-size: 11px;
		opacity: 0.6;
		margin-top: 4px;
	}
}
</style>
