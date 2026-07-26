<script setup lang="ts">
import type { RemoteContainer } from "@/api/modules/nodeProxy"
import { remoteContainerListAPI } from "@/api/modules/nodeProxy"
import Terminal from "@/components/Terminal.vue"
import { t } from "@/i18n"
import { useMessage } from "naive-ui"
import { nextTick, onMounted, ref } from "vue"

const props = defineProps<{
	nodeId: number
	nodeName: string
	canControl: boolean
}>()

const message = useMessage()

const loading = ref(false)
const containers = ref<RemoteContainer[]>([])
const selected = ref<string | null>(null)
const user = ref("")
const command = ref("/bin/sh")
const opened = ref(false)
const terminalRef = ref<InstanceType<typeof Terminal> | null>(null)

async function fetchContainers() {
	loading.value = true
	try {
		const res = await remoteContainerListAPI(props.nodeId, { page: 1, limit: 200 })
		// 只有运行中的容器能进终端
		containers.value = (res.data?.items || []).filter(item => item.state === "running")
	} catch {
		containers.value = []
	} finally {
		loading.value = false
	}
}

async function open() {
	if (!selected.value) {
		message.warning(t("node.workspace.pickContainer"))
		return
	}
	opened.value = true
	await nextTick()
	// 关键：endpoint 指向主控的 ws 代理，其余参数与本机终端完全一致。
	// 主控会剥掉 auth 后把查询串原样转给节点，所以这里不需要为远程做任何特殊处理。
	terminalRef.value?.acceptParams({
		endpoint: `/node-proxy-ws/${props.nodeId}/container/exec`,
		args: `source=container&containerid=${selected.value}&user=${user.value}&command=${command.value}&runtimeHost=`,
		error: "",
		initCmd: ""
	})
}

function close() {
	terminalRef.value?.onClose()
	opened.value = false
}

onMounted(fetchContainers)
</script>

<template>
	<div class="flex flex-col gap-3">
		<n-alert v-if="!canControl" type="warning" :show-icon="false" class="text-xs">
			{{ t("node.workspace.noControl") }}
		</n-alert>

		<template v-else>
			<n-alert type="warning" :show-icon="false" class="text-xs">
				{{ t("node.workspace.terminalWarn", { node: nodeName }) }}
			</n-alert>

			<div class="flex flex-wrap items-center gap-2">
				<n-select
					v-model:value="selected"
					class="max-w-64"
					size="small"
					filterable
					:loading="loading"
					:placeholder="t('node.workspace.pickContainer')"
					:options="containers.map(item => ({ label: item.name, value: item.name }))"
				/>
				<n-input v-model:value="command" size="small" class="max-w-40" placeholder="/bin/sh" />
				<n-button size="small" type="primary" :disabled="opened" @click="open">
					{{ t("commons.button.conn") }}
				</n-button>
				<n-button v-if="opened" size="small" @click="close">{{ t("commons.button.disconnect") }}</n-button>
				<n-button size="small" :loading="loading" @click="fetchContainers">
					{{ t("commons.button.refresh") }}
				</n-button>
			</div>

			<div v-if="opened" class="terminal-box">
				<Terminal ref="terminalRef" />
			</div>
			<n-empty v-else :description="t('node.workspace.terminalHint')" />
		</template>
	</div>
</template>

<style lang="scss" scoped>
.terminal-box {
	height: 420px;
	border: 1px solid var(--border-color);
	border-radius: var(--border-radius);
	overflow: hidden;
}
</style>
