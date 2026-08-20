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
import { formatFileLineRef, nextMatchIndex, selectionLineRange } from "./codeFileSnippet"

type SnippetRange = { startLineNumber: number; startColumn?: number; endLineNumber: number; endColumn?: number }
type SnippetEditor = {
	getSelection?: () => SnippetRange | null
	getModel?: () => {
		findMatches?: (
			searchString: string,
			searchOnlyEditableRange: boolean,
			isRegex: boolean,
			matchCase: boolean,
			wordSeparators: string | null,
			captureMatches: boolean,
		) => Array<{ range: SnippetRange }>
	} | null
	setSelection?: (range: SnippetRange) => void
	revealRangeInCenter?: (range: SnippetRange) => void
	focus?: () => void
}

const props = defineProps<{
	sessionId: number
	path: string
	attachToChat?: boolean
}>()

const emit = defineEmits<{
	insert: [snippet: { path: string; startLine: number; endLine: number }]
	close: []
}>()

const { t } = useI18n({ messages: codeEditorMessages })
const toast = useMessage()
const editorRef = ref<{ editorRef?: SnippetEditor } | null>(null)
const loading = ref(false)
const loadError = ref(false)
const content = ref("")
const searchQuery = ref("")
const matchIndex = ref(0)
const matchTotal = ref(0)

const language = computed(() => {
	const extension = fileExtension(props.path)
	return Languages.find(item => item.value.some(value => value.toLowerCase() === extension))?.label || "plaintext"
})

const currentRange = () => selectionLineRange(editorRef.value?.editorRef?.getSelection?.() || null)
const searchStatus = computed(() => {
	if (!searchQuery.value.trim()) return ""
	if (!matchTotal.value) return t("code.snippetSearchEmpty")
	return t("code.snippetSearchCount", { current: matchIndex.value + 1, total: matchTotal.value })
})

const findMatches = () => {
	const query = searchQuery.value.trim()
	if (!query) return []
	return editorRef.value?.editorRef?.getModel?.()?.findMatches?.(query, false, false, false, null, true) || []
}

const revealMatch = (step = 0) => {
	const editor = editorRef.value?.editorRef
	const matches = findMatches()
	matchTotal.value = matches.length
	if (!editor || !matches.length) {
		matchIndex.value = 0
		return
	}
	const index = step === 0 ? 0 : nextMatchIndex(matchIndex.value, matches.length, step)
	const range = matches[index]?.range
	if (!range) return
	matchIndex.value = index
	editor.setSelection?.(range)
	editor.revealRangeInCenter?.(range)
}

const onSearchKeydown = (event: KeyboardEvent) => {
	if (event.key !== "Enter") return
	event.preventDefault()
	revealMatch(event.shiftKey ? -1 : 1)
}

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
watch(searchQuery, () => revealMatch(0))
</script>

<template>
  <div class="w-[520px] max-w-[82vw]">
    <div class="mb-2 flex items-start gap-2">
      <div class="min-w-0 flex-1 truncate text-xs font-medium text-slate-600 dark:text-[var(--n-text-color)]" :title="path">
        {{ path }}
      </div>
      <n-button
        quaternary
        circle
        size="tiny"
        class="shrink-0"
        :title="t('code.closeSnippet')"
        @click="emit('close')"
      >
        <template #icon>
          <Icon name="mdi:close" :size="16" />
        </template>
      </n-button>
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
      <div class="mb-2 flex items-center gap-1">
        <n-input
          v-model:value="searchQuery"
          size="tiny"
          clearable
          :placeholder="t('code.snippetSearchPlaceholder')"
          @keydown="onSearchKeydown"
        >
          <template #prefix>
            <Icon name="mdi:magnify" :size="14" />
          </template>
          <template #suffix>
            <span class="pr-1 text-[11px] text-slate-400">{{ searchStatus }}</span>
          </template>
        </n-input>
        <n-button quaternary circle size="tiny" :disabled="!matchTotal" :title="t('code.snippetSearchPrev')" @click="revealMatch(-1)">
          <template #icon>
            <Icon name="mdi:chevron-up" :size="16" />
          </template>
        </n-button>
        <n-button quaternary circle size="tiny" :disabled="!matchTotal" :title="t('code.snippetSearchNext')" @click="revealMatch(1)">
          <template #icon>
            <Icon name="mdi:chevron-down" :size="16" />
          </template>
        </n-button>
      </div>
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
