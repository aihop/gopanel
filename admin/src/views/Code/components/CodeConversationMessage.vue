<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from "vue"
import { useI18n } from "vue-i18n"
import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import { parseConversationAttachments } from "./codeConversationAttachments"
import { sanitizeConversationMarkdown } from "./codeConversationMarkdown"
import { conversationMessageText, isUserConversationMessage } from "./codeConversationThread"
import CodeConversationAttachments from "./CodeConversationAttachments.vue"

const props = defineProps<{
	message: AIMessage
	run?: CodeExecutionRun
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const isUser = computed(() => isUserConversationMessage(props.message.role))
const parsed = computed(() => parseConversationAttachments(props.message.content || ""))
const isRunActive = computed(() => props.run?.status === "running" || props.run?.status === "queued")
const expanded = ref(false)
const previewText = computed(() => conversationMessageText(parsed.value.text, expanded.value, isRunActive.value))
const canExpand = computed(
	() => conversationMessageText(parsed.value.text, false, isRunActive.value) !== parsed.value.text,
)
const hasBubble = computed(() => Boolean(parsed.value.attachments.length || parsed.value.text))
const MdPreview = defineAsyncComponent(async () => {
	const [module] = await Promise.all([import("md-editor-v3"), import("md-editor-v3/lib/preview.css")])
	return module.MdPreview
})
</script>

<template>
  <article
    class="flex w-full items-end gap-2"
    :class="isUser ? 'flex-row-reverse' : 'flex-row'"
  >
    <div
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full"
      :class="isUser ? 'bg-blue-50 text-blue-500 dark:bg-blue-500/15 dark:text-blue-300' : 'bg-slate-100 text-slate-500 dark:bg-white/10 dark:text-slate-300'"
      :title="isUser ? t('code.userMessage') : t('code.executorMessage')"
    >
      <Icon
        :name="isUser ? 'mdi:account' : 'mdi:robot-outline'"
        :size="16"
      />
    </div>
    <div
      class="min-w-0"
      :class="isUser ? 'max-w-[80%]' : 'max-w-[90%]'"
    >
      <div
        v-if="run && !isUser"
        class="mb-1 flex flex-wrap items-center gap-2 px-1 text-[11px] tracking-[0.01em] text-[var(--n-text-color-3)]"
      >
        <span
          v-if="isRunActive"
          class="relative inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center"
          aria-hidden="true"
        >
          <span class="absolute inset-0 animate-spin rounded-full border border-current border-r-transparent motion-reduce:animate-none" />
          <span class="h-1.5 w-1.5 rounded-full bg-current animate-pulse motion-reduce:animate-none" />
        </span>
        <span>{{ t(`code.runStatus_${run.status}`) }}</span>
        <span v-if="!isRunActive && run.durationMs">{{ t("code.runDuration", { duration: run.durationMs }) }}</span>
        <span v-if="!isRunActive && run.totalTokens">{{ t("code.taskTokens", { count: run.totalTokens }) }}</span>
      </div>
      <div
        v-if="hasBubble"
        class="conversation-bubble px-3.5 py-2.5 text-[13px] leading-relaxed tracking-[0.01em]"
        :class="isUser
          ? 'conversation-bubble--user rounded-[18px] rounded-br-md bg-blue-50 text-slate-700 dark:bg-blue-500/15 dark:text-[var(--n-text-color)]'
          : 'conversation-bubble--agent rounded-[18px] rounded-bl-md bg-white text-slate-700 shadow-[0_1px_2px_rgba(15,23,42,0.06)] ring-1 ring-slate-200/80 dark:bg-white/10 dark:text-[var(--n-text-color)] dark:ring-white/10'"
      >
        <CodeConversationAttachments
          v-if="parsed.attachments.length"
          :class="parsed.text ? 'mb-2' : ''"
          :attachments="parsed.attachments"
          :session-id="message.sessionId"
        />
        <MdPreview
          v-if="previewText"
          :editor-id="`code-conversation-${message.sessionId}-${message.id}`"
          :model-value="previewText"
          :sanitize="sanitizeConversationMarkdown"
          no-mermaid
          no-echarts
          no-katex
          class="conversation-markdown"
        />
        <button
          v-if="canExpand"
          type="button"
          class="mt-1 text-[11px] text-slate-400"
          @click="expanded = !expanded"
        >
          {{ expanded ? t("code.collapseMessage") : isRunActive ? t("code.expandRunningOutput") : t("code.expandMessage") }}
        </button>
      </div>
    </div>
  </article>
</template>

<style scoped>
.conversation-markdown :deep(.md-editor-preview) {
	padding: 0;
	font-size: 13px;
	line-height: 1.65;
	letter-spacing: 0.01em;
	color: inherit;
	background: transparent;
}

.conversation-markdown :deep(.md-editor) {
	width: auto;
	max-width: 100%;
	border: none;
	box-shadow: none;
	background: transparent;
}

.conversation-markdown :deep(pre) {
	margin: 0.5rem 0;
	overflow: auto;
	border-radius: 0.75rem;
	padding: 0.75rem 1rem;
	background: rgb(15 23 42 / 0.06);
	scrollbar-width: thin;
	scrollbar-color: rgb(148 163 184 / 0.28) transparent;
}

.conversation-markdown :deep(pre::-webkit-scrollbar) {
	width: 5px;
	height: 5px;
}

.conversation-markdown :deep(pre::-webkit-scrollbar-track) {
	background: transparent;
}

.conversation-markdown :deep(pre::-webkit-scrollbar-thumb) {
	border-radius: 999px;
	background: rgb(148 163 184 / 0.28);
}

.conversation-markdown :deep(code) {
	font-size: 12px;
}

.conversation-markdown :deep(p) {
	margin: 0.35em 0;
}

.conversation-markdown :deep(p:first-child) {
	margin-top: 0;
}

.conversation-markdown :deep(p:last-child) {
	margin-bottom: 0;
}

.conversation-markdown :deep(h1),
.conversation-markdown :deep(h2),
.conversation-markdown :deep(h3) {
	margin: 0.8em 0 0.35em;
	font-weight: 600;
	letter-spacing: 0.01em;
}

.conversation-markdown :deep(ul),
.conversation-markdown :deep(ol) {
	margin: 0.4em 0;
	padding-left: 1.2em;
}

.conversation-markdown :deep(table) {
	margin: 0.6em 0;
	width: 100%;
	border-collapse: collapse;
	font-size: 12px;
}

.conversation-markdown :deep(th),
.conversation-markdown :deep(td) {
	border: 1px solid color-mix(in srgb, var(--border-color) 70%, transparent);
	padding: 0.35em 0.55em;
	text-align: left;
}
</style>
