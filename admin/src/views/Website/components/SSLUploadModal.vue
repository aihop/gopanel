<script setup lang="ts">
import FullModal from "@/components/FullModal.vue"
import type { UploadFormState } from "./sslHelpers"

defineProps<{
  show: boolean
  form: UploadFormState
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "submit"): void
}>()
</script>

<template>
  <FullModal
    :show="show"
    title="上传证书"
    class="max-w-[760px]"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top" class="space-y-2">
      <n-grid :cols="2" :x-gap="16">
        <n-form-item-gi :label="$t('website.primaryDomain')">
          <n-input v-model:value="form.primaryDomain" placeholder="example.com" />
        </n-form-item-gi>
        <n-form-item-gi label="其他域名">
          <n-input v-model:value="form.otherDomains" placeholder="www.example.com, api.example.com" />
        </n-form-item-gi>
      </n-grid>
      <n-form-item label="备注">
        <n-input v-model:value="form.description" placeholder="例如：商业证书 / 迁移证书" />
      </n-form-item>
      <n-form-item label="证书内容 PEM">
        <n-input v-model:value="form.pem" type="textarea" :autosize="{ minRows: 8, maxRows: 12 }" />
      </n-form-item>
      <n-form-item label="私钥内容 KEY">
        <n-input v-model:value="form.privateKey" type="textarea" :autosize="{ minRows: 8, maxRows: 12 }" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="emit('submit')">保存</n-button>
      </n-space>
    </template>
  </FullModal>
</template>
