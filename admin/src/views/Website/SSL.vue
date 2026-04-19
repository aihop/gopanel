<template>
  <div class="mt-4">
    <div class="px-4">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Security</div>
          <div class="text-2xl font-semibold fg-base-100">{{ $t('ssl.sslManagement') }}</div>
          <div class="text-sm leading-7 text-slate-500">
            {{ $t('ssl.sslManagementHelper') }}
          </div>
        </div>
        <n-space>
          <n-button
            type="primary"
            @click="openCloudApplyModal"
          >{{ $t('ssl.cdnApply') }}</n-button>
          <n-button
            ghost
            type="primary"
            @click="openSyncModal"
          >{{ $t('ssl.sync') }}</n-button>
          <n-button
            ghost
            type="primary"
            @click="openUploadModal"
          >{{ $t('ssl.upload') }}</n-button>
          <n-button
            ghost
            @click="fetchData"
          >{{ $t('ssl.refresh') }}</n-button>
        </n-space>
      </div>
    </div>

    <n-card class="mt-8 rounded-3xl shadow-sm">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="tableData"
        :bordered="false"
        :scroll-x="1200"
      />
    </n-card>

    <n-modal
      v-model:show="cloudApplyModalVisible"
      preset="card"
      title="云账号签注"
      style="width: 650px;"
    >
      <div class="space-y-6">
        <!-- 步骤 1 -->
        <div class="rounded-xl border border-blue-100 bg-blue-50/50 p-5 transition-all">
          <div class="mb-3 flex items-center gap-3">
            <div class="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-sm font-bold text-white">1</div>
            <div class="text-base font-bold text-slate-800">选择业务服务商</div>
          </div>
          <div class="mb-4 ml-9 text-sm text-slate-500">
            该域名实际部署在哪个云服务商（如阿里云 CDN、OSS）？系统将尝试拉取您的已有业务域名，并且未来支持证书到期自动推送到该云资源。
          </div>
          <div class="ml-9">
            <n-select
              :value="cloudApplyForm.cdnAccountId"
              :options="cloudAccountOptions"
              filterable
              placeholder="请选择部署该域名的业务云账号"
              @update:value="handleCdnAccountChange"
            />
          </div>
        </div>

        <!-- 步骤 2 -->
        <div
          class="rounded-xl border p-5 transition-all duration-300"
          :class="cloudApplyForm.cdnAccountId ? 'border-blue-100 bg-blue-50/30' : 'border-slate-100 bg-slate-50 opacity-50 grayscale pointer-events-none'"
        >
          <div class="mb-3 flex items-center gap-3">
            <div
              class="flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold text-white"
              :class="cloudApplyForm.cdnAccountId ? 'bg-blue-600' : 'bg-slate-300'"
            >
              2
            </div>
            <div class="text-base font-bold text-slate-800">选择或填写证书域名</div>
          </div>
          <div class="mb-4 ml-9 text-sm text-slate-500">
            您可以直接在下拉列表中选择从云服务商拉取的 CDN 域名，或手动输入（如 *.example.com）。
          </div>
          <div class="ml-9">
            <n-grid
              :cols="2"
              :x-gap="16"
            >
              <n-form-item-gi :label="$t('website.primaryDomain')">
                <n-auto-complete
                  v-model:value="cloudApplyForm.primaryDomain"
                  :options="cdnDomainsOptions"
                  :loading="cdnDomainsLoading"
                  placeholder="选择CDN域名或输入 *.example.com"
                  :get-show="() => true"
                  clearable
                />
              </n-form-item-gi>
              <n-form-item-gi label="其他域名 (可选)">
                <n-input
                  v-model:value="cloudApplyForm.otherDomains"
                  placeholder="www.example.com"
                />
              </n-form-item-gi>
            </n-grid>
            <n-form-item label="备注">
              <n-input
                v-model:value="cloudApplyForm.description"
                placeholder="例如：主站通配符证书"
              />
            </n-form-item>
          </div>
        </div>

        <!-- 步骤 3 -->
        <div
          class="rounded-xl border p-5 transition-all duration-300"
          :class="cloudApplyForm.primaryDomain ? 'border-blue-100 bg-blue-50/30' : 'border-slate-100 bg-slate-50 opacity-50 grayscale pointer-events-none'"
        >
          <div class="mb-3 flex items-center gap-3">
            <div
              class="flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold text-white"
              :class="cloudApplyForm.primaryDomain ? 'bg-blue-600' : 'bg-slate-300'"
            >
              3
            </div>
            <div class="text-base font-bold text-slate-800">授权 DNS 解析服务商 (可选)</div>
          </div>
          <div class="mb-4 ml-9 text-sm text-slate-500">
            如果该域名的 DNS 解析与业务不在同一个云厂商（如解析在 Cloudflare），请在此选择解析账号用于签发验证。<br />
            <span class="text-blue-500">若不选择，系统将默认使用上方第 1 步的业务服务商进行 DNS 验证。</span>
          </div>
          <div class="ml-9">
            <n-select
              v-model:value="cloudApplyForm.cloudAccountId"
              :options="cloudAccountOptions"
              filterable
              clearable
              placeholder="选择管理该域名解析的云账号（不选则默认使用上方账号）"
            />
          </div>
        </div>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="cloudApplyModalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            :disabled="!cloudApplyForm.cdnAccountId || !cloudApplyForm.primaryDomain"
            @click="handleCloudApplyCertificate"
          >提交签发</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="syncModalVisible"
      preset="card"
      style="width: 420px;"
      title="同步网站证书"
      @update:show="handleSyncModalChange"
    >
      <div class="space-y-4">
        <div class="text-sm leading-7 text-slate-500">
          选择一个已经启用的网站，系统会到默认证书存储目录中读取该域名的已签发证书，并同步到面板中以便查看和下载。
        </div>
        <n-form label-placement="top">
          <n-form-item label="选择网站">
            <n-select
              :value="syncForm.websiteId"
              :options="websiteOptions"
              filterable
              placeholder="请选择要同步证书的网站"
              @update:value="handleSyncWebsiteChange"
            />
          </n-form-item>
        </n-form>
      </div>
      <template #action>
        <n-space justify="end">
          <n-button @click="syncModalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleSyncCertificate"
          >开始同步</n-button>
        </n-space>
      </template>
    </n-modal>

    <FullModal
      :show="uploadModalVisible"
      title="上传证书"
      class="max-w-[760px]"
      @update:show="handleUploadModalChange"
    >
      <n-form
        label-placement="top"
        class="space-y-2"
      >
        <n-grid
          :cols="2"
          :x-gap="16"
        >
          <n-form-item-gi :label="$t('website.primaryDomain')">
            <n-input
              v-model:value="uploadForm.primaryDomain"
              placeholder="example.com"
              @update:value="handleUploadPrimaryDomainChange"
            />
          </n-form-item-gi>
          <n-form-item-gi label="其他域名">
            <n-input
              v-model:value="uploadForm.otherDomains"
              placeholder="www.example.com, api.example.com"
              @update:value="handleUploadOtherDomainsChange"
            />
          </n-form-item-gi>
        </n-grid>
        <n-form-item label="备注">
          <n-input
            v-model:value="uploadForm.description"
            placeholder="例如：商业证书 / 迁移证书"
            @update:value="handleUploadDescriptionChange"
          />
        </n-form-item>
        <n-form-item label="证书内容 PEM">
          <n-input
            v-model:value="uploadForm.pem"
            type="textarea"
            :autosize="{ minRows: 8, maxRows: 12 }"
            @update:value="handleUploadPemChange"
          />
        </n-form-item>
        <n-form-item label="私钥内容 KEY">
          <n-input
            v-model:value="uploadForm.privateKey"
            type="textarea"
            :autosize="{ minRows: 8, maxRows: 12 }"
            @update:value="handleUploadPrivateKeyChange"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="uploadModalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleUploadCertificate"
          >保存</n-button>
        </n-space>
      </template>
    </FullModal>

    <FullModal
      :show="detailModalVisible"
      Z
      :title="detailTitle"
      maxHeight="660px"
      @update:show="handleDetailModalChange"
    >
      <div
        v-if="currentSSL"
        class="space-y-5"
      >
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs uppercase tracking-[0.18em] text-slate-400">{{ t("website.primaryDomain") }}</div>
            <div class="mt-2 text-base font-semibold fg-base-100">{{ currentSSL.primaryDomain }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs uppercase tracking-[0.18em] text-slate-400">签发来源</div>
            <div class="mt-2 text-base font-semibold fg-base-100">
              <n-tag
                :type="sourceLabel(currentSSL).tagType"
                :bordered="false"
                round
              >
                {{ sourceLabel(currentSSL).label }}
              </n-tag>
            </div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs uppercase tracking-[0.18em] text-slate-400">颁发者</div>
            <div class="mt-2 text-base font-semibold fg-base-100">
              {{ currentSSL.organization || "--" }}
            </div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="text-xs uppercase tracking-[0.18em] text-slate-400">到期时间</div>
            <div class="mt-2 text-base font-semibold fg-base-100">
              {{ formatDateTime(currentSSL.expireDate) }}
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium text-slate-700">证书内容</div>
              <n-button
                text
                type="primary"
                @click="downloadContent(currentSSL.pem, `${currentSSL.primaryDomain}.crt`)"
              >
                下载 CRT
              </n-button>
            </div>
            <n-input
              v-model:value="currentSSL.pem"
              type="textarea"
              readonly
              :autosize="{ minRows: 10, maxRows: 16 }"
            />
          </div>
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium text-slate-700">私钥内容</div>
              <n-button
                text
                type="primary"
                @click="downloadContent(currentSSL.privateKey, `${currentSSL.primaryDomain}.key`)"
              >
                下载 KEY
              </n-button>
            </div>
            <n-input
              v-model:value="currentSSL.privateKey"
              type="textarea"
              readonly
              :autosize="{ minRows: 10, maxRows: 16 }"
            />
          </div>
        </div>
      </div>
    </FullModal>

    <n-modal
      v-model:show="applyModalVisible"
      preset="card"
      title="绑定到网站"
      class="max-w-[560px]"
      @on-update:show="handleApplyModalChange"
    >
      <div class="space-y-4">
        <div class="text-sm leading-7 text-slate-500">
          手动上传证书可以绑定到指定网站，系统会为该网站写入 `tls cert key` 配置。Caddy 自动 HTTPS
          类型无需手动绑定。
        </div>
        <n-form label-placement="top">
          <n-form-item label="选择网站">
            <n-select
              :value="applyForm.websiteId"
              :options="websiteOptions"
              filterable
              placeholder="请选择要绑定的网站"
              @update:value="handleApplyWebsiteChange"
            />
          </n-form-item>
        </n-form>
      </div>
      <template #action>
        <n-space justify="end">
          <n-button @click="applyModalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleApplyCertificate"
          >确认绑定</n-button>
        </n-space>
      </template>
    </n-modal>
    <n-modal
      v-model:show="pushCDNModalVisible"
      preset="card"
      title="推送到云厂商 CDN"
      style="width: 520px;"
    >
      <div class="space-y-4">
        <div class="text-sm leading-7 text-slate-500">
          将当前证书 ({{ currentPushSSL?.primaryDomain }}) 推送到云厂商，以便 CDN、WAF 或对象存储等云产品使用最新的证书内容。
        </div>
        <n-form
          label-placement="top"
          class="space-y-2"
        >
          <n-form-item label="选择云账号">
            <n-select
              v-model:value="pushCDNForm.cloudAccountId"
              :options="cloudAccountOptions"
              filterable
              placeholder="请选择授权的云账号"
            />
          </n-form-item>

          <n-form-item label="目标域名（选填）">
            <n-input
              v-model:value="pushCDNForm.targetDomain"
              placeholder="如不填，默认使用证书的主域名"
            />
          </n-form-item>
        </n-form>
      </div>
      <template #action>
        <n-space justify="end">
          <n-button @click="pushCDNModalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handlePushCDN"
          >开始推送</n-button>
        </n-space>
      </template>
    </n-modal>
    <n-modal
      v-model:show="pushRuleModalVisible"
      preset="card"
      title="云端部署规则管理"
      style="width: 700px;"
      @update:show="handlePushRuleModalChange"
    >
      <div class="space-y-4">
        <div class="text-sm leading-7 text-slate-500">
          管理当前证书 ({{ currentPushSSL?.primaryDomain }}) 的云端自动部署规则。当证书自动续签更新后，系统会自动推送到下方配置的云厂商资源中。
        </div>

        <n-data-table
          :loading="pushRuleLoading"
          :columns="pushRuleColumns"
          :data="pushRuleData"
          :bordered="false"
          :scroll-x="600"
        />

        <n-divider class="!my-2" />

        <n-form
          label-placement="top"
          class="space-y-2 rounded-xl border border-slate-100 bg-slate-50 p-4"
        >
          <div class="mb-2 text-sm font-semibold">新增 / 编辑规则</div>
          <n-grid
            :cols="2"
            :x-gap="16"
          >
            <n-form-item-gi label="选择云账号">
              <n-select
                v-model:value="pushRuleForm.cloudAccountId"
                :options="cloudAccountOptions"
                filterable
                placeholder="请选择授权的云账号"
              />
            </n-form-item-gi>
            <n-form-item-gi label="目标域名（选填）">
              <n-input
                v-model:value="pushRuleForm.targetDomain"
                placeholder="如不填，默认使用证书主域名"
              />
            </n-form-item-gi>
          </n-grid>
          <n-space justify="end">
            <n-button
              v-if="pushRuleForm.id"
              @click="resetPushRuleForm"
            >取消编辑</n-button>
            <n-button
              type="primary"
              :loading="submitting"
              @click="handleSavePushRule"
            >保存规则</n-button>
          </n-space>
        </n-form>
      </div>
    </n-modal>
    <n-modal
      v-model:show="logModalVisible"
      preset="card"
      title="SSL自动签证日志"
      style="width: 900px;"
      @update:show="handleLogModalChange"
    >
      <div class="h-[400px] overflow-y-auto rounded-xl bg-slate-900 p-4 font-mono text-sm text-green-400">
        <div
          v-if="logsData.length === 0"
          class="text-slate-500"
        >正在建立连接...</div>
        <div
          v-for="(log, index) in logsData"
          :key="index"
          class="mb-1 break-words"
        >
          {{ log }}
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, SelectOption, AutoCompleteOption } from "naive-ui"
import {
	ApplySSL,
	CreateSSL,
	DeleteSSL,
	GetSSL,
	ObtainSSL,
	PushToCDNAPI,
	RenewSSL,
	SearchSSLPushRule,
	CreateSSLPushRule,
	UpdateSSLPushRule,
	DeleteSSLPushRule,
	SSLSearchAPI
} from "@/api/modules/ssl"
import { ListWebsitesAPI } from "@/api/modules/website"
import { CloudCdnDomainsAPI, SearchCloudAccount } from "@/api/modules/cloud"
import { NButton, NDropdown, NTag, useDialog, useMessage, NAutoComplete } from "naive-ui"
import { computed, h, onMounted, reactive, ref } from "vue"
import { useAuthStore } from "@/store/auth"
import type { Website } from "@/api/interface/website"
import { t } from "@/i18n"
import FullModal from "@/components/FullModal.vue"

type WebsiteOption = {
	id: number
	name?: string
	primaryDomain?: string
}

type SSLRow = {
	id: number
	primaryDomain: string
	domains: string
	type: string
	provider: string
	organization: string
	expireDate: string
	startDate: string
	description: string
	privateKey: string
	pem: string
	status: string
	cloudAccountId: number
	dnsAccountId: number
	websites?: Array<{ id: number; name?: string; primaryDomain?: string }>
}

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const tableData = ref<SSLRow[]>([])
const websites = ref<WebsiteOption[]>([])
const cloudAccounts = ref<Array<{ id: number; name: string; type: string }>>([])

const syncModalVisible = ref(false)
const uploadModalVisible = ref(false)
const detailModalVisible = ref(false)
const applyModalVisible = ref(false)
const pushCDNModalVisible = ref(false)
const cloudApplyModalVisible = ref(false)
const pushRuleModalVisible = ref(false)
const logModalVisible = ref(false)

const pushRuleLoading = ref(false)
const pushRuleData = ref<Website.SSLPushRule[]>([])

const cdnDomainsLoading = ref(false)
const cdnDomainsOptions = ref<Array<{ label: string; value: string }>>([])

const currentSSL = ref<SSLRow | null>(null)
const currentApplySSLId = ref<number | null>(null)
const currentPushSSL = ref<SSLRow | null>(null)
const currentLogSSLId = ref<number | null>(null)
const logsData = ref<string[]>([])
let logEventSource: EventSource | null = null

const cloudApplyForm = reactive({
	primaryDomain: "",
	otherDomains: "",
	cloudAccountId: null as number | null, // 用于 DNS-01 签发验证的云账号
	cdnAccountId: null as number | null, // 仅用于拉取 CDN 域名列表的云账号
	description: ""
})

async function handleCdnAccountChange(value: number | null) {
	cloudApplyForm.cdnAccountId = value
	cloudApplyForm.primaryDomain = "" // 重置已选的域名
	cdnDomainsOptions.value = []
	if (!value) return

	cdnDomainsLoading.value = true
	try {
		const res = await CloudCdnDomainsAPI(value)
		cdnDomainsOptions.value = (res.data || []).map(domain => ({
			label: domain,
			value: domain
		})) as any
		if (cdnDomainsOptions.value.length === 0) {
			message.info("该云账号下暂无CDN域名")
		}
	} catch (error: any) {
		message.error(error.message || "拉取CDN域名失败，请检查账号配置")
	} finally {
		cdnDomainsLoading.value = false
	}
}

const pushCDNForm = reactive({
	cloudAccountId: null as number | null,
	targetDomain: ""
})

const pushRuleForm = reactive({
	id: null as number | null,
	cloudAccountId: null as number | null,
	targetDomain: ""
})

const handlePushRuleModalChange = (value: boolean) => {
	pushRuleModalVisible.value = value
}

async function fetchPushRules() {
	if (!currentPushSSL.value?.id) return
	pushRuleLoading.value = true
	try {
		const res = await SearchSSLPushRule({
			page: 1,
			limit: 100,
			wheres: [{ column: "ssl_id", value: currentPushSSL.value.id, operator: "=" }]
		} as any)
		pushRuleData.value = Array.isArray(res.data?.items) ? res.data.items : []
	} finally {
		pushRuleLoading.value = false
	}
}

function resetPushRuleForm() {
	pushRuleForm.id = null
	pushRuleForm.cloudAccountId = null
	pushRuleForm.targetDomain = ""
}

async function handleSavePushRule() {
	if (!pushRuleForm.cloudAccountId) {
		message.warning("请选择云账号")
		return
	}
	submitting.value = true
	try {
		if (pushRuleForm.id) {
			await UpdateSSLPushRule({
				id: pushRuleForm.id,
				cloudAccountId: pushRuleForm.cloudAccountId,
				targetDomain: pushRuleForm.targetDomain
			})
			message.success("部署规则已更新")
		} else {
			await CreateSSLPushRule({
				sslId: currentPushSSL.value!.id,
				cloudAccountId: pushRuleForm.cloudAccountId,
				targetDomain: pushRuleForm.targetDomain
			})
			message.success("部署规则已新增")
		}
		resetPushRuleForm()
		await fetchPushRules()
	} catch (error: any) {
		message.error(error.message || "保存失败")
	} finally {
		submitting.value = false
	}
}

function editPushRule(row: Website.SSLPushRule) {
	pushRuleForm.id = row.id
	pushRuleForm.cloudAccountId = row.cloudAccountId
	pushRuleForm.targetDomain = row.targetDomain
}

function deletePushRule(row: Website.SSLPushRule) {
	dialog.warning({
		title: "确认删除部署规则？",
		content: "删除后，证书续签将不再自动推送到此目标资源。",
		onPositiveClick: async () => {
			await DeleteSSLPushRule({ id: row.id })
			message.success("删除成功")
			await fetchPushRules()
		}
	})
}

const pushRuleColumns: DataTableColumns<Website.SSLPushRule> = [
	{
		title: "云账号",
		key: "cloudAccountId",
		render(row) {
			const account = cloudAccounts.value.find(item => item.id === row.cloudAccountId)
			return account ? `${account.name} (${account.type})` : `未知账号 ID: ${row.cloudAccountId}`
		}
	},
	{
		title: "目标域名",
		key: "targetDomain",
		render(row) {
			return row.targetDomain || "（默认主域名）"
		}
	},
	{
		title: "操作",
		key: "actions",
		width: 140,
		render(row) {
			return h("div", { class: "flex gap-2" }, [
				h(NButton, { size: "small", text: true, type: "primary", onClick: () => editPushRule(row) }, { default: () => "编辑" }),
				h(NButton, { size: "small", text: true, type: "error", onClick: () => deletePushRule(row) }, { default: () => "删除" })
			])
		}
	}
]

const syncForm = reactive({
	websiteId: null as number | null
})

const applyForm = reactive({
	websiteId: null as number | null
})

const uploadForm = reactive({
	primaryDomain: "",
	otherDomains: "",
	description: "",
	pem: "",
	privateKey: ""
})

const handleSyncModalChange = (value: boolean) => {
	syncModalVisible.value = value
}

const handleSyncWebsiteChange = (value: number | null) => {
	syncForm.websiteId = value
}

const handleUploadModalChange = (value: boolean) => {
	uploadModalVisible.value = value
}

const handleUploadPrimaryDomainChange = (value: string) => {
	uploadForm.primaryDomain = value
}

const handleUploadOtherDomainsChange = (value: string) => {
	uploadForm.otherDomains = value
}

const handleUploadDescriptionChange = (value: string) => {
	uploadForm.description = value
}

const handleUploadPemChange = (value: string) => {
	uploadForm.pem = value
}

const handleUploadPrivateKeyChange = (value: string) => {
	uploadForm.privateKey = value
}

const handleDetailModalChange = (value: boolean) => {
	detailModalVisible.value = value
}

const handleApplyModalChange = (value: boolean) => {
	applyModalVisible.value = value
}

const handleApplyWebsiteChange = (value: number | null) => {
	applyForm.websiteId = value
}

const websiteOptions = computed<SelectOption[]>(() =>
	websites.value.map(item => ({
		label: item.name ? `${item.name} · ${item.primaryDomain || "--"}` : item.primaryDomain || `#${item.id}`,
		value: item.id
	}))
)

const cloudAccountOptions = computed<SelectOption[]>(() =>
	cloudAccounts.value.map(item => ({
		label: `${item.name} (${item.type})`,
		value: item.id
	}))
)

const detailTitle = computed(() => (currentSSL.value ? `${currentSSL.value.primaryDomain} 证书详情` : "证书详情"))

const columns: DataTableColumns<SSLRow> = [
	{
		title: t("website.primaryDomain"),
		key: "primaryDomain",
		width: 150
	},
	{
		title: t("ssl.domainList"),
		key: "domains",
		render(row) {
			const domains = (row.domains || "")
				.split(",")
				.map(item => item.trim())
				.filter(Boolean)
			return h(
				"div",
				{ class: "flex flex-wrap gap-2" },
				domains.length
					? domains.map(domain =>
							h("span", { class: "rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-600" }, domain)
						)
					: [h("span", { class: "text-slate-400" }, "--")]
			)
		}
	},
	{
		title: t("ssl.source"),
		key: "type",
		width: 160,
		render(row) {
			const info = sourceLabel(row)
			return h(
				NTag,
				{ type: info.tagType, bordered: false, round: true,size:"small" },
				{ default: () => info.label }
			)
		}
	},
	{
		title: t("ssl.bindWebsite"),
		key: "websites",
		render(row) {
			const items = row.websites || []
			if (!items.length) {
				return h("span", { class: "text-slate-400" }, "未绑定")
			}
			return h(
				"div",
				{ class: "flex flex-wrap gap-2" },
				items.map(item =>
					h(
						"span",
						{ class: "rounded-full bg-blue-50 px-3 py-1 text-xs text-blue-600" },
						item.name || item.primaryDomain || `#${item.id}`
					)
				)
			)
		}
	},
	{
		title: t("ssl.organization"),
		key: "organization",
		width: 130,
		render(row) {
			return row.organization || "--"
		}
	},
	{
		title: t("license.expiresAt"),
		key: "expireDate",
		width: 180,
		render(row) {
			return formatDateTime(row.expireDate)
		}
	},
	{
		title: t("commons.table.status"),
		key: "status",
		width: 80,
		fixed: "right",
		render(row) {
			const expired = isExpired(row.expireDate)
			return h(
				NTag,
				{ type: expired ? "error" : "success", bordered: false, round: true },
				{ default: () => (expired ? "已过期" : "有效") }
			)
		}
	},
	{
		title: t("commons.table.operate"),
		key: "actions",
		width: 220,
		fixed: "right",
		render(row) {
			const moreOptions = [
				{ label: "下载证书", key: "download-crt" },
				{ label: "下载私钥", key: "download-key" },
				{ label: "删除", key: "delete" }
			]
			if (row.type === "upload") {
				moreOptions.splice(2, 0, { label: "绑定网站", key: "apply" })
			}

			const expired = isExpired(row.expireDate)
			if (expired && row.type !== "upload") {
				moreOptions.unshift({ label: "立即签注", key: "renew" })
			}

			if (row.status === "pending") {
				moreOptions.unshift({ label: "查看日志", key: "view-log" })
			}

			return h(
				"div",
				{ class: "flex items-center gap-2" },
				[
					h(
						NButton,
						{ size: "small", text: true, type: "primary", onClick: () => openDetail(row.id) },
						{ default: () => "详情" }
					),
					h(
						NButton,
						{ size: "small", text: true, type: "primary", onClick: () => openPushCDNModal(row) },
						{ default: () => "推送CDN" }
					),
					h(
						NButton,
						{ size: "small", text: true, type: "primary", onClick: () => openPushRuleModal(row) },
						{ default: () => "自动部署" }
					),
					h(
						NDropdown,
						{
							options: moreOptions,
							onSelect: (key: string) => {
								switch (key) {
									case "view-log":
										openLogModal(row.id)
										break
									case "renew":
										handleRenewCertificate(row)
										break
									case "download-crt":
										downloadContent(row.pem, `${row.primaryDomain}.crt`)
										break
									case "download-key":
										downloadContent(row.privateKey, `${row.primaryDomain}.key`)
										break
									case "apply":
										openApplyModal(row.id)
										break
									case "delete":
										confirmDelete(row)
										break
								}
							}
						},
						{
							default: () =>
								h(
									NButton,
									{ size: "small", text: true, type: "primary" },
									{ default: () => "更多" }
								)
						}
					)
				]
			)
		}
	}
]

function sourceLabel(row: SSLRow | { type: string }) {
	if (row.type === "caddy") return { label: "Caddy 自动 HTTPS", tagType: "success" } as const
	if (row.type === "upload") return { label: "手动上传", tagType: "warning" } as const
	if (row.type.startsWith("dns-")) {
		const providerMap: Record<string, string> = {
			aliyun: "阿里云",
			tencentcloud: "腾讯云",
			cloudflare: "Cloudflare",
			volcengine: "火山引擎",
			huaweicloud: "华为云"
		}
		const provider = row.type.replace("dns-", "")
		const name = providerMap[provider] || provider
		return { label: `云账号 (${name})`, tagType: "info" } as const
	}
	return { label: row.type || "未知来源", tagType: "default" } as const
}

function formatDateTime(value: string) {
	if (!value) {
		return "--"
	}
	const date = new Date(value)
	if (Number.isNaN(date.getTime())) {
		return value
	}
	return date.toLocaleString("zh-CN", { hour12: false })
}

function isExpired(value: string) {
	if (!value) {
		return false
	}
	const date = new Date(value)
	return !Number.isNaN(date.getTime()) && date.getTime() < Date.now()
}

function downloadContent(content: string, fileName: string) {
	const blob = new Blob([content || ""], { type: "text/plain;charset=utf-8" })
	const link = document.createElement("a")
	link.href = URL.createObjectURL(blob)
	link.download = fileName
	link.click()
	URL.revokeObjectURL(link.href)
}

async function fetchWebsites() {
	const res = await ListWebsitesAPI()
	websites.value = Array.isArray(res.data) ? res.data : []
}

async function fetchCloudAccounts() {
	const res = await SearchCloudAccount({ page: 1, limit: 100 } as any)
	cloudAccounts.value = Array.isArray(res.data?.items) ? res.data.items : []
}

async function fetchData() {
	loading.value = true
	try {
		const res = await SSLSearchAPI({
			page: 1,
			limit: 200,
			wheres: []
		} as any)
		tableData.value = Array.isArray(res.data) ? res.data : []
	} finally {
		loading.value = false
	}
}

function resetUploadForm() {
	uploadForm.primaryDomain = ""
	uploadForm.otherDomains = ""
	uploadForm.description = ""
	uploadForm.pem = ""
	uploadForm.privateKey = ""
}

function resetCloudApplyForm() {
	cloudApplyForm.primaryDomain = ""
	cloudApplyForm.otherDomains = ""
	cloudApplyForm.cloudAccountId = null
	cloudApplyForm.cdnAccountId = null
	cloudApplyForm.description = ""
	cdnDomainsOptions.value = []
}

function openCloudApplyModal() {
	resetCloudApplyForm()
	cloudApplyModalVisible.value = true
}

function openSyncModal() {
	syncForm.websiteId = null
	syncModalVisible.value = true
}

function openUploadModal() {
	resetUploadForm()
	uploadModalVisible.value = true
}

function openApplyModal(sslId: number) {
	currentApplySSLId.value = sslId
	applyForm.websiteId = null
	applyModalVisible.value = true
}

function openPushCDNModal(row: SSLRow) {
	currentPushSSL.value = row
	// 默认选中当前证书所绑定的业务云账号（或DNS验证云账号）
	pushCDNForm.cloudAccountId = row.cloudAccountId || row.dnsAccountId || null
	pushCDNForm.targetDomain = row.primaryDomain
	pushCDNModalVisible.value = true
}

function openPushRuleModal(row: SSLRow) {
	currentPushSSL.value = row
	resetPushRuleForm()
	pushRuleModalVisible.value = true
	fetchPushRules()
}

function openLogModal(id: number) {
	currentLogSSLId.value = id
	logsData.value = []
	logModalVisible.value = true

	if (logEventSource) {
		logEventSource.close()
	}

	const authStore = useAuthStore()
	const apiUrl = (window as any).__VITE_API_URL__ || "/api" // 使用您的实际 baseUrl 方案
	logEventSource = new EventSource(`${apiUrl}/ssl/${id}/logs?token=${authStore.auth}`)

	logEventSource.onmessage = event => {
		if (event.data === "ping") return
		if (event.data === "EOF" || event.data === '["EOF"]') {
			logEventSource?.close()
			fetchData() // 自动刷新列表以更新证书有效期状态
			return
		}
		logsData.value.push(event.data)
	}

	logEventSource.onerror = () => {
		logsData.value.push("连接已断开或发生错误")
		logEventSource?.close()
	}
}

const handleLogModalChange = (value: boolean) => {
	logModalVisible.value = value
	if (!value && logEventSource) {
		logEventSource.close()
	}
}

async function openDetail(id: number) {
	const res = await GetSSL(id)
	currentSSL.value = res.data as unknown as SSLRow
	detailModalVisible.value = true
}

async function handleSyncCertificate() {
	if (!syncForm.websiteId) {
		message.warning("请选择网站")
		return
	}
	submitting.value = true
	try {
		await ObtainSSL({ ID: syncForm.websiteId })
		message.success("已同步 Caddy 自动证书")
		syncModalVisible.value = false
		await fetchData()
	} finally {
		submitting.value = false
	}
}

async function handleRenewCertificate(row: SSLRow) {
	dialog.info({
		title: "确认重新签发证书？",
		content: `将立刻为域名 ${row.primaryDomain} 重新发起签发流程。`,
		positiveText: "确认",
		negativeText: "取消",
		onPositiveClick: async () => {
			loading.value = true
			try {
				await RenewSSL({ id: row.id })
				message.success("已提交重签请求")
				// 提交后立刻打开日志流弹窗，实时观察签注过程
				openLogModal(row.id)
			} catch (error: any) {
				message.error(error.message || "提交失败")
			} finally {
				loading.value = false
			}
		}
	})
}

async function handleCloudApplyCertificate() {
	if (!cloudApplyForm.primaryDomain || !cloudApplyForm.cloudAccountId) {
		message.warning("请填写主域名并选择云账号")
		return
	}
	submitting.value = true
	try {
		const res = await CreateSSL({
			primaryDomain: cloudApplyForm.primaryDomain,
			otherDomains: cloudApplyForm.otherDomains,
			description: cloudApplyForm.description,
			dnsAccountId: cloudApplyForm.cloudAccountId,
			cloudAccountId: cloudApplyForm.cdnAccountId || 0,
			acmeAccountId: 0,
			type: "dns",
			provider: "acme-dns"
		} as any)
		message.success("已提交云账号签发请求")
		cloudApplyModalVisible.value = false
		// 提交后立刻打开日志流弹窗
		if (res.data && res.data.id) {
			openLogModal(res.data.id)
		} else {
			await fetchData()
		}
	} catch (error: any) {
		message.error(error.message || "提交签发请求失败")
	} finally {
		submitting.value = false
	}
}

async function handleUploadCertificate() {
	if (!uploadForm.primaryDomain || !uploadForm.pem || !uploadForm.privateKey) {
		message.warning("请填写主域名、证书内容和私钥")
		return
	}
	submitting.value = true
	try {
		await CreateSSL({
			primaryDomain: uploadForm.primaryDomain,
			otherDomains: uploadForm.otherDomains,
			description: uploadForm.description,
			provider: "custom",
			acmeAccountId: 0,
			cloudAccountId: 0,
			type: "upload",
			pem: uploadForm.pem,
			privateKey: uploadForm.privateKey
		} as any)
		message.success("证书已上传")
		uploadModalVisible.value = false
		await fetchData()
	} finally {
		submitting.value = false
	}
}

async function handleApplyCertificate() {
	if (!applyForm.websiteId || !currentApplySSLId.value) {
		message.warning("请选择网站")
		return
	}
	submitting.value = true
	try {
		await ApplySSL({
			websiteId: applyForm.websiteId,
			SSLId: currentApplySSLId.value
		})
		message.success("证书已绑定到网站")
		applyModalVisible.value = false
		await fetchData()
	} finally {
		submitting.value = false
	}
}

async function handlePushCDN() {
	if (!pushCDNForm.cloudAccountId) {
		message.warning("请选择云账号")
		return
	}
	submitting.value = true
	try {
		await PushToCDNAPI({
		  sslId: currentPushSSL.value?.id as number,
		  cloudAccountId: pushCDNForm.cloudAccountId,
		  targetDomain: pushCDNForm.targetDomain
		} as any)
		message.success(`已成功推送到指定的 CDN`)
		pushCDNModalVisible.value = false
	} catch (error: any) {
		message.error(error.message || "推送失败")
	} finally {
		submitting.value = false
	}
}

function confirmDelete(row: SSLRow) {
	dialog.warning({
		title: "确认删除吗？",
	    positiveText: "确认",
		negativeText: "取消",
		content:
			row.type === "caddy"
				? "删除后仅清理面板中的证书记录，不会删除 Caddy 已签发的证书文件。"
				: "删除后将移除当前上传证书记录。",
		onPositiveClick: async () => {
			await DeleteSSL({ id: row.id })
			message.success("删除成功")
			await fetchData()
		}
	})
}

onMounted(async () => {
	await Promise.all([fetchData(), fetchWebsites(), fetchCloudAccounts()])
})
</script>
