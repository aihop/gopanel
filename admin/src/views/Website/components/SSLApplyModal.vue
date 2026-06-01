<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { ApplyFormState } from "./sslHelpers"

defineProps<{
  show: boolean
  form: ApplyFormState
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
    title="绑定到网站"
    class="max-w-[560px]"
    @update:show="emit('update:show', $event)"
  >
    <div class="space-y-4">
      <div class="text-sm leading-7 fg-secondary-color">
        手动上传证书可以绑定到指定网站，系统会为该网站写入 `tls cert key` 配置。Caddy 自动 HTTPS 类型无需手动绑定。
      </div>
      <n-form label-placement="top">
        <n-form-item label="选择网站">
          <n-select
            v-model:value="form.websiteId"
            :options="websiteOptions"
            filterable
            placeholder="请选择要绑定的网站"
          />
        </n-form-item>
      </n-form>
      <div
        v-if="selectedRuntimeText"
        class="rounded-2xl px-4 py-4 text-sm fg-secondary-color" style="border: 1px solid color-mix(in srgb, var(--border-color) 80%, transparent); background-color: color-mix(in srgb, var(--bg-default-color) 95%, transparent)"
      >
        {{ selectedRuntimeText }}
      </div>
    </div>
    <template #action>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="emit('submit')">确认绑定</n-button>
      </n-space>
    </template>
  </n-modal>
</template>
