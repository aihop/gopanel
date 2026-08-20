<script setup lang="ts">
import { nextTick, ref, watch } from "vue"
import Icon from "@/components/common/Icon.vue"
import { conversationRunForMessage } from "./codeConversationThread"
import { isConversationSubmitKey } from "./codeConversationMarkdown"
import CodeConversationMessage from "./CodeConversationMessage.vue"
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
	runs,
	closed,
	initializing,
	running,
	canSend,
	loadHistory,
	sendInstruction,
	stopExecution,
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
	() => [messages.value.length, messages.value.at(-1)?.content],
	() => void scrollToBottom(),
)

const submit = async () => {
	const taskId = await sendInstruction()
	if (taskId) emit("taskCreated", taskId)
	await scrollToBottom()
}

const onComposerKeydown = (event: KeyboardEvent) => {
	if (!isConversationSubmitKey(event)) return
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
  <section class="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-transparent">
    <n-spin :show="loading && messages.length === 0" class="flex min-h-0 flex-1 flex-col">
      <div
        ref="listRef"
        class="min-h-0 flex-1 overflow-auto px-4 pb-36 pt-4"
      >
        <div
          v-if="loadError && messages.length === 0"
          class="flex min-h-full flex-col items-center justify-center gap-3"
        >
          <n-empty :description="t('code.historyLoadFailed')" />
          <n-button size="small" @click="loadHistory()">{{ t("code.retry") }}</n-button>
        </div>
        <n-empty
          v-else-if="!loading && messages.length === 0"
          class="flex min-h-full items-center justify-center"
          :description="t('code.conversationEmpty')"
        />
        <div
          v-else
          class="mx-auto flex min-h-full max-w-[46rem] flex-col justify-end gap-4"
        >
          <CodeConversationMessage
            v-for="item in messages"
            :key="item.id"
            :message="item"
            :run="conversationRunForMessage(runs, item.runId)"
          />
        </div>
      </div>
    </n-spin>

    <footer class="conversation-composer absolute inset-x-3 bottom-3 z-10 rounded-2xl border border-slate-200/80 bg-white/95 p-2.5 shadow-[0_10px_30px_rgba(15,23,42,0.08)] backdrop-blur-md dark:border-white/10 dark:bg-[color-mix(in_srgb,var(--bg-default-color)_92%,transparent)]">
      <p class="px-1 pb-1.5 text-xs tracking-[0.01em] text-[var(--n-text-color-3)]">{{ composerDisabledHint() }}</p>
      <div class="flex items-end gap-2">
        <n-input
          v-model:value="draft"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 6 }"
          :disabled="!canSend"
          :placeholder="t('code.promptPlaceholder')"
          @keydown="onComposerKeydown"
        />
        <n-button
          v-if="running"
          circle
          size="small"
          :loading="stopping"
          :title="t('code.stopExecution')"
          @click="stopExecution"
        >
          <template #icon>
            <Icon name="mdi:stop" :size="16" />
          </template>
        </n-button>
        <n-button
          type="primary"
          circle
          size="small"
          :disabled="!canSend || !draft.trim()"
          :loading="sending"
          :title="t('code.sendInstruction')"
          @click="submit"
        >
          <template #icon>
            <Icon name="mdi:arrow-up" :size="16" />
          </template>
        </n-button>
      </div>
    </footer>
  </section>
</template>
