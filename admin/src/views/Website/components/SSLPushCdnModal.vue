<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { PushCDNFormState, SSLRow } from "./sslHelpers"

defineProps<{
  show: boolean
  form: PushCDNFormState
  cloudAccountOptions: SelectOption[]
  currentSsl: SSLRow | null
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "submit"): void
}>()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="推送到云厂商 CDN"
    style="width: 520px"
    @update:show="emit('update:show', $event)"
  >
    <div class="space-y-4">
      <div class="text-sm leading-7 text-slate-500">
        将当前证书 ({{ currentSsl?.primaryDomain }}) 推送到云厂商，以便 CDN、WAF 或对象存储等云产品使用最新的证书内容。
      </div>
      <n-form label-placement="top" class="space-y-2">
        <n-form-item label="选择云账号">
          <n-select
            v-model:value="form.cloudAccountId"
            :options="cloudAccountOptions"
            filterable
            placeholder="请选择授权的云账号"
          />
        </n-form-item>
        <n-form-item label="目标域名（选填）">
          <n-input v-model:value="form.targetDomain" placeholder="如不填，默认使用证书的主域名" />
        </n-form-item>
      </n-form>
    </div>
    <template #action>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="emit('submit')">开始推送</n-button>
      </n-space>
    </template>
  </n-modal>
</template>
