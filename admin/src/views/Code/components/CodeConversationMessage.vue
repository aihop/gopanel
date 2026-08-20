<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue"
import { useI18n } from "vue-i18n"
import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import { parseConversationAttachments } from "./codeConversationAttachments"
import { sanitizeConversationMarkdown } from "./codeConversationMarkdown"
import { isUserConversationMessage } from "./codeConversationThread"
import CodeConversationAttachments from "./CodeConversationAttachments.vue"

const props = defineProps<{
	message: AIMessage
	run?: CodeExecutionRun
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const isUser = computed(() => isUserConversationMessage(props.message.role))
const parsed = computed(() => parseConversationAttachments(props.message.content || ""))
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
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-white"
      :class="isUser ? 'bg-blue-500' : 'bg-slate-400 dark:bg-slate-500'"
      :title="isUser ? t('code.userMessage') : t('code.executorMessage')"
    >
      <Icon
        :name="isUser ? 'mdi:account' : 'mdi:robot-outline'"
        :size="16"
      />
    </div>
    <div
      class="min-w-0"
      :class="isUser ? 'max-w-[min(78%,28rem)]' : 'max-w-[min(86%,38rem)]'"
    >
      <div
        v-if="run && !isUser"
        class="mb-1 flex flex-wrap items-center gap-2 px-1 text-[11px] tracking-[0.01em] text-[var(--n-text-color-3)]"
      >
        <span>{{ t(`code.runStatus_${run.status}`) }}</span>
        <span v-if="run.durationMs">{{ t("code.runDuration", { duration: run.durationMs }) }}</span>
        <span v-if="run.totalTokens">{{ t("code.taskTokens", { count: run.totalTokens }) }}</span>
      </div>
      <div
        class="conversation-bubble px-3.5 py-2.5 text-[13px] leading-relaxed tracking-[0.01em]"
        :class="isUser
          ? 'conversation-bubble--user rounded-[18px] rounded-br-md bg-blue-500 text-white'
          : 'conversation-bubble--agent rounded-[18px] rounded-bl-md bg-white text-slate-700 shadow-[0_1px_2px_rgba(15,23,42,0.06)] ring-1 ring-slate-200/80 dark:bg-white/10 dark:text-[var(--n-text-color)] dark:ring-white/10'"
      >
        <CodeConversationAttachments
          v-if="parsed.attachments.length"
          :class="parsed.text ? 'mb-2' : ''"
          :attachments="parsed.attachments"
          :session-id="message.sessionId"
        />
        <MdPreview
          v-if="parsed.text"
          :editor-id="`code-conversation-${message.sessionId}-${message.id}`"
          :model-value="parsed.text"
          :sanitize="sanitizeConversationMarkdown"
          no-mermaid
          no-echarts
          no-katex
          class="conversation-markdown"
        />
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
}

.conversation-bubble--user .conversation-markdown :deep(pre),
.conversation-bubble--user .conversation-markdown :deep(code) {
	background: rgb(255 255 255 / 0.16);
	color: inherit;
}

.conversation-bubble--user .conversation-markdown :deep(a) {
	color: rgb(219 234 254);
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
