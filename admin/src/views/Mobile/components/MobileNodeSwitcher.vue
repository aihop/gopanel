<script setup lang="ts">
import type { MobileNode } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"
import { useI18n } from "vue-i18n"

defineProps<{
	nodes: MobileNode[]
	selectedId: number
	loading: boolean
}>()

const emit = defineEmits<{
	select: [node: MobileNode]
}>()

const show = defineModel<boolean>("show", { required: true })
const { t } = useI18n({ messages: mobileMessages })

function statusText(status: MobileNode["status"]): string {
	return t(`mobile.nodeStatus_${status}`)
}

function statusType(status: MobileNode["status"]): "success" | "error" | "warning" | "default" {
	if (status === "online") return "success"
	if (status === "unauthorized") return "warning"
	if (status === "offline") return "error"
	return "default"
}

function selectNode(node: MobileNode) {
	emit("select", node)
	show.value = false
}
</script>

<template>
	<n-drawer v-model:show="show" placement="bottom" height="70dvh">
		<n-drawer-content :title="t('mobile.selectNode')" closable body-content-style="padding: 12px 16px 24px;">
			<n-spin :show="loading">
				<n-empty v-if="!nodes.length" :description="t('mobile.noNodes')" class="py-12" />
				<div v-else class="space-y-3">
					<button
						v-for="node in nodes"
						:key="node.id"
						type="button"
						class="w-full rounded-2xl border bg-white p-4 text-left transition-colors"
						:class="node.id === selectedId ? 'border-blue-500 ring-2 ring-blue-100' : 'border-slate-200'"
						@click="selectNode(node)"
					>
						<div class="flex items-center justify-between gap-3">
							<div class="min-w-0">
								<div class="flex items-center gap-2">
									<span class="truncate font-semibold">{{ node.name }}</span>
									<n-tag v-if="node.isLocal" size="tiny" :bordered="false">{{ t("mobile.controllerNode") }}</n-tag>
									<n-tag v-if="node.isProd" size="tiny" type="error" :bordered="false">{{ t("mobile.productionNode") }}</n-tag>
								</div>
								<div class="mt-1 truncate text-xs text-slate-500">{{ node.summary.hostname || node.version || t("mobile.nodeDetailsUnavailable") }}</div>
							</div>
							<n-tag size="small" :type="statusType(node.status)" :bordered="false">{{ statusText(node.status) }}</n-tag>
						</div>
						<div v-if="node.status === 'online'" class="mt-3 grid grid-cols-3 gap-2 text-xs text-slate-500">
							<span>{{ t("mobile.cpu") }} {{ node.summary.cpuPercent.toFixed(0) }}%</span>
							<span>{{ t("mobile.memory") }} {{ node.summary.memPercent.toFixed(0) }}%</span>
							<span>{{ t("mobile.warnings") }} {{ node.warnings.length }}</span>
						</div>
						<div v-else class="mt-3 text-xs text-slate-500">{{ statusText(node.status) }}</div>
					</button>
				</div>
			</n-spin>
		</n-drawer-content>
	</n-drawer>
</template>
