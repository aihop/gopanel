<template>
  <div
    ref="terminalElement"
    @wheel="onTermWheel"
  ></div>
</template>

<script lang="ts" setup>
import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import { useAuthStore } from "@/store/auth"
import { enc } from "crypto-js"
import { nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import "@xterm/xterm/css/xterm.css"

const { t } = useI18n()
const terminalElement = ref<HTMLDivElement | null>(null)
const fitAddon = new FitAddon()
const termReady = ref(false)
const webSocketReady = ref(false)
const term = ref()
const terminalSocket = ref<WebSocket>()
const heartbeatTimer = ref<NodeJS.Timeout>()
const latency = ref(0)
const initCmd = ref("")

const readyWatcher = watch(
	() => webSocketReady.value && termReady.value,
	ready => {
		if (ready) {
			changeTerminalSize()
			readyWatcher() // unwatch self
		}
	}
)

interface WsProps {
	endpoint: string
	args: string
	error: string
	initCmd: string
}
function acceptParams(props: WsProps) {
	nextTick(() => {
		if (props.error.length !== 0) {
			initError(props.error)
		} else {
			initCmd.value = props.initCmd || ""
			init(props.endpoint, props.args)
		}
	})
}

function newTerm() {
	const background = getComputedStyle(document.documentElement).getPropertyValue("--panel-terminal-bg-color").trim()
	term.value = new Terminal({
		lineHeight: 1.2,
		fontSize: 12,
		fontFamily: "Monaco, Menlo, Consolas, 'Courier New', monospace",
		theme: {
			background
		},
		cursorBlink: true,
		cursorStyle: "underline",
		scrollback: 1000,
		scrollSensitivity: 15,
		tabStopWidth: 4
	})
}

function init(endpoint: string, args: string) {
	if (initTerminal(true)) {
		initWebSocket(endpoint, args)
	}
}

function initError(errorInfo: string) {
	if (initTerminal(false)) {
		term.value.write(errorInfo)
	}
}

function onClose(isKeepShow: boolean = false) {
	window.removeEventListener("resize", changeTerminalSize)
	try {
		terminalSocket.value?.close()
	} catch {}
	if (!isKeepShow) {
		try {
			term.value.dispose()
		} catch {}
	}
	if (terminalElement.value != null) {
		terminalElement.value.innerHTML = ""
	}
}

// terminal 相关代码 start

function initTerminal(online: boolean = false): boolean {
	newTerm()
	if (terminalElement.value) {
		term.value.open(terminalElement.value)
		term.value.loadAddon(fitAddon)
		window.addEventListener("resize", changeTerminalSize)
		if (online) {
			term.value.onData((data: any) => sendMsg(data))
		}
		termReady.value = true
	}
	return termReady.value
}

function changeTerminalSize() {
	fitAddon.fit()
	if (isWsOpen()) {
		const { cols, rows } = term.value
		terminalSocket.value!.send(
			JSON.stringify({
				type: "resize",
				cols,
				rows
			})
		)
	}
}

/**
 * Support for Ctrl+MouseWheel to scaling fonts
 * @param event WheelEvent
 */
function onTermWheel(event: WheelEvent) {
	if (event.ctrlKey) {
		event.preventDefault()
		if (event.deltaY > 0) {
			// web font-size mini 12px
			if (term.value.options.fontSize > 12) {
				term.value.options.fontSize = term.value.options.fontSize - 1
			}
		} else {
			term.value.options.fontSize = term.value.options.fontSize + 1
		}
		changeTerminalSize()
	}
}

// terminal 相关代码 end

// websocket 相关代码 start

function initWebSocket(endpoint_: string, args: string = "") {
	const href = window.location.href
	const protocol = href.split("//")[0] === "http:" ? "ws" : "wss"
	const host = href.split("//")[1].split("/")[0]
	const endpoint = endpoint_.replace(/^\/+/, "")

	const authStore = useAuthStore()
	const auth = authStore.getAuth() || ""
	const url = `${protocol}://${host}/api/${endpoint}?cols=${term.value.cols}&rows=${term.value.rows}&${args}&token=${encodeURIComponent(auth)}`
	terminalSocket.value = new WebSocket(url)
	terminalSocket.value.onopen = runRealTerminal
	terminalSocket.value.onmessage = onWSReceive
	terminalSocket.value.onclose = closeRealTerminal
	terminalSocket.value.onerror = errorRealTerminal
	heartbeatTimer.value = setInterval(() => {
		if (isWsOpen()) {
			terminalSocket.value!.send(
				JSON.stringify({
					type: "heartbeat",
					timestamp: `${new Date().getTime()}`
				})
			)
		}
	}, 1000 * 10)
}

function runRealTerminal() {
	webSocketReady.value = true
	if (initCmd.value !== "") {
		sendMsg(initCmd.value)
	}
}

function onWSReceive(message: MessageEvent) {
	const wsMsg = JSON.parse(message.data)
	switch (wsMsg.type) {
		case "cmd": {
			term.value.element && term.value.focus()
			if (wsMsg.data) {
				let receiveMsg = enc.Base64.parse(wsMsg.data).toString(enc.Utf8)
				if (initCmd.value !== "") {
					receiveMsg = receiveMsg?.replace(initCmd.value.trim(), "").trim()
					initCmd.value = ""
				}
				term.value.write(receiveMsg)
			}
			break
		}
		case "heartbeat": {
			latency.value = new Date().getTime() - wsMsg.timestamp
			break
		}
	}
}

function errorRealTerminal(ex: any) {
	let message = ex.message
	if (!message) message = t("terminal.disconnected")
	term.value.write(`\x1B[31m${message}\x1B[m\r\n`)
}

function closeRealTerminal(ev: CloseEvent) {
	if (heartbeatTimer.value) {
		clearInterval(heartbeatTimer.value)
	}
	term.value.write(`\r\n${ev.reason || t("terminal.disconnected")}`)
}

function isWsOpen() {
	const readyState = terminalSocket.value && terminalSocket.value.readyState
	return readyState === 1
}

function sendMsg(data: string) {
	if (isWsOpen()) {
		terminalSocket.value!.send(
			JSON.stringify({
				type: "cmd",
				data: enc.Base64.stringify(enc.Utf8.parse(data))
			})
		)
	}
}

// websocket 相关代码 end

defineExpose({
	acceptParams,
	onClose,
	isWsOpen,
	sendMsg,
	getLatency: () => latency.value
})

onBeforeUnmount(() => {
	onClose()
})
</script>

<style lang="scss" scoped>
#terminal {
	width: 100%;
	height: 100%;
}
:deep(.xterm) {
	padding: 5px !important;
}
</style>
