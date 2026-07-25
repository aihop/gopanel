<template>
	<!--
		抽屉是浮层：只在用户主动打开的几秒里覆盖右侧内容。
		这几秒用户正在选节点，本来不会去点表格右侧的操作列，所以覆盖不构成干扰；
		换来的是布局宽度恒定，终端不需要 refit。
	-->
	<n-drawer v-model:show="visible" :width="300" placement="right" :trap-focus="false">
		<n-drawer-content :title="t('node.drawer.title')" closable body-content-style="padding: 12px;">
			<template #header>
				<div class="flex items-center justify-between gap-2">
					<span>{{ t("node.drawer.title") }}</span>
					<div class="flex items-center gap-1">
						<n-button quaternary size="tiny" :loading="nodeStore.loading" @click="nodeStore.refreshNow()">
							<template #icon>
								<Icon name="mdi:refresh" :size="15" />
							</template>
						</n-button>
						<n-button quaternary size="tiny" @click="goManage">
							<template #icon>
								<Icon name="mdi:cog-outline" :size="15" />
							</template>
						</n-button>
					</div>
				</div>
			</template>

			<n-alert v-if="nodeStore.error" type="error" :show-icon="false" class="mb-2">
				{{ nodeStore.error }}
			</n-alert>

			<n-spin :show="nodeStore.loading && !nodeStore.loaded">
				<n-empty v-if="nodeStore.loaded && !nodeStore.list.length" :description="t('node.drawer.empty')">
					<template #extra>
						<n-button size="small" @click="goManage">{{ t("node.drawer.addNode") }}</n-button>
					</template>
				</n-empty>

				<div v-else class="flex flex-col gap-2">
					<div
						v-for="node of nodeStore.list"
						:key="node.id"
						class="node-card rounded p-2"
						:class="{ 'is-focus': node.id === nodeStore.focusId }"
					>
						<div class="mb-1 flex items-center justify-between gap-2">
							<div class="flex min-w-0 items-center gap-1">
								<Icon name="mdi:circle" :size="9" :class="levelDotClass[nodeLevel(node)]" />
								<span class="truncate text-sm font-medium">{{ node.name }}</span>
								<n-tag v-if="node.isProd" size="tiny" type="error" :bordered="false">
									{{ t("node.prod") }}
								</n-tag>
							</div>
							<n-tag size="tiny" :type="levelTagType[nodeLevel(node)]" :bordered="false">
								{{ statusText(node.status) }}
							</n-tag>
						</div>

						<!-- 离线/未授权时没有可信摘要，只展示原因，不显示上一次的水位读数免得被误读成当前值 -->
						<template v-if="node.status === 'online'">
							<div class="metric">
								<span class="metric-label">{{ t("node.metric.cpu") }}</span>
								<n-progress
									type="line"
									:percentage="clamp(node.summary.cpuPercent)"
									:color="usageColor(node.summary.cpuPercent)"
									:height="5"
									:show-indicator="false"
								/>
								<span class="metric-value">{{ node.summary.cpuPercent.toFixed(0) }}%</span>
							</div>
							<div class="metric">
								<span class="metric-label">{{ t("node.metric.memory") }}</span>
								<n-progress
									type="line"
									:percentage="clamp(node.summary.memPercent)"
									:color="usageColor(node.summary.memPercent)"
									:height="5"
									:show-indicator="false"
								/>
								<span class="metric-value">{{ node.summary.memPercent.toFixed(0) }}%</span>
							</div>
							<div v-if="node.summary.diskMaxPercent > 0" class="metric">
								<span class="metric-label">{{ t("node.metric.disk") }}</span>
								<n-progress
									type="line"
									:percentage="clamp(node.summary.diskMaxPercent)"
									:color="usageColor(node.summary.diskMaxPercent)"
									:height="5"
									:show-indicator="false"
								/>
								<span class="metric-value">{{ node.summary.diskMaxPercent.toFixed(0) }}%</span>
							</div>

							<div class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs opacity-70">
								<span>{{ t("node.metric.container") }} {{ node.summary.containerRunning }}/{{ node.summary.containerTotal }}</span>
								<span v-if="node.summary.certTotal > 0">
									{{ t("node.metric.cert") }} {{ node.summary.certTotal }}
								</span>
								<span v-if="node.version">{{ node.version }}</span>
							</div>
						</template>

						<div v-else class="text-xs opacity-70">
							<div v-if="node.statusMsg" class="break-all">{{ node.statusMsg }}</div>
							<div v-if="node.lastSeenAt && !isZeroTime(node.lastSeenAt)">
								{{ t("node.lastSeen", { time: formatTime(node.lastSeenAt) }) }}
							</div>
						</div>

						<div v-if="node.warnings?.length" class="mt-1 flex flex-col gap-0.5">
							<div
								v-for="(warning, index) of node.warnings"
								:key="index"
								class="flex items-center gap-1 text-xs"
								:class="warning.level === 'danger' ? 'text-red-500' : 'text-amber-500'"
							>
								<Icon name="mdi:alert-circle-outline" :size="12" />
								<span>{{ warningText(warning) }}</span>
							</div>
						</div>
					</div>
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { t } from "@/i18n"
import NodeStore from "@/store/modules/node"
import dayjs from "@/utils/dayjs"
import { computed } from "vue"
import { useRouter } from "vue-router"
import { levelDotClass, levelTagType, nodeLevel, statusText, usageColor, warningText } from "./nodeDisplay"

const nodeStore = NodeStore()
const router = useRouter()

const visible = computed({
	get: () => nodeStore.drawerVisible,
	set: (value: boolean) => {
		if (value) {
			nodeStore.openDrawer()
		} else {
			nodeStore.closeDrawer()
		}
	}
})

function clamp(percent: number): number {
	if (!percent || percent < 0) return 0
	return percent > 100 ? 100 : percent
}

/** Go 的零值时间序列化成 0001-01-01，直接格式化会显示成一个荒谬的年份 */
function isZeroTime(value: string): boolean {
	return !value || value.startsWith("0001-01-01")
}

function formatTime(value: string): string {
	return dayjs(value).format("MM-DD HH:mm")
}

function goManage() {
	nodeStore.closeDrawer()
	router.push({ name: "Node-Index" })
}
</script>

<style lang="scss" scoped>
.node-card {
	border: 1px solid var(--border-color);

	&.is-focus {
		border-color: var(--primary-color);
	}
}

.metric {
	display: flex;
	align-items: center;
	gap: 6px;
	margin-top: 3px;

	.metric-label {
		font-size: 11px;
		opacity: 0.7;
		width: 28px;
		flex: 0 0 28px;
	}

	.metric-value {
		font-size: 11px;
		opacity: 0.8;
		width: 32px;
		flex: 0 0 32px;
		text-align: right;
	}
}
</style>
