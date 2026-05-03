<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { SyncFormState } from "./sslHelpers"

defineProps<{
  show: boolean
  form: SyncFormState
  websiteOptions: SelectOption[]
  selectedRuntimeText: string
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
    style="width: 420px"
    title="同步网站证书"
    @update:show="emit('update:show', $event)"
  >
    <div class="space-y-4">
      <div class="text-sm leading-7 text-slate-500">
        选择一个已经启用的网站，系统会到默认证书存储目录中读取该域名的已签发证书，并同步到面板中以便查看和下载。
      </div>
      <n-form label-placement="top">
        <n-form-item label="选择网站">
          <n-select
            v-model:value="form.websiteId"
            :options="websiteOptions"
            filterable
            placeholder="请选择要同步证书的网站"
          />
        </n-form-item>
      </n-form>
      <div
        v-if="selectedRuntimeText"
        class="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600"
      >
        {{ selectedRuntimeText }}
      </div>
    </div>
    <template #action>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="emit('submit')">开始同步</n-button>
      </n-space>
    </template>
  </n-modal>
</template>
