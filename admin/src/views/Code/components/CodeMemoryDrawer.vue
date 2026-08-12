<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeMemoryEntry, CodeMemorySetting } from "@/api/interface/codeMemories"
import type { AIProviderAccount } from "@/api/interface/aiAccounts"
import Icon from "@/components/common/Icon.vue"
import { codeMemoryMessages } from "../codeMemoryMessages"

const props = defineProps<{
	show: boolean
	entries: CodeMemoryEntry[]
	setting: CodeMemorySetting | null
	accounts: AIProviderAccount[]
	loading: boolean
	loadFailed: boolean
	saving: boolean
	savingSetting: boolean
	removingId: number
}>()
const emit = defineEmits<{
	"update:show": [show: boolean]
	refresh: []
	add: [content: string, allProjects: boolean]
	remove: [id: number]
	saveSetting: [value: { enabled: boolean; accountId: number; growthThreshold: number }]
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

// 抽取设置默认收起。它是一次性配置，日常打开这个抽屉是为了看记忆，
// 不该每次都被一堆输入框挡在前面。
const showSetting = ref(false)
const settingDraft = ref({ accountId: 0, growthThreshold: 8 })

watch(
	() => props.setting,
	value => {
		if (!value) return
		settingDraft.value = {
			accountId: value.accountId ?? 0,
			growthThreshold: value.growthThreshold ?? 8,
		}
	},
	{ immediate: true },
)

// 账号在系统设置里统一管理，这里只选用哪个。0 表示自动（按优先级挑）。
const accountOptions = computed(() => [
	{ label: t("code.memoryAccountAuto"), value: 0 },
	...props.accounts.map(account => ({ label: `${account.name} · ${account.model}`, value: account.id })),
])

function saveSetting(enabled: boolean) {
	emit("saveSetting", { enabled, ...settingDraft.value })
}
</script>

<template>
	<n-drawer :show="show" placement="right" style="width: min(560px, 100vw)" @update:show="emit('update:show', $event)">
		<n-drawer-content :title="t('code.memoryTitle')" closable body-content-style="padding: 16px;">
			<div class="mb-3 flex items-center justify-between gap-3">
				<p class="text-xs text-slate-400">{{ t("code.memoryHint") }}</p>
				<div class="flex shrink-0 items-center gap-1">
					<n-button quaternary circle size="small" :aria-label="t('code.memorySetting')" @click="showSetting = !showSetting">
						<template #icon><Icon name="mdi:cog-outline" :size="16" /></template>
					</n-button>
					<n-button quaternary circle size="small" :loading="loading" :aria-label="t('code.memoryRefresh')" @click="emit('refresh')">
						<template #icon><Icon name="mdi:refresh" :size="16" /></template>
					</n-button>
					<n-button quaternary circle size="small" :aria-label="t('code.memoryAdd')" @click="composing = true">
						<template #icon><Icon name="mdi:plus" :size="18" /></template>
					</n-button>
				</div>
			</div>

			<!-- 没启用时把提示顶到最前：这时候列表必然是空的，
			     不说清原因用户只会以为功能坏了 -->
			<n-alert v-if="setting && !setting.ready && !showSetting" type="warning" :show-icon="false" class="mb-3">
				<div class="text-xs font-medium">{{ t("code.memoryDisabledTitle") }}</div>
				<p class="mt-1 text-[11px] leading-relaxed opacity-80">{{ setting.readyReason || t("code.memoryDisabledHint") }}</p>
				<n-button size="tiny" class="mt-2" @click="showSetting = true">{{ t("code.memorySetting") }}</n-button>
			</n-alert>

			<div v-if="showSetting" class="mb-3 flex flex-col gap-2 rounded-xl border border-slate-200/80 p-3">
				<n-select v-model:value="settingDraft.accountId" size="small" :options="accountOptions" />
				<p class="text-[11px] text-slate-400">{{ t("code.memoryAccountHint") }}</p>
				<div class="flex items-center gap-2">
					<span class="shrink-0 text-[11px] text-slate-500">{{ t("code.memoryThreshold") }}</span>
					<n-input-number v-model:value="settingDraft.growthThreshold" size="small" :min="0" :max="100" class="w-24" />
				</div>
				<p class="text-[11px] text-slate-400">{{ t("code.memoryThresholdHint") }}</p>
				<div class="flex justify-end gap-2">
					<n-button size="tiny" quaternary :disabled="savingSetting" @click="showSetting = false">{{ t("code.cancel") }}</n-button>
					<n-button v-if="setting?.enabled" size="tiny" :loading="savingSetting" @click="saveSetting(false)">
						{{ t("code.disable") }}
					</n-button>
					<n-button size="tiny" type="primary" :loading="savingSetting" @click="saveSetting(true)">
						{{ setting?.enabled ? t("code.save") : t("code.memoryEnable") }}
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
