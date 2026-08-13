<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { NButton, NSpace, NSwitch, NTag, useDialog, useMessage } from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import { deleteAIProviderAccount, getAIProviderAccounts, saveAIProviderAccount } from "@/api/modules/code"
import type { AIProviderAccount, AIProviderProtocol, AIReasoningEffort } from "@/api/interface/aiAccounts"
import { aiAccountMessages } from "./aiAccountMessages"

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n({ messages: aiAccountMessages })
const accounts = ref<AIProviderAccount[]>([])
const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()

const emptyForm = () => ({
	name: "",
	protocol: "openai_chat_completions" as AIProviderProtocol,
	baseUrl: "",
	apiKey: "",
	model: "",
	enabled: true,
	useForMemoryExtraction: false,
	useForSecurityAnalysis: false,
	priority: 100,
	defaultReasoningEffort: "" as AIReasoningEffort,
})
const form = ref(emptyForm())

const protocolOptions = computed(() => [
	{ label: t("aiAccount.protocolChat"), value: "openai_chat_completions" },
	{ label: t("aiAccount.protocolResponses"), value: "openai_responses" },
	{ label: t("aiAccount.protocolAnthropic"), value: "anthropic_messages" }
])
const reasoningOptions = computed(() => [
	{ label: t("aiAccount.reasoningNone"), value: "" },
	{ label: t("aiAccount.reasoningLow"), value: "low" },
	{ label: t("aiAccount.reasoningMedium"), value: "medium" },
	{ label: t("aiAccount.reasoningHigh"), value: "high" }
])
const isAnthropic = computed(() => form.value.protocol === "anthropic_messages")
const baseUrlPlaceholder = computed(() =>
	t(isAnthropic.value ? "aiAccount.baseUrlPlaceholderAnthropic" : "aiAccount.baseUrlPlaceholderOpenAI")
)
const modelPlaceholder = computed(() =>
	t(isAnthropic.value ? "aiAccount.modelPlaceholderAnthropic" : "aiAccount.modelPlaceholderOpenAI")
)
watch(isAnthropic, value => {
	if (value) form.value.defaultReasoningEffort = ""
})

const modalTitle = computed(() => t(editingId.value ? "aiAccount.editTitle" : "aiAccount.createTitle"))

async function fetchData() {
	loading.value = true
	try {
		const response = await getAIProviderAccounts()
		if (response.code === 0) accounts.value = response.data || []
	} catch {
		message.error(t("aiAccount.loadFailed"))
	} finally {
		loading.value = false
	}
}

function openCreateModal() {
	editingId.value = undefined
	form.value = emptyForm()
	modalVisible.value = true
}

function openEditModal(account: AIProviderAccount) {
	editingId.value = account.id
	form.value = {
		name: account.name,
		protocol: account.protocol || "openai_chat_completions",
		baseUrl: account.baseUrl,
		apiKey: "",
		model: account.model,
		enabled: account.enabled,
		useForMemoryExtraction: account.useForMemoryExtraction,
		useForSecurityAnalysis: account.useForSecurityAnalysis,
		priority: account.priority,
		defaultReasoningEffort: account.defaultReasoningEffort,
	}
	modalVisible.value = true
}

// 保存会实连探测一次。失败就不落库——这份凭据的主要消费者是后台的记忆抽取，
// 填错的话它会一直静默失败，保存时是唯一能让用户当场知道的时机。
async function submit() {
	if (!form.value.name.trim() || !form.value.baseUrl.trim() || !form.value.model.trim()) {
		message.warning(t("aiAccount.required"))
		return
	}
	saving.value = true
	try {
		const response = await saveAIProviderAccount(form.value, editingId.value)
		if (response.code !== 0) throw new Error(response.message)
		message.success(t("aiAccount.saveSuccess"))
		modalVisible.value = false
		await fetchData()
	} catch (error) {
		message.error(error instanceof Error && error.message ? error.message : t("aiAccount.saveFailed"))
	} finally {
		saving.value = false
	}
}

function confirmDelete(account: AIProviderAccount) {
	dialog.warning({
		title: t("aiAccount.deleteTitle"),
		content: t("aiAccount.deleteConfirm", { name: account.name }),
		positiveText: t("aiAccount.delete"),
		negativeText: t("aiAccount.cancel"),
		onPositiveClick: async () => {
			try {
				await deleteAIProviderAccount(account.id)
				message.success(t("aiAccount.deleted"))
				await fetchData()
			} catch {
				message.error(t("aiAccount.deleteFailed"))
			}
		},
	})
}

// 能力是探测出来的事实，只展示不可编辑——用户不该需要知道自己选的模型
// 支不支持 temperature，填错的代价是后台调用静默 400。
function capabilityTags(account: AIProviderAccount) {
	const items: Array<{ label: string; ok: boolean }> = [
		{ label: "temperature", ok: account.supportsTemperature },
		{ label: "json_schema", ok: account.supportsJsonSchema },
	]
	if (account.defaultReasoningEffort) {
		items.push({ label: t("aiAccount.reasoning"), ok: account.supportsReasoningEffort })
	}
	return items
}

const columns: DataTableColumns<AIProviderAccount> = [
	{ title: () => t("aiAccount.name"), key: "name", width: 150 },
	{ title: () => t("aiAccount.protocol"), key: "protocol", width: 190, render: account => h(NTag, { size: "small", bordered: false }, { default: () => t(`aiAccount.${account.protocol === "openai_responses" ? "protocolResponses" : account.protocol === "anthropic_messages" ? "protocolAnthropic" : "protocolChat"}`) }) },
	{ title: () => t("aiAccount.model"), key: "model", width: 160 },
	{ title: () => t("aiAccount.baseUrl"), key: "baseUrl", ellipsis: { tooltip: true } },
	{ title: () => t("aiAccount.priority"), key: "priority", width: 80 },
	{
		title: () => t("aiAccount.memory"),
		key: "useForMemoryExtraction",
		width: 100,
		render: account =>
			h(
				NTag,
				{ size: "small", type: account.useForMemoryExtraction ? "success" : "default", bordered: false },
				{ default: () => t(account.useForMemoryExtraction ? "aiAccount.authorized" : "aiAccount.unauthorized") },
			),
	},
	{
		title: () => t("securityMonitoring.aiAuthorization"),
		key: "useForSecurityAnalysis",
		width: 110,
		render: account =>
			h(
				NTag,
				{ size: "small", type: account.useForSecurityAnalysis ? "success" : "default", bordered: false },
				{ default: () => t(account.useForSecurityAnalysis ? "securityMonitoring.authorized" : "securityMonitoring.unauthorized") },
			),
	},
	{
		title: () => t("aiAccount.capabilities"),
		key: "capabilities",
		width: 220,
		render: account =>
			h(
				NSpace,
				{ size: 4 },
				{
					default: () =>
						capabilityTags(account).map(item =>
							h(
								NTag,
								{ size: "small", type: item.ok ? "info" : "warning", bordered: false },
								{ default: () => `${item.ok ? "✓" : "✗"} ${item.label}` },
							),
						),
				},
			),
	},
	{
		title: () => t("aiAccount.enabled"),
		key: "enabled",
		width: 80,
		render: account => h(NSwitch, { value: account.enabled, disabled: true, size: "small" }),
	},
	{
		title: () => t("aiAccount.actions"),
		key: "actions",
		width: 130,
		render: account =>
			h(NSpace, { size: 4 }, {
				default: () => [
					h(NButton, { size: "tiny", quaternary: true, onClick: () => openEditModal(account) }, { default: () => t("aiAccount.edit") }),
					h(NButton, { size: "tiny", quaternary: true, type: "error", onClick: () => confirmDelete(account) }, { default: () => t("aiAccount.delete") }),
				],
			}),
	},
]

onMounted(() => void fetchData())
</script>

<template>
  <div class="ai-account-root mt-4">
    <div class="rounded-3xl border border-blue-100 bg-white p-6 shadow-sm">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
			{{ t("aiAccount.eyebrow") }}
          </div>
          <div class="text-2xl font-semibold text-slate-900">
			{{ t("aiAccount.title") }}
          </div>
          <div class="text-sm leading-7 text-slate-500">
			{{ t("aiAccount.description") }}
          </div>
        </div>
        <n-space>
          <n-button type="primary" @click="openCreateModal">
			{{ t("aiAccount.add") }}
          </n-button>
          <n-button ghost @click="fetchData">
			{{ t("aiAccount.refresh") }}
          </n-button>
        </n-space>
      </div>
    </div>

    <n-card class="mt-8 rounded-3xl shadow-sm">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="accounts"
        :bordered="false"
        :scroll-x="1000"
      />
    </n-card>

    <n-modal
      :show="modalVisible"
      preset="card"
      :title="modalTitle"
      style="width: 600px;"
      @update:show="val => (modalVisible = val)"
    >
      <n-form label-placement="top">
		<n-form-item :label="t('aiAccount.accountName')">
		  <n-input v-model:value="form.name" :placeholder="t('aiAccount.accountNamePlaceholder')" />
        </n-form-item>
		<n-form-item :label="t('aiAccount.protocol')">
		  <n-select v-model:value="form.protocol" :options="protocolOptions" />
		  <template #feedback>{{ t("aiAccount.protocolHint") }}</template>
        </n-form-item>
		<n-form-item :label="t('aiAccount.baseUrl')">
		  <n-input v-model:value="form.baseUrl" :placeholder="baseUrlPlaceholder" />
        </n-form-item>
		<n-form-item :label="t('aiAccount.model')">
		  <n-input v-model:value="form.model" :placeholder="modelPlaceholder" />
		</n-form-item>
		<n-form-item :label="t('aiAccount.apiKey')">
          <n-input
            v-model:value="form.apiKey"
            type="password"
            show-password-on="click"
			:placeholder="editingId ? t('aiAccount.keepKey') : t('aiAccount.apiKeyPlaceholder')"
          />
        </n-form-item>
		<n-form-item :label="t('aiAccount.reasoning')">
		  <n-select v-model:value="form.defaultReasoningEffort" :options="reasoningOptions" :disabled="isAnthropic" />
          <template #feedback>
			{{ t(isAnthropic ? "aiAccount.reasoningUnavailable" : "aiAccount.reasoningHint") }}
          </template>
        </n-form-item>
		<n-form-item :label="t('aiAccount.priority')">
          <n-input-number v-model:value="form.priority" :min="0" :max="999" class="w-32" />
          <template #feedback>
			{{ t("aiAccount.priorityHint") }}
          </template>
        </n-form-item>
		<n-form-item :label="t('aiAccount.enabled')">
          <n-switch v-model:value="form.enabled" />
        </n-form-item>
		<n-form-item :label="t('aiAccount.memoryAuthorization')">
          <n-switch v-model:value="form.useForMemoryExtraction" />
          <template #feedback>
			{{ t("aiAccount.memoryAuthorizationHint") }}
          </template>
        </n-form-item>
		<n-form-item :label="t('securityMonitoring.aiAuthorization')">
		  <n-switch v-model:value="form.useForSecurityAnalysis" />
		  <template #feedback>
			{{ t("securityMonitoring.aiAuthorizationHint") }}
		  </template>
		</n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button :disabled="saving" @click="modalVisible = false">
			{{ t("aiAccount.cancel") }}
          </n-button>
          <n-button type="primary" :loading="saving" @click="submit">
			{{ t("aiAccount.saveAndProbe") }}
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
