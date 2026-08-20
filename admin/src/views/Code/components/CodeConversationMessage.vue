<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue"
import { useI18n } from "vue-i18n"
import type { AIMessage, CodeExecutionRun } from "@/api/interface/code"
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
const parsed = computed(() => parseConversationAttachments(props.message.content || ""))
const MdPreview = defineAsyncComponent(async () => {
	const [module] = await Promise.all([import("md-editor-v3"), import("md-editor-v3/lib/preview.css")])
	return module.MdPreview
})
</script>

<template>
  <article
    class="flex w-full"
    :class="isUserConversationMessage(message.role) ? 'justify-end' : 'justify-start'"
  >
    <div
      class="max-w-[min(100%,42rem)]"
      :class="isUserConversationMessage(message.role) ? 'rounded-2xl bg-slate-100 px-3.5 py-2.5 dark:bg-white/10' : 'px-1'"
    >
      <div
        v-if="run && !isUserConversationMessage(message.role)"
        class="mb-1.5 flex flex-wrap items-center gap-2 text-[11px] tracking-[0.01em] text-[var(--n-text-color-3)]"
      >
        <span>{{ t(`code.runStatus_${run.status}`) }}</span>
        <span v-if="run.durationMs">{{ t("code.runDuration", { duration: run.durationMs }) }}</span>
        <span v-if="run.totalTokens">{{ t("code.taskTokens", { count: run.totalTokens }) }}</span>
      </div>
      <CodeConversationAttachments
        v-if="parsed.attachments.length"
        class="mb-2"
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
  </article>
</template>

<style scoped>
.conversation-markdown :deep(.md-editor-preview) {
	font-size: 13px;
	line-height: 1.65;
	letter-spacing: 0.01em;
	color: var(--fg-default-color, inherit);
	background: transparent;
}

.conversation-markdown :deep(.md-editor) {
	background: transparent;
}

.conversation-markdown :deep(pre) {
	margin: 0.5rem 0;
	overflow: auto;
	border-radius: 0.75rem;
	padding: 0.75rem 1rem;
	background: color-mix(in srgb, var(--border-color) 18%, transparent);
}

.conversation-markdown :deep(code) {
	font-size: 12px;
}

.conversation-markdown :deep(p) {
	margin: 0.35em 0;
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
