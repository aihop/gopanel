<script setup lang="ts">
import { nextTick, ref, watch } from "vue"
import { conversationMessageText, isLongConversationMessage } from "./codeConversationThread"
import { useCodeConversation } from "./useCodeConversation"

const props = defineProps<{
	sessionId: number | null
	taskId: number | null
}>()
const emit = defineEmits<{
	taskCreated: [taskId: number]
}>()

const {
	t,
	loading,
	sending,
	stopping,
	loadError,
	draft,
	messages,
	expandedMessageIds,
	closed,
	initializing,
	running,
	canSend,
	loadHistory,
	sendInstruction,
	stopExecution,
	toggleMessageExpanded,
} = useCodeConversation(
	() => props.sessionId,
	() => props.taskId,
)

const listRef = ref<HTMLElement | null>(null)

const scrollToBottom = async () => {
	await nextTick()
	const list = listRef.value
	if (list) list.scrollTop = list.scrollHeight
}

watch(
	() => messages.value.length,
	() => void scrollToBottom(),
)

const submit = async () => {
	const taskId = await sendInstruction()
	if (taskId) emit("taskCreated", taskId)
	await scrollToBottom()
}

const onComposerKeydown = (event: KeyboardEvent) => {
	if (event.key !== "Enter" || !(event.metaKey || event.ctrlKey) || event.isComposing) return
	event.preventDefault()
	void submit()
}

const composerDisabledHint = () => {
	if (closed.value) return t("code.conversationClosed")
	if (initializing.value) return t("code.conversationInitializing")
	return t("code.promptHint")
}
</script>

<template>
  <section class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white">
    <n-spin :show="loading && messages.length === 0" class="flex min-h-0 flex-1 flex-col">
      <div
        v-if="loadError && messages.length === 0"
        class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-8"
      >
        <n-empty :description="t('code.historyLoadFailed')" />
        <n-button size="small" @click="loadHistory()">{{ t("code.retry") }}</n-button>
      </div>
      <n-empty
        v-else-if="!loading && messages.length === 0"
        class="min-h-0 flex-1"
        :description="t('code.conversationEmpty')"
      />
      <div
        v-else
        ref="listRef"
        class="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-4"
      >
        <article
          v-for="item in messages"
          :key="item.id"
          class="rounded-2xl border border-slate-200 p-4"
          :class="item.role === 'user' ? 'bg-blue-50' : 'bg-slate-50'"
        >
          <div class="mb-2 flex items-center justify-between gap-3">
            <n-tag size="small" :type="item.role === 'user' ? 'info' : 'success'" :bordered="false">
              {{ item.role === "user" ? t("code.userMessage") : t("code.executorMessage") }}
            </n-tag>
            <span class="text-xs text-slate-400">{{ new Date(item.createdAt).toLocaleString() }}</span>
          </div>
          <pre class="whitespace-pre-wrap break-words font-sans text-sm leading-6 text-slate-700">{{
            conversationMessageText(item.content, expandedMessageIds.has(item.id))
          }}</pre>
          <n-button
            v-if="isLongConversationMessage(item.content)"
            text
            type="primary"
            size="small"
            class="mt-2"
            @click="toggleMessageExpanded(item.id)"
          >
            {{ expandedMessageIds.has(item.id) ? t("code.collapseMessage") : t("code.expandMessage") }}
          </n-button>
        </article>
      </div>
    </n-spin>

    <footer class="shrink-0 border-t border-slate-200 p-3">
      <p class="mb-2 text-xs text-slate-500">{{ composerDisabledHint() }}</p>
      <div class="flex items-end gap-2">
        <n-input
          v-model:value="draft"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          :disabled="!canSend"
          :placeholder="t('code.promptPlaceholder')"
          @keydown="onComposerKeydown"
        />
        <div class="flex shrink-0 flex-col gap-2">
          <n-button
            v-if="running"
            size="small"
            :loading="stopping"
            @click="stopExecution"
          >
            {{ t("code.stopExecution") }}
          </n-button>
          <n-button
            type="primary"
            size="small"
            :disabled="!canSend || !draft.trim()"
            :loading="sending"
            @click="submit"
          >
            {{ t(sending ? "code.sendingInstruction" : "code.sendInstruction") }}
          </n-button>
        </div>
      </div>
    </footer>
  </section>
</template>
