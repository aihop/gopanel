<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue"
import { NButton, NSpace, NSwitch, NTag, useDialog, useMessage } from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import { deleteAIProviderAccount, getAIProviderAccounts, saveAIProviderAccount } from "@/api/modules/code"
import type { AIProviderAccount, AIReasoningEffort } from "@/api/interface/aiAccounts"

const message = useMessage()
const dialog = useDialog()
const accounts = ref<AIProviderAccount[]>([])
const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()

const emptyForm = () => ({
	name: "",
	baseUrl: "",
	apiKey: "",
	model: "",
	enabled: true,
	useForMemoryExtraction: false,
	priority: 100,
	defaultReasoningEffort: "" as AIReasoningEffort,
})
const form = ref(emptyForm())

const reasoningOptions = [
	{ label: "不设置（由服务端默认）", value: "" },
	{ label: "低", value: "low" },
	{ label: "中", value: "medium" },
	{ label: "高", value: "high" },
]

const modalTitle = computed(() => (editingId.value ? "编辑 AI 账号" : "新增 AI 账号"))

async function fetchData() {
	loading.value = true
	try {
		const response = await getAIProviderAccounts()
		if (response.code === 0) accounts.value = response.data || []
	} catch {
		message.error("AI 账号加载失败")
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
		baseUrl: account.baseUrl,
		apiKey: "",
		model: account.model,
		enabled: account.enabled,
		useForMemoryExtraction: account.useForMemoryExtraction,
		priority: account.priority,
		defaultReasoningEffort: account.defaultReasoningEffort,
	}
	modalVisible.value = true
}

// 保存会实连探测一次。失败就不落库——这份凭据的主要消费者是后台的记忆抽取，
// 填错的话它会一直静默失败，保存时是唯一能让用户当场知道的时机。
async function submit() {
	if (!form.value.name.trim() || !form.value.baseUrl.trim() || !form.value.model.trim()) {
		message.warning("请填写账号名称、服务地址和模型名称")
		return
	}
	saving.value = true
	try {
		const response = await saveAIProviderAccount(form.value, editingId.value)
		if (response.code !== 0) throw new Error(response.message)
		message.success("已保存，连接测试通过")
		modalVisible.value = false
		await fetchData()
	} catch (error) {
		message.error(error instanceof Error && error.message ? error.message : "保存失败")
	} finally {
		saving.value = false
	}
}

function confirmDelete(account: AIProviderAccount) {
	dialog.warning({
		title: "删除 AI 账号",
		content: `删除「${account.name}」后，正在使用它的功能将无法调用模型。`,
		positiveText: "删除",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				await deleteAIProviderAccount(account.id)
				message.success("已删除")
				await fetchData()
			} catch {
				message.error("删除失败")
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
		items.push({ label: "推理强度", ok: account.supportsReasoningEffort })
	}
	return items
}

const columns: DataTableColumns<AIProviderAccount> = [
	{ title: "名称", key: "name", width: 150 },
	{ title: "模型", key: "model", width: 160 },
	{ title: "服务地址", key: "baseUrl", ellipsis: { tooltip: true } },
	{ title: "优先级", key: "priority", width: 80 },
	{
		title: "记忆抽取",
		key: "useForMemoryExtraction",
		width: 100,
		render: account =>
			h(
				NTag,
				{ size: "small", type: account.useForMemoryExtraction ? "success" : "default", bordered: false },
				{ default: () => (account.useForMemoryExtraction ? "已授权" : "未授权") },
			),
	},
	{
		title: "模型能力",
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
		title: "启用",
		key: "enabled",
		width: 80,
		render: account => h(NSwitch, { value: account.enabled, disabled: true, size: "small" }),
	},
	{
		title: "操作",
		key: "actions",
		width: 130,
		render: account =>
			h(NSpace, { size: 4 }, {
				default: () => [
					h(NButton, { size: "tiny", quaternary: true, onClick: () => openEditModal(account) }, { default: () => "编辑" }),
					h(NButton, { size: "tiny", quaternary: true, type: "error", onClick: () => confirmDelete(account) }, { default: () => "删除" }),
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
            AI Accounts
          </div>
          <div class="text-2xl font-semibold text-slate-900">
            AI 账号
          </div>
          <div class="text-sm leading-7 text-slate-500">
            配一次，面板内需要调用模型的功能都能复用。目前用于开发工作台的长期记忆抽取。
            保存时会实连测试一次，并探测该模型支持哪些参数。
          </div>
        </div>
        <n-space>
          <n-button type="primary" @click="openCreateModal">
            新增账号
          </n-button>
          <n-button ghost @click="fetchData">
            刷新
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
        <n-form-item label="账号名称">
          <n-input v-model:value="form.name" placeholder="例如：OpenAI 主账号" />
        </n-form-item>
        <n-form-item label="服务地址">
          <n-input v-model:value="form.baseUrl" placeholder="https://api.openai.com/v1" />
        </n-form-item>
        <n-form-item label="模型">
          <n-input v-model:value="form.model" placeholder="gpt-4o-mini" />
        </n-form-item>
        <n-form-item label="密钥">
          <n-input
            v-model:value="form.apiKey"
            type="password"
            show-password-on="click"
            :placeholder="editingId ? '留空保留当前密钥' : 'sk-...'"
          />
        </n-form-item>
        <n-form-item label="默认推理强度">
          <n-select v-model:value="form.defaultReasoningEffort" :options="reasoningOptions" />
          <template #feedback>
            仅在保存时探测确认该模型支持时才会生效
          </template>
        </n-form-item>
        <n-form-item label="优先级">
          <n-input-number v-model:value="form.priority" :min="0" :max="999" class="w-32" />
          <template #feedback>
            数字越小越优先。功能设为「自动」时按这个顺序挑账号
          </template>
        </n-form-item>
        <n-form-item label="启用">
          <n-switch v-model:value="form.enabled" />
        </n-form-item>
        <n-form-item label="允许用于记忆抽取">
          <n-switch v-model:value="form.useForMemoryExtraction" />
          <template #feedback>
            抽取会把整段会话记录发给该服务，需要单独授权
          </template>
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button :disabled="saving" @click="modalVisible = false">
            取消
          </n-button>
          <n-button type="primary" :loading="saving" @click="submit">
            保存并测试连接
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
