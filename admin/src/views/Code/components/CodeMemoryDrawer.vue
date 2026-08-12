<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeMemoryEntry } from "@/api/interface/codeMemories"
import Icon from "@/components/common/Icon.vue"
import { codeMemoryMessages } from "../codeMemoryMessages"

const props = defineProps<{
	show: boolean
	entries: CodeMemoryEntry[]
	loading: boolean
	loadFailed: boolean
	saving: boolean
	removingId: number
}>()
const emit = defineEmits<{
	"update:show": [show: boolean]
	refresh: []
	add: [content: string, allProjects: boolean]
	remove: [id: number]
}>()
const { t } = useI18n({ messages: codeMemoryMessages })

const composing = ref(false)
const draft = ref("")
const allProjects = ref(false)

// 关掉抽屉时把草稿清掉：留着会让下次打开出现一段来路不明的文字。
watch(
	() => props.show,
	value => {
		if (!value) {
			composing.value = false
			draft.value = ""
			allProjects.value = false
		}
	},
)

function submit() {
	if (!draft.value.trim() || props.saving) return
	emit("add", draft.value.trim(), allProjects.value)
	draft.value = ""
	allProjects.value = false
	composing.value = false
}
</script>

<template>
	<n-drawer :show="show" placement="right" style="width: min(560px, 100vw)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.memoryTitle')" closable body-content-style="padding: 16px;">
			<div class="mb-3 flex items-center justify-between gap-3">
				<p class="text-xs text-slate-400">{{ t("code.memoryHint") }}</p>
				<div class="flex shrink-0 items-center gap-1">
					<n-button quaternary circle size="small" :loading="loading" :aria-label="t('code.memoryRefresh')" @click="emit('refresh')">
						<template #icon><Icon name="mdi:refresh" :size="16" /></template>
					</n-button>
					<n-button quaternary circle size="small" :aria-label="t('code.memoryAdd')" @click="composing = true">
						<template #icon><Icon name="mdi:plus" :size="18" /></template>
					</n-button>
				</div>
			</div>

			<div v-if="composing" class="mb-3 rounded-xl border border-slate-200/80 p-3">
				<n-input
					v-model:value="draft"
					type="textarea"
					:rows="2"
					:placeholder="t('code.memoryPlaceholder')"
					:disabled="saving"
					@keydown.enter.exact.prevent="submit"
				/>
				<div class="mt-2 flex items-center justify-between gap-2">
					<n-checkbox v-model:checked="allProjects" :disabled="saving" size="small">
						{{ t("code.memoryAllProjects") }}
					</n-checkbox>
					<div class="flex items-center gap-2">
						<n-button size="tiny" quaternary :disabled="saving" @click="composing = false">{{ t("code.cancel") }}</n-button>
						<n-button size="tiny" type="primary" :loading="saving" :disabled="!draft.trim()" @click="submit">
							{{ t("code.memoryAdd") }}
						</n-button>
					</div>
				</div>
			</div>

			<div v-if="loading && entries.length === 0" class="flex min-h-[200px] items-center justify-center">
				<n-spin size="small" />
			</div>
			<div v-else-if="loadFailed && entries.length === 0" class="flex min-h-[200px] flex-col items-center justify-center gap-2 text-xs text-red-500">
				<span>{{ t("code.memoryLoadFailed") }}</span>
				<n-button text type="primary" size="tiny" @click="emit('refresh')">{{ t("code.retry") }}</n-button>
			</div>
			<div v-else-if="entries.length === 0 && !composing" class="flex min-h-[200px] flex-col items-center justify-center gap-2 text-center">
				<div class="text-sm font-medium text-slate-600">{{ t("code.memoryEmptyTitle") }}</div>
				<p class="max-w-[280px] text-xs text-slate-400">{{ t("code.memoryEmptyHint") }}</p>
				<n-button size="small" class="mt-1" @click="composing = true">{{ t("code.memoryAddFirst") }}</n-button>
			</div>
			<!-- 顺序即后端返回的注入顺序，界面不再自己排 -->
			<div v-else class="divide-y divide-slate-100">
				<div v-for="entry in entries" :key="entry.id" class="group flex items-start gap-2 py-2.5">
					<p class="min-w-0 flex-1 text-sm leading-relaxed text-slate-700">{{ entry.content }}</p>
					<n-button
						quaternary
						circle
						size="tiny"
						class="shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
						:loading="removingId === entry.id"
						:aria-label="t('code.memoryRemoveLabel')"
						@click="emit('remove', entry.id)"
					>
						<template #icon><Icon name="mdi:close" :size="14" /></template>
					</n-button>
				</div>
			</div>
		</n-drawer-content>
	</n-drawer>
</template>
