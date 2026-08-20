<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import type { CodeStructureSearchHit } from "@/api/modules/codeEditor"
import { codeEditorMessages } from "../codeEditorMessages"
import { codeFileIcon } from "./codeFileIcon"

defineProps<{
	hits: CodeStructureSearchHit[]
	loading: boolean
	truncated: boolean
}>()

const emit = defineEmits<{
	select: [hit: CodeStructureSearchHit]
}>()

const { t } = useI18n({ messages: codeEditorMessages })
</script>

<template>
  <div class="flex min-h-0 flex-col">
    <div v-if="loading" class="flex items-center justify-center py-3">
      <n-spin size="small" />
    </div>
    <n-empty
      v-else-if="!hits.length"
      size="small"
      class="py-3"
      :description="t('code.structureSearchEmpty')"
    />
    <div v-else class="structure-search-scroll min-h-0 overflow-auto px-1 py-1">
      <button
        v-for="hit in hits"
        :key="`${hit.kind}:${hit.path}:${hit.line || 0}`"
        type="button"
        class="flex w-full flex-col gap-0.5 rounded-lg px-2 py-1.5 text-left hover:bg-slate-100 dark:hover:bg-white/10"
        @click="emit('select', hit)"
      >
        <div class="flex min-w-0 items-center gap-1.5 text-xs text-slate-700 dark:text-[var(--n-text-color)]">
          <Icon
            v-bind="codeFileIcon(hit.path, hit.isDir)"
            :size="15"
          />
          <span class="min-w-0 truncate" :title="hit.path">{{ hit.path }}</span>
          <span v-if="hit.line" class="shrink-0 text-[11px] text-slate-400">L{{ hit.line }}</span>
        </div>
        <div v-if="hit.preview" class="truncate pl-5 font-mono text-[11px] text-slate-400" :title="hit.preview">
          {{ hit.preview }}
        </div>
      </button>
      <div v-if="truncated" class="px-2 py-2 text-[11px] text-slate-400">
        {{ t("code.structureSearchTruncated") }}
      </div>
    </div>
  </div>
</template>
