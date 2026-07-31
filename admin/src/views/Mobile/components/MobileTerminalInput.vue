<script setup lang="ts">
import { nextTick, ref } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"

defineProps<{ connected: boolean; hasControl: boolean; ctrlActive: boolean }>()
const emit = defineEmits<{
	takeControl: []
	releaseControl: []
	send: [data: string]
	shortcut: [data: string]
	toggleCtrl: []
}>()
const { t } = useI18n({ messages: mobileMessages })
const commandInput = ref<HTMLInputElement | null>(null)
const commandDraft = ref("")
const composing = ref(false)

function insertSymbol(symbol: string) {
	const input = commandInput.value
	const start = input?.selectionStart ?? commandDraft.value.length
	const end = input?.selectionEnd ?? start
	commandDraft.value = `${commandDraft.value.slice(0, start)}${symbol}${commandDraft.value.slice(end)}`
	nextTick(() => {
		commandInput.value?.focus()
		commandInput.value?.setSelectionRange(start + symbol.length, start + symbol.length)
	})
}

function submit() {
	if (composing.value || !commandDraft.value) return
	emit("send", `${commandDraft.value}\r`)
	commandDraft.value = ""
	nextTick(() => commandInput.value?.focus())
}

function toggleControl(hasControl: boolean) {
	if (hasControl) emit("releaseControl")
	else emit("takeControl")
}
</script>

<template>
	<form
		class="flex shrink-0 items-center gap-2 border-t border-white/10 bg-slate-950 px-2 py-2"
		@submit.prevent="submit"
	>
		<button
			v-if="connected"
			type="button"
			class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border transition active:scale-95"
			:class="
				hasControl
					? 'border-emerald-400/40 bg-emerald-500/20 text-emerald-200'
					: 'border-blue-400/30 bg-blue-500/20 text-blue-100'
			"
			:title="hasControl ? t('mobile.releaseTerminalControl') : t('mobile.takeTerminalControl')"
			:aria-label="hasControl ? t('mobile.releaseTerminalControl') : t('mobile.takeTerminalControl')"
			:aria-pressed="hasControl"
			@click="toggleControl(hasControl)"
		>
			<Icon :name="hasControl ? 'mdi:keyboard-off-outline' : 'mdi:keyboard-outline'" :size="20" />
		</button>
		<input
			ref="commandInput"
			v-model="commandDraft"
			type="text"
			inputmode="text"
			enterkeyhint="send"
			autocapitalize="none"
			autocomplete="off"
			autocorrect="off"
			:spellcheck="false"
			:disabled="!hasControl"
			:placeholder="hasControl ? t('mobile.terminalInputPlaceholder') : t('mobile.terminalReadOnly')"
			class="h-10 min-w-0 flex-1 rounded-xl border border-white/10 bg-white/5 px-3 text-base text-white outline-none transition placeholder:text-slate-500 focus:border-blue-400/70 focus:bg-white/[0.08] disabled:cursor-not-allowed"
			@compositionstart="composing = true"
			@compositionend="composing = false"
		/>
		<button
			type="submit"
			:disabled="!hasControl || !commandDraft"
			class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-400/30 bg-blue-500/20 text-blue-100 transition active:scale-95 active:bg-blue-500/35 disabled:border-white/5 disabled:bg-white/5 disabled:text-slate-600"
			:title="t('mobile.sendTerminalInput')"
			:aria-label="t('mobile.sendTerminalInput')"
		>
			<Icon name="mdi:send" :size="19" />
		</button>
	</form>
	<div
		class="flex shrink-0 items-center gap-1.5 overflow-x-auto border-t border-white/10 bg-slate-950 px-2 py-2 transition-opacity [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
		:class="hasControl ? '' : 'opacity-50'"
		role="toolbar"
		:aria-label="t('mobile.terminal')"
	>
		<button
			v-for="shortcut in [
				{ label: 'Esc', data: '\x1b' },
				{ label: 'Tab', data: '\t' }
			]"
			:key="shortcut.label"
			type="button"
			:disabled="!hasControl"
			class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed"
			:aria-label="shortcut.label"
			@pointerdown.prevent
			@click="emit('shortcut', shortcut.data)"
		>
			{{ shortcut.label }}
		</button>
		<button
			type="button"
			:disabled="!hasControl"
			class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed"
			:class="ctrlActive ? 'border-blue-400 bg-blue-500/25 text-blue-200' : ''"
			aria-label="Ctrl"
			:aria-pressed="ctrlActive"
			@pointerdown.prevent
			@click="emit('toggleCtrl')"
		>
			Ctrl
		</button>
		<button
			v-for="symbol in ['/', '-', '_', '.', '~', '|']"
			:key="symbol"
			type="button"
			:disabled="!hasControl"
			class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-blue-400/20 bg-blue-500/10 px-2 font-mono text-base font-medium text-blue-200 transition active:scale-95 active:bg-blue-500/25 disabled:cursor-not-allowed"
			:aria-label="`${t('mobile.insertTerminalSymbol')} ${symbol}`"
			@pointerdown.prevent
			@click="insertSymbol(symbol)"
		>
			{{ symbol }}
		</button>
		<button
			v-for="shortcut in [
				{ label: '←', data: '\x1b[D' },
				{ label: '↑', data: '\x1b[A' },
				{ label: '↓', data: '\x1b[B' },
				{ label: '→', data: '\x1b[C' },
				{ label: '⌫', data: '\x7f' },
				{ label: '↵', data: '\r' }
			]"
			:key="shortcut.label"
			type="button"
			:disabled="!hasControl"
			class="flex h-10 min-w-10 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/5 px-2 font-mono text-sm font-medium text-slate-200 transition active:scale-95 active:bg-white/15 disabled:cursor-not-allowed"
			:class="shortcut.label === '↵' ? 'border-blue-500/40 bg-blue-500/15 text-blue-200' : ''"
			:aria-label="shortcut.label"
			@pointerdown.prevent
			@click="emit('shortcut', shortcut.data)"
		>
			{{ shortcut.label }}
		</button>
	</div>
</template>
