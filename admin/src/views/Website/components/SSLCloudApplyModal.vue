<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { CloudApplyFormState } from "./sslHelpers"

defineProps<{
  show: boolean
  form: CloudApplyFormState
  cloudAccountOptions: SelectOption[]
  cdnDomainsOptions: Array<{ label: string; value: string }>
  cdnDomainsLoading: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "cdn-account-change", value: number | null): void
  (e: "submit"): void
}>()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="云账号签注"
    style="width: 650px"
    @update:show="emit('update:show', $event)"
  >
    <div class="space-y-6">
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
            :value="form.cdnAccountId"
            :options="cloudAccountOptions"
            filterable
            placeholder="请选择部署该域名的业务云账号"
            @update:value="emit('cdn-account-change', $event)"
          />
        </div>
      </div>

      <div
        class="rounded-xl border p-5 transition-all duration-300"
        :class="form.cdnAccountId ? 'border-blue-100 bg-blue-50/30' : 'pointer-events-none border-slate-100 bg-slate-50 opacity-50 grayscale'"
      >
        <div class="mb-3 flex items-center gap-3">
          <div
            class="flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold text-white"
            :class="form.cdnAccountId ? 'bg-blue-600' : 'bg-slate-300'"
          >
            2
          </div>
          <div class="text-base font-bold text-slate-800">选择或填写证书域名</div>
        </div>
        <div class="mb-4 ml-9 text-sm text-slate-500">
          您可以直接在下拉列表中选择从云服务商拉取的 CDN 域名，或手动输入（如 *.example.com）。
        </div>
        <div class="ml-9">
          <n-grid :cols="2" :x-gap="16">
            <n-form-item-gi :label="$t('website.primaryDomain')">
              <n-auto-complete
                v-model:value="form.primaryDomain"
                :options="cdnDomainsOptions"
                :loading="cdnDomainsLoading"
                placeholder="选择CDN域名或输入 *.example.com"
                :get-show="() => true"
                clearable
              />
            </n-form-item-gi>
            <n-form-item-gi label="其他域名 (可选)">
              <n-input v-model:value="form.otherDomains" placeholder="www.example.com" />
            </n-form-item-gi>
          </n-grid>
          <n-form-item label="备注">
            <n-input v-model:value="form.description" placeholder="例如：主站通配符证书" />
          </n-form-item>
        </div>
      </div>

      <div
        class="rounded-xl border p-5 transition-all duration-300"
        :class="form.primaryDomain ? 'border-blue-100 bg-blue-50/30' : 'pointer-events-none border-slate-100 bg-slate-50 opacity-50 grayscale'"
      >
        <div class="mb-3 flex items-center gap-3">
          <div
            class="flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold text-white"
            :class="form.primaryDomain ? 'bg-blue-600' : 'bg-slate-300'"
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
            v-model:value="form.cloudAccountId"
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
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button
          type="primary"
          :loading="submitting"
          :disabled="!form.cdnAccountId || !form.primaryDomain"
          @click="emit('submit')"
        >
          提交签发
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>
