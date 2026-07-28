<script setup lang="ts">
import type { RemoteContainer } from "@/api/modules/nodeProxy"
import { remoteContainerListAPI } from "@/api/modules/nodeProxy"
import { t } from "@/i18n"
import { useAuthStore } from "@/store/auth"
import { useMessage } from "naive-ui"
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue"

const props = defineProps<{
	nodeId: number
	nodeName: string
}>()

const message = useMessage()
const authStore = useAuthStore()

const loading = ref(false)
const containers = ref<RemoteContainer[]>([])
const selected = ref<string | null>(null)
const tail = ref(200)
const follow = ref(true)
const logText = ref("")
const connected = ref(false)
const boxRef = ref<HTMLElement | null>(null)

let socket: WebSocket | null = null

async function fetchContainers() {
	loading.value = true
	try {
		const res = await remoteContainerListAPI(props.nodeId, { page: 1, limit: 200 })
		containers.value = res.data?.items || []
	} catch {
		containers.value = []
	} finally {
		loading.value = false
	}
}

function close() {
	if (socket) {
		// 与本机日志页一致：先发 close conn 让节点侧结束 follow，再关连接
		try {
			socket.send("close conn")
		} catch {
			// 连接已经断了，忽略
		}
		socket.close()
		socket = null
	}
	connected.value = false
}

function open() {
	if (!selected.value) {
		message.warning(t("node.workspace.pickContainerLog"))
		return
	}
	close()
	logText.value = ""

	const protocol = window.location.protocol === "https:" ? "wss" : "ws"
	const auth = authStore.getAuth() || ""
	// endpoint 走主控的 ws 代理；container/since/tail/follow 会被原样透传给节点
	const url =
		`${protocol}://${window.location.host}/api/node-proxy-ws/${props.nodeId}/container/logs` +
		`?container=${encodeURIComponent(selected.value)}&since=all&tail=${tail.value}` +
		`&follow=${follow.value}&runtimeHost=&token=${encodeURIComponent(auth)}`

	socket = new WebSocket(url)
	connected.value = true
	socket.onmessage = event => {
		// 去掉 ANSI 转义，否则日志里会出现乱码控制符
		logText.value += String(event.data).replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, "")
		nextTick(() => {
			if (boxRef.value) {
				boxRef.value.scrollTop = boxRef.value.scrollHeight
			}
		})
	}
	socket.onclose = () => {
		connected.value = false
	}
	socket.onerror = () => {
		connected.value = false
		message.error(t("node.workspace.logFailed"))
	}
}

onMounted(fetchContainers)
onBeforeUnmount(close)
</script>

<template>
	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center gap-2">
			<n-select
				v-model:value="selected"
				class="max-w-64"
				size="small"
				filterable
				:loading="loading"
				:placeholder="t('node.workspace.pickContainerLog')"
				:options="containers.map(item => ({ label: `${item.name} (${item.state})`, value: item.name }))"
			/>
			<n-input-number v-model:value="tail" size="small" class="w-28" :min="10" :max="5000" />
			<n-checkbox v-model:checked="follow">{{ t("node.workspace.follow") }}</n-checkbox>
			<n-button size="small" type="primary" @click="open">{{ t("commons.button.view") }}</n-button>
			<n-button v-if="connected" size="small" @click="close">{{ t("commons.button.stop") }}</n-button>
			<n-button size="small" :loading="loading" @click="fetchContainers">
				{{ t("commons.button.refresh") }}
			</n-button>
		</div>

		<div v-if="logText" ref="boxRef" class="log-box">{{ logText }}</div>
		<n-empty v-else :description="t('node.workspace.logHint')" />
	</div>
</template>

<style lang="scss" scoped>
.log-box {
	height: 420px;
	overflow: auto;
	padding: 8px 10px;
	border: 1px solid var(--border-color);
	border-radius: var(--border-radius);
	background-color: var(--bg-secondary-color);
	font-family: var(--font-family-mono);
	font-size: 12px;
	line-height: 1.5;
	white-space: pre-wrap;
	word-break: break-all;
}
</style>
