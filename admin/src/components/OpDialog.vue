<template>
	<n-modal
		:show="open"
		@update:show="v => (open = v)"
		:title="form.title"
		preset="dialog"
		:mask-closable="false"
		:close-on-esc="false"
		:style="form.sseUrl ? 'width: 70vw; max-width: 980px' : 'width: 400px'"
		@close="handleClose"
	>
		<template v-if="!form.sseUrl">
			<n-alert type="warning" :show-icon="true" class="mb-3">
				<div v-for="(item, index) in form.msgs" :key="index" style="line-height: 20px; word-wrap: break-word">
					<span>{{ item }}</span>
				</div>
			</n-alert>
			<slot name="content"></slot>
			<ul v-for="(item, index) in form.names" :key="index">
				<div style="word-wrap: break-word">
					<li>{{ item }}</li>
				</div>
			</ul>
		</template>
		<template v-else>
			<div style="display:flex; justify-content: space-between; align-items:center; margin-bottom: 12px;">
				<div style="font-size: 12px; color: #64748b;">{{ logStatusLabel }}</div>
			</div>
			<div
				ref="terminalRef"
				style="height: 55vh; overflow: auto; background: #0b1020; color: #e2e8f0; border-radius: 8px; padding: 12px; font-size: 12px; line-height: 18px;"
			>
				<div v-for="(line, index) in streamLogs" :key="index" style="white-space: pre-wrap; word-break: break-word;">
					{{ line }}
				</div>
			</div>
		</template>
		<template #action>
			<n-space justify="end">
				<n-button @click="open = false" :disabled="loading">
					{{ form.sseUrl ? $t("commons.button.close") : $t("commons.button.cancel") }}
				</n-button>
				<n-button v-if="!form.sseUrl" type="primary" @click="onConfirm" :loading="loading">
					{{ $t("commons.button.confirm") }}
				</n-button>
			</n-space>
		</template>
	</n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onUnmounted } from "vue"
import { NModal, NAlert, NButton, NSpace } from "naive-ui"
import { MsgError, MsgSuccess } from "../utils/message"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

const form = reactive({
	msgs: [] as string[],
	title: "",
	names: [] as string[],
	api: null as null | ((params: any) => Promise<any>),
	params: {},
	sseUrl: ""
})
const loading = ref(false)
const open = ref(false)
const successMsg = ref("")
const streamLogs = ref<string[]>([])
const logStatus = ref<"idle" | "connecting" | "streaming" | "success" | "failed">("idle")
const terminalRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

interface DialogProps {
	title: string
	msg?: string
	names?: Array<string>
	api?: ((params: any) => Promise<any>) | null
	params?: any
	successMsg?: string
	sseUrl?: string
}
const acceptParams = (props: DialogProps): void => {
	form.title = props.title
	form.names = props.names || []
	form.msgs = (props.msg || "").split("\n")
	form.api = (props.api as (params: any) => Promise<any>) || null
	form.params = props.params || {}
	successMsg.value = props.successMsg || ""
	form.sseUrl = props.sseUrl || ""
	streamLogs.value = []
	logStatus.value = "idle"
	open.value = true
	if (form.sseUrl) {
		startLogStream()
	}
}

const emit = defineEmits(["search", "cancel", "submit"])

const logStatusLabel = computed(() => {
	switch (logStatus.value) {
		case "connecting":
			return "连接中..."
		case "streaming":
			return "执行中..."
		case "success":
			return "已完成"
		case "failed":
			return "失败"
		default:
			return ""
	}
})

const scrollToBottom = () => {
	nextTick(() => {
		if (!terminalRef.value) return
		terminalRef.value.scrollTop = terminalRef.value.scrollHeight
	})
}

const stopLogStream = () => {
	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
}

const startLogStream = () => {
	stopLogStream()
	if (!form.sseUrl) return
	logStatus.value = "connecting"
	eventSource = new EventSource(form.sseUrl)

	eventSource.onmessage = event => {
		if (event.data === "ping" || event.data === ":" || event.data === "") return
		logStatus.value = logStatus.value === "connecting" ? "streaming" : logStatus.value
		streamLogs.value.push(event.data)
		scrollToBottom()
	}

	eventSource.addEventListener("status", event => {
		const data = (event as MessageEvent).data
		if (data === "success") {
			logStatus.value = "success"
		} else if (data === "failed") {
			logStatus.value = "failed"
		} else {
			logStatus.value = "streaming"
		}
	})

	eventSource.addEventListener("eof", () => {
		stopLogStream()
		if (logStatus.value === "success") {
			MsgSuccess(t("commons.msg.operationSuccess"))
		} else if (logStatus.value === "failed") {
			MsgError("操作失败")
		}
		emit("search")
	})

	eventSource.onerror = () => {
		streamLogs.value.push("连接已断开或发生错误")
		scrollToBottom()
		stopLogStream()
		logStatus.value = logStatus.value === "success" ? "success" : "failed"
	}
}

const onConfirm = async () => {
	if (!form.api) {
		emit("submit")
		open.value = false
		return
	}
	loading.value = true
	try {
		await form.api(form.params)
		emit("cancel")
		emit("search")
		MsgSuccess(successMsg.value || t("commons.msg.deleteSuccess"))
		open.value = false
	} catch (e) {
		// 错误处理
	} finally {
		loading.value = false
	}
}

const handleClose = () => {
	emit("cancel")
	stopLogStream()
	open.value = false
}

onUnmounted(() => {
	stopLogStream()
})

defineExpose({
	acceptParams
})
</script>
