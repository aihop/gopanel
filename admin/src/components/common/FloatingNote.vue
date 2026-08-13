<template>
	<section class="floating-note">
		<header class="floating-note__header">
			<div class="flex min-w-0 items-center gap-2">
				<Icon name="mdi:notebook-edit-outline" :size="18" />
				<span class="truncate font-medium">{{ t("floatingNote.title") }}</span>
				<span class="save-state" :class="{ 'is-dirty': dirty }">
					{{ dirty ? t("floatingNote.unsaved") : t("floatingNote.saved") }}
				</span>
			</div>
			<n-tooltip>
				<template #trigger>
					<n-button quaternary circle size="small" @click="emit('close')">
						<template #icon><Icon name="mdi:close" :size="17" /></template>
					</n-button>
				</template>
				{{ t("floatingNote.close") }}
			</n-tooltip>
		</header>

		<div class="floating-note__body">
			<n-spin v-if="loading" class="grow" />
			<n-alert v-else-if="loadError" type="error" :show-icon="false">
				<div class="flex items-center justify-between gap-2">
					<span>{{ t("floatingNote.loadFailed") }}</span>
					<n-button text type="primary" @click="loadNote">{{ t("floatingNote.retry") }}</n-button>
				</div>
			</n-alert>
			<n-input
				v-else
				v-model:value="content"
				type="textarea"
				class="floating-note__input"
				:placeholder="t('floatingNote.placeholder')"
				maxlength="20000"
			/>
			<n-alert v-if="saveError" type="error" :show-icon="false">{{ t("floatingNote.saveFailed") }}</n-alert>
		</div>

		<footer class="floating-note__footer">
			<span class="text-xs opacity-60">{{ t("floatingNote.length", { count: content.length }) }}</span>
			<n-button type="primary" size="small" :loading="saving" :disabled="loading || !dirty" @click="saveNote">
				{{ saving ? t("floatingNote.saving") : t("floatingNote.save") }}
			</n-button>
		</footer>
	</section>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import { userNoteGetAPI, userNoteSaveAPI } from "@/api/modules/userNote"
import { floatingNoteMessages } from "@/i18n/locales/floatingNote"
import { useMessage } from "naive-ui"
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"

const emit = defineEmits<{ close: [] }>()
const message = useMessage()
const { t } = useI18n({ messages: floatingNoteMessages })
const loading = ref(false)
const saving = ref(false)
const loadError = ref(false)
const saveError = ref(false)
const content = ref("")
const savedContent = ref("")
const dirty = computed(() => content.value !== savedContent.value)

async function loadNote() {
	loading.value = true
	loadError.value = false
	try {
		const response = await userNoteGetAPI()
		content.value = response.data?.content || ""
		savedContent.value = content.value
	} catch {
		loadError.value = true
	} finally {
		loading.value = false
	}
}

async function saveNote() {
	if (!dirty.value || saving.value) return
	const contentToSave = content.value
	saving.value = true
	saveError.value = false
	try {
		const response = await userNoteSaveAPI(contentToSave)
		savedContent.value = response.data.content
		message.success(t("floatingNote.saveSuccess"))
	} catch {
		saveError.value = true
	} finally {
		saving.value = false
	}
}

onMounted(loadNote)
</script>

<style scoped lang="scss">
.floating-note { display: flex; width: min(360px, calc(100vw - 32px)); height: min(420px, calc(100svh - 32px)); flex-direction: column; overflow: hidden; pointer-events: auto; border: 1px solid rgb(245 158 11 / 30%); border-radius: 16px; background: var(--n-color, var(--bg-default-color)); box-shadow: 0 20px 48px rgb(15 23 42 / 22%); }
.floating-note__header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 10px 10px 14px; border-bottom: 1px solid rgb(245 158 11 / 22%); background: rgb(254 243 199 / 86%); color: #78350f; }
.save-state { font-size: 11px; opacity: 0.65; &.is-dirty { color: #b45309; opacity: 1; } }
.floating-note__body { display: flex; min-height: 0; flex: 1; flex-direction: column; gap: 8px; padding: 12px; }
.floating-note__input { flex: 1; min-height: 0; :deep(textarea) { height: 100% !important; resize: none; line-height: 1.65; } :deep(.n-input-wrapper), :deep(.n-input__textarea) { height: 100%; } }
.floating-note__footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; border-top: 1px solid var(--border-color); }
</style>
