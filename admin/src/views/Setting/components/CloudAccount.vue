<template>
  <div class="mt-4">
    <div class="rounded-3xl border border-blue-100 bg-white p-6 shadow-sm">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Cloud Accounts</div>
          <div class="text-2xl font-semibold text-slate-900">云账号授权</div>
          <div class="text-sm leading-7 text-slate-500">
            统一管理各大云服务商的 API 凭据，授权后可用于 CDN 资源管理、通配符证书申请、DNS-01 验证以及自动续期等多种云资源操作。
          </div>
        </div>
        <n-space>
          <n-button
            type="primary"
            @click="openCreateModal"
          >新增授权</n-button>
          <n-button
            ghost
            @click="fetchData"
          >刷新</n-button>
        </n-space>
      </div>
    </div>

    <n-card class="mt-8 rounded-3xl shadow-sm">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="tableData"
        :bordered="false"
        :scroll-x="800"
      />
    </n-card>

    <n-modal
      :show="modalVisible"
      preset="card"
      :title="modalTitle"
      style="width: 600px;"
      @update:show="(val) => (modalVisible = val)"
    >
      <n-form
        :key="formRenderKey"
        ref="formRef"
        :model="form"
        :rules="rules"
        label-placement="top"
      >
        <n-form-item
          label="账户别名"
          path="name"
        >
          <n-input
            v-model:value="form.name"
            placeholder="例如：我的阿里云账号1"
          />
        </n-form-item>
        <n-form-item
          label="服务商"
          path="type"
        >
          <n-select
            v-model:value="form.type"
            :options="providerOptions"
            placeholder="请选择服务商"
          />
        </n-form-item>

        <div
          v-if="form.type"
          class="rounded-xl border border-slate-100 bg-slate-50 p-4 mb-4"
        >
          <div v-if="['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'aws', 'spaceship'].includes(form.type)">
            <n-form-item
              :label="accessKeyLabel"
              path="auth.accessKey"
            >
              <n-input
                v-model:value="form.auth.accessKey"
                :placeholder="accessKeyPlaceholder"
                autocomplete="off"
              />
            </n-form-item>
            <n-form-item
              :label="secretKeyLabel"
              path="auth.secretKey"
            >
              <n-input
                v-model:value="form.auth.secretKey"
                type="password"
                show-password-on="click"
                :placeholder="secretKeyPlaceholder"
                :input-props="{ autocomplete: 'new-password' }"
              />
              <div
                v-if="currentId && form.auth.secretKey"
                class="mt-2 text-xs text-slate-500"
              >
                已回填密钥，点击输入框右侧图标可查看明文
              </div>
            </n-form-item>
          </div>
          <div v-else-if="form.type === 'cloudflare'">
            <n-form-item
              label="API Token"
              path="auth.token"
            >
              <n-input
                v-model:value="form.auth.token"
                type="password"
                show-password-on="click"
                placeholder="请输入 API Token"
                :input-props="{ autocomplete: 'new-password' }"
              />
            </n-form-item>
          </div>
          <div v-else>
            <n-form-item
              label="认证信息 (JSON格式)"
              path="auth.raw"
            >
              <n-input
                v-model:value="form.auth.raw"
                type="textarea"
                placeholder='{"key": "value"}'
              />
            </n-form-item>
          </div>
        </div>
      </n-form>
      <template #action>
        <n-space justify="end">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleSave"
          >保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref, computed, nextTick } from "vue"
import { NButton, NTag, useDialog, useMessage, FormInst } from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import { CreateCloudAccount, DeleteCloudAccount, SearchCloudAccount, UpdateCloudAccount } from "@/api/modules/cloud"
import type { Website } from "@/api/interface/website"

const message = useMessage()
const dialog = useDialog()
const formRef = ref<FormInst | null>(null)

const loading = ref(false)
const submitting = ref(false)
const tableData = ref<Website.CloudAccount[]>([])

const modalVisible = ref(false)
const currentId = ref<number | null>(null)
const formRenderKey = ref(0)

const providerOptions = [
	{ label: "阿里云 (Aliyun)", value: "aliyun" },
	{ label: "腾讯云 (TencentCloud)", value: "tencentcloud" },
	{ label: "AWS (亚马逊云科技)", value: "aws" },
	{ label: "Cloudflare", value: "cloudflare" },
	{ label: "火山引擎 (Volcengine)", value: "volcengine" },
	{ label: "华为云 (HuaweiCloud)", value: "huaweicloud" },
	{ label: "Spaceship", value: "spaceship" }
]

const form = reactive({
	name: "",
	type: "aliyun",
	auth: {
		accessKey: "",
		secretKey: "",
		token: "",
		raw: ""
	}
})

const rules = {
	name: { required: true, message: "请输入账户别名", trigger: "blur" },
	type: { required: true, message: "请选择服务商", trigger: "change" }
}

const modalTitle = computed(() => (currentId.value ? "编辑云账号授权" : "新增云账号授权"))
const isTencentCloud = computed(() => form.type === "tencentcloud")
const isSpaceship = computed(() => form.type === "spaceship")
const accessKeyLabel = computed(() => (isTencentCloud.value ? "SecretId" : isSpaceship.value ? "API Key" : "Access Key ID"))
const accessKeyPlaceholder = computed(() => (isTencentCloud.value ? "请输入腾讯云 SecretId" : isSpaceship.value ? "请输入 Spaceship API Key" : "请输入 Access Key"))
const secretKeyLabel = computed(() => (isTencentCloud.value ? "SecretKey" : isSpaceship.value ? "API Secret" : "Secret Key"))
const secretKeyPlaceholder = computed(() => (isTencentCloud.value ? "请输入腾讯云 SecretKey" : isSpaceship.value ? "请输入 Spaceship API Secret" : "请输入 Secret Key"))

const columns: DataTableColumns<Website.CloudAccount> = [
	{ title: "ID", key: "id", width: 80 },
	{ title: "账户别名", key: "name", width: 200 },
	{
		title: "服务商",
		key: "type",
		width: 180,
		render(row) {
			const provider = providerOptions.find(p => p.value === row.type)
			return h(NTag, { type: "info", bordered: false, round: true }, { default: () => provider?.label || row.type })
		}
	},
	{ title: "创建时间", key: "createdAt", width: 180 },
	{
		title: "操作",
		key: "actions",
		width: 150,
		fixed: "right",
		render(row) {
			return h("div", { class: "flex gap-2" }, [
				h(NButton, { size: "small", text: true, type: "primary", onClick: () => openEditModal(row) }, { default: () => "编辑" }),
				h(NButton, { size: "small", text: true, type: "error", onClick: () => confirmDelete(row) }, { default: () => "删除" })
			])
		}
	}
]

async function fetchData() {
	loading.value = true
	try {
		const res = await SearchCloudAccount({ page: 1, limit: 100 } as any)
		tableData.value = Array.isArray(res.data?.items) ? res.data.items : []
	} finally {
		loading.value = false
	}
}

function resetAuthForm() {
	form.auth.accessKey = ""
	form.auth.secretKey = ""
	form.auth.token = ""
	form.auth.raw = ""
}

function parseAuthorization(authorization: Website.CloudAccount["authorization"]) {
	if (!authorization) return null
	if (typeof authorization === "string") {
		try {
			return JSON.parse(authorization)
		} catch (e) {
			return authorization
		}
	}
	return authorization
}

function resetForm() {
	currentId.value = null
	form.name = ""
	form.type = "aliyun"
	resetAuthForm()
}

function openCreateModal() {
	resetForm()
	formRenderKey.value += 1
	modalVisible.value = true
}

async function openEditModal(row: Website.CloudAccount) {
	currentId.value = row.id || null
	// 递增 key，准备彻底销毁重建表单
	formRenderKey.value += 1
	
	// 先修改类型，触发视图更新后再回填数据
	form.type = row.type
	form.name = row.name
	resetAuthForm()
	
	// 增加 tick 保证 DOM 分支切换后再填值
	await nextTick()

	try {
		const authObj: any = parseAuthorization(row.authorization)
		if (authObj) {
			if (row.type === 'cloudflare') {
				form.auth.token = authObj.token || ""
			} else if (row.type === 'spaceship') {
				form.auth.accessKey = authObj.apiKey || authObj.accessKey || ""
				form.auth.secretKey = authObj.apiSecret || authObj.secretKey || ""
			} else if (['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'aws'].includes(row.type)) {
				form.auth.accessKey = authObj.secretId || authObj.SecretId || authObj.accessKey || ""
				form.auth.secretKey = authObj.secretKey || authObj.SecretKey || ""
			} else {
				form.auth.raw = typeof authObj === "string" ? authObj : JSON.stringify(authObj, null, 2)
			}
		}
	} catch (e) {
		form.auth.raw = typeof row.authorization === 'string' ? row.authorization : JSON.stringify(row.authorization)
	}

	await nextTick()
	modalVisible.value = true
}

async function handleSave() {
	formRef.value?.validate(async errors => {
		if (errors) return
		
		submitting.value = true
		try {
			// 构建 authorization 对象
			let authObj: any = {}
			if (form.type === 'cloudflare') {
				authObj = { token: form.auth.token }
			} else if (form.type === 'spaceship') {
				authObj = { apiKey: form.auth.accessKey, apiSecret: form.auth.secretKey }
			} else if (['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'aws'].includes(form.type)) {
				authObj = form.type === "tencentcloud"
					? { secretId: form.auth.accessKey, accessKey: form.auth.accessKey, secretKey: form.auth.secretKey }
					: { accessKey: form.auth.accessKey, secretKey: form.auth.secretKey }
			} else {
				try {
					authObj = JSON.parse(form.auth.raw || "{}")
				} catch (e) {
					authObj = form.auth.raw || "{}"
				}
			}

			const payload = {
				name: form.name,
				type: form.type,
				authorization: authObj
			}

			if (currentId.value) {
				await UpdateCloudAccount({ ...payload, id: currentId.value })
				message.success("修改成功")
			} else {
				await CreateCloudAccount(payload)
				message.success("新增成功")
			}
			modalVisible.value = false
			await fetchData()
		} catch (error: any) {
			message.error(error.message || "保存失败")
		} finally {
			submitting.value = false
		}
	})
}

function confirmDelete(row: Website.CloudAccount) {
	dialog.warning({
		title: "确认删除",
		content: `确认要删除授权账户 "${row.name}" 吗？这可能会影响依赖该账户的相关服务（如 SSL 自动续期等）。`,
		onPositiveClick: async () => {
			await DeleteCloudAccount({ id: row.id! })
			message.success("删除成功")
			await fetchData()
		}
	})
}

onMounted(() => {
	fetchData()
})
</script>
