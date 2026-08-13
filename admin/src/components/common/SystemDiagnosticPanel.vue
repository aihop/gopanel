<template>
	<section class="diagnostic-panel">
		<header class="diagnostic-panel__header" @pointerdown="emit('drag-start', $event)">
			<div class="flex min-w-0 items-center gap-2">
				<span class="diagnostic-panel__icon"><Icon name="mdi:stethoscope" :size="20" /></span>
				<div class="min-w-0">
					<div class="truncate font-semibold">{{ t("systemDiagnostic.title") }}</div>
					<div class="mt-0.5 flex items-center gap-2 text-xs opacity-65">
						<span>{{ t("systemDiagnostic.readOnly") }}</span>
						<span v-if="state" :class="state.snapshot.controlPlane.healthy ? 'text-emerald-600' : 'text-orange-600'">
							{{ t(state.snapshot.controlPlane.healthy ? "systemDiagnostic.healthy" : "systemDiagnostic.unhealthy") }}
						</span>
					</div>
				</div>
			</div>
			<n-tooltip>
				<template #trigger>
					<n-button quaternary circle size="small" @pointerdown.stop @click="emit('close')">
						<template #icon><Icon name="mdi:close" :size="18" /></template>
					</n-button>
				</template>
				{{ t("systemDiagnostic.close") }}
			</n-tooltip>
		</header>

		<div v-if="loading" class="diagnostic-panel__center">
			<n-spin size="large" />
			<span class="text-sm opacity-65">{{ t("systemDiagnostic.loading") }}</span>
		</div>
		<div v-else-if="loadError" class="diagnostic-panel__center">
			<n-alert type="error" :show-icon="false">{{ t("systemDiagnostic.loadFailed") }}</n-alert>
			<n-button type="primary" @click="loadState">{{ t("systemDiagnostic.retry") }}</n-button>
		</div>
		<template v-else>
			<div class="diagnostic-panel__account">
				<n-select
					v-model:value="accountId"
					:options="accountOptions"
					:placeholder="t('systemDiagnostic.selectAI')"
					:disabled="sending || !accountOptions.length"
				/>
			</div>
			<n-alert v-if="!accountOptions.length" class="mx-3 mt-3" type="warning" :show-icon="false">
				{{ t("systemDiagnostic.noAccount") }}
			</n-alert>

			<div ref="messageList" class="diagnostic-panel__messages">
				<div v-if="!messages.length" class="diagnostic-panel__empty">
					<Icon name="mdi:database-search-outline" :size="42" class="opacity-35" />
					<div class="font-medium">{{ t("systemDiagnostic.emptyTitle") }}</div>
					<p class="m-0 max-w-[300px] text-center text-sm leading-6 opacity-60">
						{{ t("systemDiagnostic.emptyDescription") }}
					</p>
					<div class="mt-2 flex flex-wrap justify-center gap-2">
						<n-button v-for="item in quickQuestions" :key="item" size="tiny" secondary @click="useQuickQuestion(item)">
							{{ item }}
						</n-button>
					</div>
				</div>
				<article
					v-for="messageItem in messages"
					:key="messageItem.id || `${messageItem.role}-${messageItem.createdAt}`"
					class="diagnostic-message"
					:class="messageItem.role === 'user' ? 'is-user' : 'is-agent'"
				>
					<div class="diagnostic-message__content">
						{{ messageItem.content }}
						<span v-if="sending && messageItem === streamingMessage" class="diagnostic-message__cursor" />
					</div>
				</article>
			</div>

			<footer class="diagnostic-panel__composer">
				<n-alert v-if="sendError" type="error" :show-icon="false">{{ t("systemDiagnostic.sendFailed") }}</n-alert>
				<n-input
					v-model:value="content"
					type="textarea"
					:placeholder="t('systemDiagnostic.placeholder')"
					:autosize="{ minRows: 2, maxRows: 5 }"
					maxlength="4000"
					@keydown.ctrl.enter.prevent="send"
					@keydown.meta.enter.prevent="send"
				/>
				<div class="flex items-center justify-between gap-3">
					<span class="text-xs leading-5 opacity-50">{{ t("systemDiagnostic.privacy") }}</span>
					<n-button type="primary" :loading="sending" :disabled="!accountId || sending" @click="send">
						{{ sending ? t("systemDiagnostic.sending") : t("systemDiagnostic.send") }}
					</n-button>
				</div>
			</footer>
		</template>
	</section>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import { getSystemDiagnosticState, streamSystemDiagnostic } from "@/api/modules/systemDiagnostic"
import type { SystemDiagnosticMessage, SystemDiagnosticState } from "@/api/interface/systemDiagnostic"
import { systemDiagnosticMessages } from "@/i18n/locales/systemDiagnostic"
import { useMessage } from "naive-ui"
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

const emit = defineEmits<{ close: []; "drag-start": [event: PointerEvent] }>()
const { t } = useI18n({ messages: systemDiagnosticMessages })
const message = useMessage()
const state = ref<SystemDiagnosticState | null>(null)
const messages = ref<SystemDiagnosticMessage[]>([])
const loading = ref(true)
const sending = ref(false)
const loadError = ref(false)
const sendError = ref(false)
const content = ref("")
const accountId = ref<number | null>(null)
const messageList = ref<HTMLElement | null>(null)
const streamingMessage = ref<SystemDiagnosticMessage | null>(null)
let streamController: AbortController | null = null
const accountStorageKey = "gopanel_system_diagnostic_account"

const accountOptions = computed(() => (state.value?.accounts || []).map(account => ({
	label: `${account.name} · ${account.model}`,
	value: account.id
})))
const quickQuestions = computed(() => [
	t("systemDiagnostic.quickBackup"),
	t("systemDiagnostic.quickControlPlane"),
	t("systemDiagnostic.quickFailures")
])

async function loadState() {
	loading.value = true
	loadError.value = false
	try {
		const response = await getSystemDiagnosticState()
		state.value = response.data
		messages.value = response.data.messages || []
		const saved = Number(localStorage.getItem(accountStorageKey))
		const selected = response.data.accounts.find(account => account.id === saved) || response.data.accounts[0]
		accountId.value = selected?.id || null
		await scrollToBottom()
	} catch {
		loadError.value = true
	} finally {
		loading.value = false
	}
}

async function send() {
	const question = content.value.trim()
	if (!question) {
		message.warning(t("systemDiagnostic.inputRequired"))
		return
	}
	if (!accountId.value || sending.value) return
	const createdAt = new Date().toISOString()
	const temporaryID = -Date.now()
	const userMessage: SystemDiagnosticMessage = { id: temporaryID, role: "user", content: question, createdAt }
	const assistantMessage: SystemDiagnosticMessage = { id: temporaryID - 1, role: "agent", content: "", createdAt }
	let streamStarted = false
	messages.value.push(userMessage, assistantMessage)
	streamingMessage.value = assistantMessage
	content.value = ""
	sending.value = true
	sendError.value = false
	const controller = new AbortController()
	streamController = controller
	await scrollToBottom()
	try {
		await streamSystemDiagnostic(question, accountId.value, {
			onStart: value => {
				streamStarted = true
				Object.assign(userMessage, value.userMessage)
			},
			onDelta: delta => {
				assistantMessage.content += delta
				void scrollToBottom()
			},
			onDone: value => {
				Object.assign(userMessage, value.userMessage)
				Object.assign(assistantMessage, value.assistantMessage)
			}
		}, controller.signal)
	} catch {
		if (!streamStarted) messages.value = messages.value.filter(item => item !== userMessage && item !== assistantMessage)
		if (!controller.signal.aborted) sendError.value = true
	} finally {
		sending.value = false
		streamingMessage.value = null
		streamController = null
	}
}

function useQuickQuestion(question: string) {
	content.value = question
}

async function scrollToBottom() {
	await nextTick()
	if (messageList.value) messageList.value.scrollTop = messageList.value.scrollHeight
}

watch(accountId, value => {
	if (value) localStorage.setItem(accountStorageKey, String(value))
})

onMounted(loadState)
onBeforeUnmount(() => streamController?.abort())
</script>

<style scoped>
.diagnostic-panel { display: flex; height: min(720px, calc(100svh - 32px)); width: min(440px, calc(100vw - 32px)); flex-direction: column; overflow: hidden; border: 1px solid rgb(59 130 246 / 24%); border-radius: 18px; background: var(--n-color, var(--bg-default-color)); box-shadow: 0 24px 60px rgb(15 23 42 / 25%); pointer-events: auto; }
.diagnostic-panel__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 12px 12px 14px; cursor: ns-resize; user-select: none; touch-action: none; border-bottom: 1px solid rgb(59 130 246 / 18%); background: linear-gradient(135deg, rgb(219 234 254 / 92%), rgb(238 242 255 / 92%)); color: #1e3a8a; }
.diagnostic-panel__icon { display: grid; width: 34px; height: 34px; place-items: center; border-radius: 10px; background: rgb(37 99 235 / 12%); }
.diagnostic-panel__account { padding: 12px 12px 0; }
.diagnostic-panel__center { display: flex; min-height: 280px; flex: 1; flex-direction: column; align-items: center; justify-content: center; gap: 14px; padding: 24px; }
.diagnostic-panel__messages { min-height: 0; flex: 1; overflow-y: auto; padding: 14px 12px; }
.diagnostic-panel__empty { display: flex; min-height: 100%; flex-direction: column; align-items: center; justify-content: center; gap: 8px; padding: 24px 8px; }
.diagnostic-message { display: flex; margin-bottom: 12px; }
.diagnostic-message.is-user { justify-content: flex-end; }
.diagnostic-message__content { max-width: 88%; white-space: pre-wrap; overflow-wrap: anywhere; border-radius: 14px; padding: 10px 12px; font-size: 14px; line-height: 1.65; }
.diagnostic-message.is-user .diagnostic-message__content { border-bottom-right-radius: 4px; background: #2563eb; color: white; }
.diagnostic-message.is-agent .diagnostic-message__content { border-bottom-left-radius: 4px; background: rgb(148 163 184 / 14%); }
.diagnostic-message__cursor { display: inline-block; width: 2px; height: 1em; margin-left: 2px; vertical-align: -2px; background: currentColor; animation: diagnostic-cursor 0.9s steps(1) infinite; }
.diagnostic-panel__composer { display: flex; flex-direction: column; gap: 9px; padding: 12px; border-top: 1px solid var(--border-color); }
@keyframes diagnostic-cursor { 50% { opacity: 0; } }
</style>
