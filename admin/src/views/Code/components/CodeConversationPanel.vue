<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useMessage } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { conversationRunForMessage } from "./codeConversationThread"
import { isConversationSubmitKey } from "./codeConversationMarkdown"
import {
	attachmentsFromPaths,
	conversationAttachmentFromPath,
	extractDroppedAttachments,
} from "./codeConversationAttachments"
import CodeConversationAttachments from "./CodeConversationAttachments.vue"
import CodeConversationMessage from "./CodeConversationMessage.vue"
import ProjectStructurePanel from "./ProjectStructurePanel.vue"
import { useCodeConversation } from "./useCodeConversation"

const props = defineProps<{
	sessionId: number | null
	taskId: number | null
}>()
const emit = defineEmits<{
	taskCreated: [taskId: number]
}>()
const toast = useMessage()

const {
	t,
	loading,
	sending,
	stopping,
	loadError,
	draft,
	attachments,
	displayMessages,
	runs,
	closed,
	initializing,
	running,
	canSend,
	hasComposerContent,
	workDir,
	loadHistory,
	addAttachments,
	removeAttachment,
	sendInstruction,
	stopExecution,
} = useCodeConversation(
	() => props.sessionId,
	() => props.taskId,
)

const listRef = ref<HTMLElement | null>(null)
const dropping = ref(false)

const scrollToBottom = async () => {
	await nextTick()
	const list = listRef.value
	if (list) list.scrollTop = list.scrollHeight
}

watch(
	() => [displayMessages.value.length, displayMessages.value.at(-1)?.content],
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

const wailsRuntime = () => (window as Window & { runtime?: WailsRuntime }).runtime

const acceptDroppedAttachments = (dataTransfer: DataTransfer | null) => {
	const result = extractDroppedAttachments(dataTransfer, workDir.value)
	if (result.attachments.length) {
		addAttachments(result.attachments)
		return true
	}
	if (result.missingPath && !wailsRuntime()?.OnFileDrop) toast.warning(t("code.attachmentPathMissing"))
	return false
}

const onDragOver = (event: DragEvent) => {
	event.preventDefault()
	if (event.dataTransfer) event.dataTransfer.dropEffect = "copy"
	dropping.value = true
}

const onDragLeave = (event: DragEvent) => {
	const next = event.relatedTarget
	if (next instanceof Node && event.currentTarget instanceof Node && event.currentTarget.contains(next)) return
	dropping.value = false
}

const onDrop = (event: DragEvent) => {
	event.preventDefault()
	dropping.value = false
	acceptDroppedAttachments(event.dataTransfer)
}

const onPaste = (event: ClipboardEvent) => {
	const data = event.clipboardData
	if (!data?.files.length && !data?.getData("text/uri-list")) return
	if (acceptDroppedAttachments(data)) event.preventDefault()
}

const attachStructureFile = (file: { path: string }) => {
	const item = conversationAttachmentFromPath(file.path, workDir.value)
	if (item) addAttachments([item])
}

const insertStructureSnippet = (snippet: { path: string; startLine: number; endLine: number }) => {
	const item = conversationAttachmentFromPath(snippet.path, workDir.value)
	if (!item) return
	addAttachments([
		{
			...item,
			id: `${item.path}:${snippet.startLine}-${snippet.endLine}`,
			name: `${item.name}:${snippet.startLine}-${snippet.endLine}`,
			startLine: snippet.startLine,
			endLine: snippet.endLine,
		},
	])
}

type WailsRuntime = {
	OnFileDrop?: (callback: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean) => void
	OnFileDropOff?: () => void
}

onMounted(() => {
	wailsRuntime()?.OnFileDrop?.((_x, _y, paths) => {
		addAttachments(attachmentsFromPaths(paths, workDir.value))
	}, true)
})
onBeforeUnmount(() => {
	wailsRuntime()?.OnFileDropOff?.()
})
</script>

<template>
  <section class="flex h-full min-h-0 min-w-0 flex-1 overflow-hidden bg-transparent">
    <div
      class="conversation-drop-target relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      :class="dropping ? 'conversation-drop-target--active' : ''"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
      @paste="onPaste"
    >
      <n-spin
        :show="loading && displayMessages.length === 0"
        class="flex min-h-0 flex-1 flex-col"
        content-class="flex min-h-0 flex-1 flex-col"
      >
        <div
          ref="listRef"
          class="min-h-0 flex-1 overflow-auto px-4 pb-40 pt-4"
        >
          <div
            v-if="loadError && displayMessages.length === 0"
            class="flex min-h-full flex-col items-center justify-center gap-3"
          >
            <n-empty :description="t('code.historyLoadFailed')" />
            <n-button
              size="small"
              @click="loadHistory()"
            >
              {{ t("code.retry") }}
            </n-button>
          </div>
          <n-empty
            v-else-if="!loading && displayMessages.length === 0"
            class="flex min-h-full items-center justify-center"
            :description="t('code.conversationEmpty')"
          />
          <div
            v-else
            class="flex min-h-full w-full flex-col justify-end gap-3"
          >
            <CodeConversationMessage
              v-for="item in displayMessages"
              :key="item.id"
              :message="item"
              :run="conversationRunForMessage(runs, item.runId)"
            />
          </div>
        </div>
      </n-spin>

      <footer class="conversation-composer absolute inset-x-3 bottom-3 z-10 rounded-2xl border border-slate-200/80 bg-white/95 p-2.5 shadow-[0_10px_30px_rgba(15,23,42,0.08)] backdrop-blur-md dark:border-white/10 dark:bg-[color-mix(in_srgb,var(--bg-default-color)_92%,transparent)]">
        <p class="px-1 pb-1.5 text-xs tracking-[0.01em] text-[var(--n-text-color-3)]">
          {{ dropping ? t("code.attachmentDropActive") : composerDisabledHint() }}
        </p>
        <CodeConversationAttachments
          v-if="attachments.length"
          class="px-1 pb-2"
          compact
          removable
          :attachments="attachments"
          :session-id="sessionId"
          @remove="removeAttachment"
        />
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
              <Icon
                name="mdi:stop"
                :size="16"
              />
            </template>
          </n-button>
          <n-button
            type="primary"
            circle
            size="small"
            :disabled="!canSend || !hasComposerContent"
            :loading="sending"
            :title="t('code.sendInstruction')"
            @click="submit"
          >
            <template #icon>
              <Icon
                name="mdi:arrow-up"
                :size="16"
              />
            </template>
          </n-button>
        </div>
      </footer>
    </div>

    <aside
      v-if="sessionId"
      class="hidden min-h-0 w-72 shrink-0 self-stretch overflow-hidden border-l border-slate-200/70 lg:flex lg:flex-col dark:border-white/10"
    >
      <ProjectStructurePanel
        :key="sessionId"
        class="h-full min-h-0"
        attach-to-chat
        :session-id="sessionId"
        @select-file="attachStructureFile"
        @insert-snippet="insertStructureSnippet"
      />
    </aside>
  </section>
</template>

<style scoped>
.conversation-drop-target {
	--wails-drop-target: drop;
}

.conversation-drop-target--active .conversation-composer,
.conversation-drop-target :deep(.wails-drop-target-active) .conversation-composer {
	border-color: rgb(59 130 246 / 0.55);
	box-shadow: 0 10px 30px rgb(37 99 235 / 0.12);
}
</style>
