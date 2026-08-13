<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeMemoryAuditEvent } from "@/api/interface/codeMemories"
import { codeMemoryMessages } from "../codeMemoryMessages"

const props = defineProps<{
	summary: string
	auditEvents: CodeMemoryAuditEvent[]
	profileLoading: boolean
	profileLoadFailed: boolean
	auditLoading: boolean
	auditLoadFailed: boolean
	saving: boolean
}>()
const emit = defineEmits<{ refreshProfile: []; refreshAudit: []; save: [content: string]; clear: [] }>()
const { t } = useI18n({ messages: codeMemoryMessages })
const editing = ref(false)
const draft = ref("")

watch(
	() => props.summary,
	value => {
		if (editing.value && value === draft.value.trim()) editing.value = false
		if (!editing.value) draft.value = value
	},
	{ immediate: true }
)

function startEditing() {
	draft.value = props.summary
	editing.value = true
}

function cancelEditing() {
	draft.value = props.summary
	editing.value = false
}

function save() {
	const content = draft.value.trim()
	if (!content || props.saving) return
	if (content === props.summary.trim()) {
		editing.value = false
		return
	}
	emit("save", content)
}

function formatTime(value: string) {
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
	<n-collapse class="rounded-xl border border-slate-200/80 px-3 dark:border-[var(--border-color)]">
		<n-collapse-item name="profile">
			<template #header>
				<div>
					<div class="text-xs font-medium text-slate-600 dark:text-slate-300">
						{{ t("code.memoryProfile") }}
					</div>
					<div class="mt-0.5 text-[11px] text-slate-400">{{ t("code.memoryProfileHint") }}</div>
				</div>
			</template>
			<div v-if="profileLoading && !summary" class="flex h-20 items-center justify-center pb-3">
				<n-spin size="small" />
			</div>
			<div
				v-else-if="profileLoadFailed && !summary"
				class="flex h-20 flex-col items-center justify-center gap-2 pb-3 text-xs text-red-500"
			>
				<span>{{ t("code.memoryProfileLoadFailed") }}</span>
				<n-button text type="primary" size="tiny" @click="emit('refreshProfile')">
					{{ t("code.retry") }}
				</n-button>
			</div>
			<div v-else-if="editing" class="pb-3">
				<n-input
					v-model:value="draft"
					type="textarea"
					:rows="4"
					:disabled="saving"
					:placeholder="t('code.memoryProfilePlaceholder')"
				/>
				<div class="mt-2 flex justify-end gap-2">
					<n-button size="tiny" quaternary :disabled="saving" @click="cancelEditing">
						{{ t("code.cancel") }}
					</n-button>
					<n-button size="tiny" type="primary" :loading="saving" :disabled="!draft.trim()" @click="save">
						{{ t("code.save") }}
					</n-button>
				</div>
			</div>
			<div v-else class="pb-3">
				<p
					v-if="summary"
					class="whitespace-pre-wrap text-xs leading-relaxed text-slate-600 dark:text-slate-300"
				>
					{{ summary }}
				</p>
				<p v-else class="text-xs text-slate-400">{{ t("code.memoryProfileEmpty") }}</p>
				<div class="mt-2 flex justify-end gap-2">
					<n-button
						v-if="summary"
						size="tiny"
						quaternary
						type="error"
						:disabled="saving"
						@click="emit('clear')"
					>
						{{ t("code.clear") }}
					</n-button>
					<n-button size="tiny" secondary :disabled="saving" @click="startEditing">
						{{ summary ? t("code.edit") : t("code.memoryProfileAdd") }}
					</n-button>
				</div>
			</div>

			<div class="border-t border-slate-100 pb-3 pt-3 dark:border-[var(--border-color)]">
				<div class="mb-2 text-[11px] font-medium text-slate-500">{{ t("code.memoryAuditTitle") }}</div>
				<div v-if="auditLoading" class="flex h-12 items-center justify-center"><n-spin size="small" /></div>
				<div v-else-if="auditLoadFailed" class="flex items-center justify-between gap-2 text-xs text-red-500">
					<span>{{ t("code.memoryAuditLoadFailed") }}</span>
					<n-button text type="primary" size="tiny" @click="emit('refreshAudit')">
						{{ t("code.retry") }}
					</n-button>
				</div>
				<p v-else-if="auditEvents.length === 0" class="text-[11px] text-slate-400">
					{{ t("code.memoryAuditEmpty") }}
				</p>
				<div v-else class="max-h-40 space-y-2 overflow-y-auto">
					<div
						v-for="event in auditEvents"
						:key="event.id"
						class="rounded-lg bg-slate-50 p-2 dark:bg-white/5"
					>
						<div class="flex items-center justify-between gap-2 text-[11px]">
							<span class="font-medium text-slate-600 dark:text-slate-300">
								{{ t(`code.memoryAuditAction.${event.action}`) }}
							</span>
							<span class="shrink-0 text-slate-400">{{ formatTime(event.createdAt) }}</span>
						</div>
						<p class="mt-1 line-clamp-2 whitespace-pre-wrap text-[11px] text-slate-400">
							{{ event.after || event.before || t("code.memoryAuditCleared") }}
						</p>
					</div>
				</div>
			</div>
		</n-collapse-item>
	</n-collapse>
</template>
