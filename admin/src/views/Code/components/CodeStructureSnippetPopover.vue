<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getCodeSessionFile } from "@/api/modules/codeEditor"
import FtEditor from "@/components/FtEditor/index.vue"
import Icon from "@/components/common/Icon.vue"
import { Languages } from "@/global/mimetype"
import { codeEditorMessages } from "../codeEditorMessages"
import { fileExtension } from "./codeConversationAttachments"
import { formatFileLineRef, selectionLineRange } from "./codeFileSnippet"

const props = defineProps<{
	sessionId: number
	path: string
	attachToChat?: boolean
}>()

const emit = defineEmits<{
	insert: [snippet: { path: string; startLine: number; endLine: number }]
}>()

const { t } = useI18n({ messages: codeEditorMessages })
const toast = useMessage()
const editorRef = ref<{ editorRef?: { getSelection?: () => { startLineNumber: number; endLineNumber: number } | null } } | null>(null)
const loading = ref(false)
const loadError = ref(false)
const content = ref("")

const language = computed(() => {
	const extension = fileExtension(props.path)
	return Languages.find(item => item.value.some(value => value.toLowerCase() === extension))?.label || "plaintext"
})

const currentRange = () => selectionLineRange(editorRef.value?.editorRef?.getSelection?.() || null)

const loadFile = async () => {
	loading.value = true
	loadError.value = false
	try {
		const response = await getCodeSessionFile(props.sessionId, props.path)
		content.value = response.data.content || ""
	} catch {
		content.value = ""
		loadError.value = true
	} finally {
		loading.value = false
	}
}

const copySnippet = async () => {
	const range = currentRange()
	try {
		await navigator.clipboard.writeText(formatFileLineRef(props.path, range.start, range.end))
		toast.success(t("code.snippetCopied"))
	} catch {
		toast.error(t("code.snippetCopyFailed"))
	}
}

const insertSnippet = () => {
	const range = currentRange()
	emit("insert", { path: props.path, startLine: range.start, endLine: range.end })
}

watch(
	() => [props.sessionId, props.path],
	() => void loadFile(),
	{ immediate: true },
)
</script>

<template>
  <div class="w-[520px] max-w-[82vw]">
    <div class="mb-2 truncate text-xs font-medium text-slate-600 dark:text-[var(--n-text-color)]" :title="path">
      {{ path }}
    </div>
    <p class="mb-2 text-[11px] tracking-[0.01em] text-[var(--n-text-color-3)]">
      {{ t("code.snippetSelectHint") }}
    </p>
    <div v-if="loading" class="flex h-64 items-center justify-center">
      <n-spin size="small" />
    </div>
    <div v-else-if="loadError" class="py-6 text-center text-xs text-slate-500">
      {{ t("code.snippetLoadFailed") }}
    </div>
    <template v-else>
      <FtEditor
        ref="editorRef"
        v-model="content"
        class="overflow-hidden rounded-lg"
        :language="language"
        height="280px"
        :show-toolbar="false"
        readonly
      />
      <div class="mt-2 flex justify-end gap-2">
        <n-button size="tiny" @click="copySnippet">
          <template #icon>
            <Icon name="mdi:content-copy" :size="14" />
          </template>
          {{ t("code.copySnippet") }}
        </n-button>
        <n-button v-if="attachToChat" type="primary" size="tiny" @click="insertSnippet">
          {{ t("code.insertSnippet") }}
        </n-button>
      </div>
    </template>
  </div>
</template>
