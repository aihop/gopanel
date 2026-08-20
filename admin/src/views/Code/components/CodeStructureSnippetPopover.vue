<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getCodeSessionFile } from "@/api/modules/codeEditor"
import Icon from "@/components/common/Icon.vue"
import { codeEditorMessages } from "../codeEditorMessages"
import { clampLineRange, formatFileSnippet, splitFileLines } from "./codeFileSnippet"

const props = defineProps<{
	sessionId: number
	path: string
	attachToChat?: boolean
}>()

const emit = defineEmits<{
	insert: [snippet: string]
}>()

const { t } = useI18n({ messages: codeEditorMessages })
const toast = useMessage()
const loading = ref(false)
const loadError = ref(false)
const content = ref("")
const startLine = ref(1)
const endLine = ref(1)
const picking = ref(false)

const lines = computed(() => splitFileLines(content.value))
const range = computed(() => clampLineRange(startLine.value, endLine.value, lines.value.length || 1))

const loadFile = async () => {
	loading.value = true
	loadError.value = false
	try {
		const response = await getCodeSessionFile(props.sessionId, props.path)
		content.value = response.data.content || ""
		startLine.value = 1
		endLine.value = Math.min(20, splitFileLines(content.value).length || 1)
		picking.value = false
	} catch {
		content.value = ""
		loadError.value = true
	} finally {
		loading.value = false
	}
}

const selectLine = (line: number) => {
	if (!picking.value) {
		startLine.value = line
		endLine.value = line
		picking.value = true
		return
	}
	endLine.value = line
	picking.value = false
}

const snippetText = () => formatFileSnippet(props.path, content.value, startLine.value, endLine.value)

const copySnippet = async () => {
	try {
		await navigator.clipboard.writeText(snippetText())
		toast.success(t("code.snippetCopied"))
	} catch {
		toast.error(t("code.snippetCopyFailed"))
	}
}

const insertSnippet = () => {
	emit("insert", snippetText())
}

watch(
	() => [props.sessionId, props.path],
	() => void loadFile(),
	{ immediate: true },
)
</script>

<template>
  <div class="w-[360px] max-w-[80vw]">
    <div class="mb-2 truncate text-xs font-medium text-slate-600 dark:text-[var(--n-text-color)]" :title="path">
      {{ path }}
    </div>
    <div v-if="loading" class="flex h-32 items-center justify-center">
      <n-spin size="small" />
    </div>
    <div v-else-if="loadError" class="py-6 text-center text-xs text-slate-500">
      {{ t("code.snippetLoadFailed") }}
    </div>
    <template v-else>
      <div class="mb-2 flex items-center gap-2">
        <n-input-number
          v-model:value="startLine"
          :min="1"
          :max="lines.length || 1"
          :show-button="false"
          size="tiny"
          :placeholder="t('code.snippetFromLine')"
        />
        <span class="text-xs text-slate-400">-</span>
        <n-input-number
          v-model:value="endLine"
          :min="1"
          :max="lines.length || 1"
          :show-button="false"
          size="tiny"
          :placeholder="t('code.snippetToLine')"
        />
      </div>
      <div class="mb-2 max-h-56 overflow-auto rounded-lg bg-slate-50 font-mono text-[11px] leading-5 dark:bg-white/5">
        <button
          v-for="(line, index) in lines"
          :key="index"
          type="button"
          class="flex w-full gap-2 px-2 text-left hover:bg-blue-50 dark:hover:bg-white/10"
          :class="index + 1 >= range.start && index + 1 <= range.end ? 'bg-blue-100 dark:bg-blue-500/20' : ''"
          @click="selectLine(index + 1)"
        >
          <span class="w-8 shrink-0 text-right text-slate-400">{{ index + 1 }}</span>
          <span class="min-w-0 whitespace-pre-wrap break-all text-slate-700 dark:text-[var(--n-text-color)]">{{ line || " " }}</span>
        </button>
      </div>
      <div class="flex justify-end gap-2">
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
