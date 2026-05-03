<script setup lang="ts">
import type { DataTableColumns, SelectOption } from "naive-ui"
import type { Website } from "@/api/interface/website"
import type { PushRuleFormState } from "./sslHelpers"

defineProps<{
  show: boolean
  loading: boolean
  submitting: boolean
  form: PushRuleFormState
  data: Website.SSLPushRule[]
  columns: DataTableColumns<Website.SSLPushRule>
  cloudAccountOptions: SelectOption[]
  currentPrimaryDomain?: string
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "submit"): void
  (e: "reset"): void
}>()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="云端部署规则管理"
    style="width: 700px"
    @update:show="emit('update:show', $event)"
  >
    <div class="space-y-4">
      <div class="text-sm leading-7 text-slate-500">
        管理当前证书 ({{ currentPrimaryDomain }}) 的云端自动部署规则。当证书自动续签更新后，系统会自动推送到下方配置的云厂商资源中。
      </div>

      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="data"
        :bordered="false"
        :scroll-x="600"
      />

      <n-divider class="!my-2" />

      <n-form label-placement="top" class="space-y-2 rounded-xl border border-slate-100 bg-slate-50 p-4">
        <div class="mb-2 text-sm font-semibold">新增 / 编辑规则</div>
        <n-grid :cols="2" :x-gap="16">
          <n-form-item-gi label="选择云账号">
            <n-select
              v-model:value="form.cloudAccountId"
              :options="cloudAccountOptions"
              filterable
              placeholder="请选择授权的云账号"
            />
          </n-form-item-gi>
          <n-form-item-gi label="目标域名（选填）">
            <n-input v-model:value="form.targetDomain" placeholder="如不填，默认使用证书主域名" />
          </n-form-item-gi>
        </n-grid>
        <n-space justify="end">
          <n-button v-if="form.id" @click="emit('reset')">取消编辑</n-button>
          <n-button type="primary" :loading="submitting" @click="emit('submit')">保存规则</n-button>
        </n-space>
      </n-form>
    </div>
  </n-modal>
</template>
