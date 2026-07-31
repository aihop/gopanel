<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { MobileNode } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"

defineProps<{ node: MobileNode | null; online: boolean; cpuPercent: number; memoryPercent: number; load: number }>()
const { t } = useI18n({ messages: mobileMessages })
</script>

<template>
	<n-alert v-if="node && !online" type="warning" :title="t(`mobile.nodeStatus_${node.status}`)">
		<div>{{ t("mobile.nodeSnapshotUnavailable") }}</div>
		<div v-if="node.lastSeenAt" class="mt-1 text-xs">
			{{ t("mobile.lastSeen", { time: new Date(node.lastSeenAt).toLocaleString() }) }}
		</div>
	</n-alert>
	<div class="grid grid-cols-2 gap-3">
		<div class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="text-xs text-slate-500">{{ t("mobile.cpu") }}</div>
			<div class="mt-2 text-2xl font-bold">{{ online ? `${cpuPercent}%` : "—" }}</div>
			<n-progress v-if="online" type="line" :percentage="cpuPercent" :show-indicator="false" class="mt-3" />
		</div>
		<div class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="text-xs text-slate-500">{{ t("mobile.memory") }}</div>
			<div class="mt-2 text-2xl font-bold">{{ online ? `${memoryPercent}%` : "—" }}</div>
			<n-progress v-if="online" type="line" :percentage="memoryPercent" :show-indicator="false" class="mt-3" />
		</div>
		<div class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="text-xs text-slate-500">{{ t("mobile.load") }}</div>
			<div class="mt-2 text-2xl font-bold">{{ online ? load.toFixed(2) : "—" }}</div>
		</div>
		<div class="rounded-2xl bg-white p-4 shadow-sm">
			<div class="text-xs text-slate-500">{{ t("mobile.disk") }}</div>
			<div class="mt-2 text-2xl font-bold">
				{{ online ? `${Math.round(node?.summary.diskMaxPercent || 0)}%` : "—" }}
			</div>
		</div>
	</div>
	<section v-if="node && online" class="rounded-2xl bg-white p-4 shadow-sm">
		<div class="grid grid-cols-2 gap-4 text-sm">
			<div>
				<div class="text-xs text-slate-500">{{ t("mobile.runningContainers") }}</div>
				<div class="mt-1 font-semibold">
					{{ node.summary.containerRunning }}/{{ node.summary.containerTotal }}
				</div>
			</div>
			<div>
				<div class="text-xs text-slate-500">{{ t("mobile.certificates") }}</div>
				<div class="mt-1 font-semibold">
					{{ node.summary.certExpiringCount }}/{{ node.summary.certTotal }} {{ t("mobile.expiring") }}
				</div>
			</div>
		</div>
		<div v-if="node.warnings.length" class="mt-3 flex flex-wrap gap-2">
			<n-tag
				v-for="(warning, index) in node.warnings"
				:key="index"
				size="small"
				:type="warning.level === 'danger' ? 'error' : 'warning'"
			>
				{{ t(`mobile.warning_${warning.type}`) }}
			</n-tag>
		</div>
	</section>
</template>
