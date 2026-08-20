<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { getCodeSessionImagePreview } from "@/api/modules/code"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"
import type { ComposerAttachment, ConversationAttachment } from "./codeConversationAttachments"
import { isWorkspaceRelativePath } from "./codeConversationAttachments"

const { t } = useI18n({ messages: codeWorkspaceMessages })

const props = withDefaults(
	defineProps<{
		attachments: Array<ConversationAttachment | ComposerAttachment>
		sessionId?: number | null
		removable?: boolean
		compact?: boolean
	}>(),
	{ sessionId: null, removable: false, compact: false },
)

const emit = defineEmits<{
	remove: [path: string]
}>()

const previewSrc = ref<Record<string, string>>({})
let requestVersion = 0

const items = computed(() =>
	props.attachments.map(item => ({
		...item,
		src: ("previewUrl" in item && item.previewUrl) || previewSrc.value[item.path] || "",
	})),
)

const loadPreviews = async () => {
	const version = ++requestVersion
	const sessionId = props.sessionId
	if (!sessionId) return
	const next: Record<string, string> = { ...previewSrc.value }
	await Promise.all(
		props.attachments.map(async item => {
			if (item.kind !== "image" || !isWorkspaceRelativePath(item.path) || next[item.path]) return
			if ("previewUrl" in item && item.previewUrl) return
			try {
				const response = await getCodeSessionImagePreview(sessionId, item.path)
				if (version !== requestVersion || response.code !== 0) return
				next[item.path] = `data:${response.data.contentType};base64,${response.data.content}`
			} catch {
				return
			}
		}),
	)
	if (version === requestVersion) previewSrc.value = next
}

watch(
	() => [props.sessionId, props.attachments.map(item => item.path).join("\n")],
	() => void loadPreviews(),
	{ immediate: true },
)
onBeforeUnmount(() => {
	requestVersion += 1
})
</script>

<template>
  <ul
    v-if="items.length"
    class="flex flex-wrap gap-2"
    :class="compact ? 'pb-1' : ''"
  >
    <li
      v-for="item in items"
      :key="item.path + ':' + (item.startLine || '') + '-' + (item.endLine || '')"
      class="relative"
    >
      <figure
        v-if="item.kind === 'image' && item.src"
        class="overflow-hidden rounded-xl border border-slate-200/80 bg-slate-50 dark:border-white/10 dark:bg-white/5"
        :class="compact ? 'h-16 w-16' : 'max-h-56 max-w-xs'"
        :title="item.path"
      >
        <img
          :src="item.src"
          :alt="item.name"
          :class="compact ? 'h-full w-full object-cover' : 'max-h-56 w-auto object-contain'"
        >
      </figure>
      <div
        v-else
        class="flex max-w-[16rem] items-center gap-1.5 rounded-lg border border-slate-200/80 bg-white/90 px-2 py-1 text-[11px] tracking-[0.01em] dark:border-white/10 dark:bg-white/5"
        :title="item.path"
      >
        <Icon
          :name="item.kind === 'image' ? 'mdi:image-outline' : 'mdi:file-code-outline'"
          :size="14"
        />
        <span class="min-w-0 truncate">{{ item.name }}</span>
      </div>
      <button
        v-if="removable"
        type="button"
        class="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-slate-700 text-white"
        :title="t('code.attachmentRemove')"
        @click.stop="emit('remove', item.startLine && item.endLine ? `${item.path}:${item.startLine}-${item.endLine}` : item.path)"
      >
        <Icon
          name="mdi:close"
          :size="10"
        />
      </button>
    </li>
  </ul>
</template>
