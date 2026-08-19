import { computed, type Ref } from "vue"
import type { MobileNode, MobileOverview } from "@/api/modules/mobile"

export function useMobileNodeMetrics(
	nodes: Ref<MobileNode[]>,
	selectedNodeId: Ref<number>,
	overview: Ref<MobileOverview | null>
) {
	const selectedNode = computed(
		() => nodes.value.find(item => item.id === selectedNodeId.value) || nodes.value[0] || null
	)
	const memoryPercent = computed(() =>
		Math.round(
			selectedNode.value?.isLocal
				? overview.value?.system.memoryUsedPercent || 0
				: selectedNode.value?.summary.memPercent || 0
		)
	)
	const cpuPercent = computed(() =>
		Math.round(
			selectedNode.value?.isLocal
				? overview.value?.system.cpuUsedPercent || 0
				: selectedNode.value?.summary.cpuPercent || 0
		)
	)
	const load1 = computed(() =>
		selectedNode.value?.isLocal ? overview.value?.system.load1 || 0 : selectedNode.value?.summary.load1 || 0
	)
	const nodeIsOnline = computed(() => selectedNode.value?.status === "online")
	const nodeCanOperate = computed(() => Boolean(selectedNode.value?.isLocal || selectedNode.value?.hasControlToken))
	return { selectedNode, memoryPercent, cpuPercent, load1, nodeIsOnline, nodeCanOperate }
}
