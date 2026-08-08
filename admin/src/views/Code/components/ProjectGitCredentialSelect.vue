<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { CodeGitCredential } from "@/api/interface/code"
import { getCodeGitCredentials, saveCodeGitCredential } from "@/api/modules/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

const props = defineProps<{ modelValue: number }>()
const emit = defineEmits<{ "update:modelValue": [value: number] }>()
const { t } = useI18n({ messages: codeProjectMessages })
const message = useMessage()
const credentials = ref<CodeGitCredential[]>([])
const loading = ref(false)
const loadError = ref(false)
const showEditor = ref(false)
const saving = ref(false)
const editingId = ref<number>()
const form = ref({ name: "", username: "", secret: "" })

const options = computed(() => [
	{ label: t("code.gitCredentialSystem"), value: 0 },
	...credentials.value.map(item => ({ label: `${item.name} · ${item.username}`, value: item.id })),
])

async function load(notify = false) {
	loading.value = true
	try {
		const response = await getCodeGitCredentials()
		credentials.value = response.data || []
		loadError.value = false
	} catch (error) {
		loadError.value = true
	} finally {
		loading.value = false
	}
}

function openEditor() {
	const selected = credentials.value.find(item => item.id === props.modelValue)
	editingId.value = selected?.id
	form.value = { name: selected?.name || "", username: selected?.username || "", secret: "" }
	showEditor.value = true
}

async function save() {
	if (!form.value.name.trim() || !form.value.username.trim() || (!editingId.value && !form.value.secret.trim())) {
		message.warning(t("code.gitCredentialRequired"))
		return
	}
	saving.value = true
	try {
		const response = await saveCodeGitCredential({
			name: form.value.name.trim(), username: form.value.username.trim(), secret: form.value.secret,
		}, editingId.value)
		await load()
		emit("update:modelValue", response.data.id)
		showEditor.value = false
		message.success(t("code.gitCredentialSaved"))
	} catch (error) {
		void 0
	} finally {
		saving.value = false
	}
}

onMounted(() => void load())
</script>

<template>
	<div class="rounded-xl bg-[var(--n-color-embedded)] p-3">
		<div class="mb-2 flex items-center justify-between gap-3">
			<div>
				<div class="text-sm font-medium">{{ t("code.gitCredential") }}</div>
				<div class="mt-1 text-xs text-[var(--n-text-color-3)]">{{ t("code.gitCredentialHint") }}</div>
			</div>
			<n-button size="tiny" secondary @click="openEditor">{{ t("code.gitCredentialManage") }}</n-button>
		</div>
		<n-select
			:value="modelValue"
			:options="options"
			:loading="loading"
			@update:value="emit('update:modelValue', $event)"
		/>
		<n-alert v-if="loadError" class="mt-2" type="error" :show-icon="false">
			<div class="flex items-center justify-between gap-2">
				<span>{{ t("code.gitCredentialLoadFailed") }}</span>
				<n-button text size="tiny" @click="load(true)">{{ t("code.retry") }}</n-button>
			</div>
		</n-alert>
		<n-modal v-model:show="showEditor" preset="card" style="width: 520px" :title="t('code.gitCredentialEditor')">
			<div class="flex flex-col gap-3">
				<n-input v-model:value="form.name" :placeholder="t('code.gitCredentialName')" />
				<n-input v-model:value="form.username" :placeholder="t('code.gitCredentialUsername')" />
				<n-input
					v-model:value="form.secret"
					type="password"
					show-password-on="click"
					:placeholder="editingId ? t('code.gitCredentialSecretKeep') : t('code.gitCredentialSecret')"
				/>
				<n-alert type="info" :show-icon="false">{{ t("code.gitCredentialSecurity") }}</n-alert>
			</div>
			<template #footer>
				<div class="flex justify-end gap-2">
					<n-button @click="showEditor = false">{{ t("code.gitCredentialCancel") }}</n-button>
					<n-button type="primary" :loading="saving" @click="save">{{ t("code.gitCredentialSave") }}</n-button>
				</div>
			</template>
		</n-modal>
	</div>
</template>
