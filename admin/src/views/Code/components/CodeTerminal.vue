<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick, watch } from "vue"
import { Terminal } from "@xterm/xterm"
import "@xterm/xterm/css/xterm.css"
import { useCodexRuntimeState } from "../useCodexRuntimeState"
import { codeTerminalMessages } from "../codeTerminalMessages"
import CodeTerminalStatusBar from "./CodeTerminalStatusBar.vue"
import CodeTerminalScrollToBottom from "./CodeTerminalScrollToBottom.vue"
import CodeTerminalDeliveredNotice from "./CodeTerminalDeliveredNotice.vue"
import { useCodeTerminalConnection } from "./useCodeTerminalConnection"
import { useCodeTerminalScrollAnchor } from "./useCodeTerminalScrollAnchor"

const props = defineProps<{
	taskId: number | null
	sessionId?: number | null
	autoTakeControl?: boolean
	reserveTopRightActions?: boolean
}>()
const emit = defineEmits<{
	(e: "task-created", taskId: number): void
	(e: "new-session"): void
}>()
const terminalRef = ref<HTMLElement | null>(null)
let term: Terminal
let activatedOnce = false

const { scrolledUp, syncScrollAnchor, jumpToTerminalBottom } = useCodeTerminalScrollAnchor(() => term)
const {
	runtimeState,
	runtimeError,
	runtimeSupported,
	executorId,
	loadRuntimeState,
	startRuntimePolling,
	stopRuntimePolling,
	disableRuntimeState,
} = useCodexRuntimeState(() => props.sessionId, ref(true))

const terminalConnection = useCodeTerminalConnection({
	sessionId: computed(() => props.sessionId),
	taskId: computed(() => props.taskId),
	autoTakeControl: computed(() => props.autoTakeControl),
	onTaskCreated: (taskId) => emit("task-created", taskId),
	onWriteTerminalData: (data, forceBottom) => {
		const buffer = term.buffer.active
		const shouldFollow = forceBottom || buffer.baseY - buffer.viewportY <= 1
		term.write(data, () => {
			if (shouldFollow) term.scrollToBottom()
			syncScrollAnchor()
		})
	},
	onSyncScrollAnchor: syncScrollAnchor,
	onJumpToTerminalBottom: jumpToTerminalBottom,
})

const {
	isActive,
	nativeProtocol,
	hasTerminalControl,
	reconnecting,
	connectionFailed,
	terminalInactive,
	sessionDelivered,
	initTerminal,
	reconnectTerminal,
	resumeTerminal,
	takeTerminalControl,
	disconnectTerminal,
	handleResize,
	cleanup,
	updateAutoTakeControl,
	activate,
	deactivate,
	setupResizeObserver,
	refreshDeliveredSession,
} = terminalConnection

const initializeSessionTerminal = async () => {
	if (await refreshDeliveredSession()) return
	initTerminal(terminalRef)
	startRuntimePolling()
}

onMounted(() => {
	void initializeSessionTerminal()
	window.addEventListener("resize", handleResize)
	setupResizeObserver(terminalRef)
})

watch(
	() => props.autoTakeControl,
	requested => {
		updateAutoTakeControl(requested)
	}
)

watch(
	() => props.taskId,
	(taskId, previousTaskId) => {
		if (taskId === previousTaskId) return
		void nextTick(() => {
			if (!isActive.value || sessionDelivered.value || !term) return
			jumpToTerminalBottom()
		})
	}
)

onActivated(() => {
	activate()
	if (!activatedOnce) {
		activatedOnce = true
		return
	}
	startRuntimePolling()
	void nextTick(() => {
		handleResize()
		void loadRuntimeState()
	})
})

onDeactivated(() => {
	deactivate()
	stopRuntimePolling()
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	stopRuntimePolling()
	cleanup()
})
</script>

<template>
	<div
		class="flex h-full min-h-0 w-full flex-col bg-[#1e1e1e]"
		:class="{ 'code-terminal--reserved-actions': reserveTopRightActions }"
	>
		<CodeTerminalStatusBar
			v-if="sessionId && runtimeSupported"
			:runtime-state="runtimeState"
			:runtime-error="runtimeError"
			:native-protocol="nativeProtocol"
			:has-control="hasTerminalControl"
			:reconnecting="reconnecting"
			:connection-failed="connectionFailed"
			:terminal-inactive="terminalInactive"
			:reserve-top-right-actions="reserveTopRightActions"
			@reconnect="reconnectTerminal"
			@resume="resumeTerminal"
			@take-control="takeTerminalControl"
		/>
		<CodeTerminalDeliveredNotice v-if="sessionDelivered" @new-session="emit('new-session')" />
		<div v-else class="relative min-h-0 w-full flex-1">
			<div ref="terminalRef" class="h-full w-full" />
			<CodeTerminalScrollToBottom :visible="scrolledUp" @jump="jumpToTerminalBottom" />
		</div>
	</div>
</template>

<style scoped>
:deep(.xterm),
:deep(.xterm-screen),
:deep(.xterm-helpers),
:deep(.xterm-viewport) {
	height: 100%;
}

:deep(.xterm) {
	padding: 16px 18px;
}

.code-terminal--reserved-actions :deep(.xterm) {
	padding-right: 5rem;
}

:deep(.xterm-viewport) {
	overflow-y: auto !important;
}
:deep(.xterm-viewport::-webkit-scrollbar) {
	width: 8px;
}
:deep(.xterm-viewport::-webkit-scrollbar-track) {
	background: #1e1e1e;
}
:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
	background: #424242;
	border-radius: 4px;
}
</style>
