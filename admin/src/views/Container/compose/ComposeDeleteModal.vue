<template>
  <n-modal
    :show="show"
    preset="dialog"
    :title="'删除 - ' + row?.name"
    @update:show="emit('update:show', $event)"
  >
    <template #default>
      <n-checkbox
        :checked="deleteWithFile"
        @update:checked="emit('update:delete-with-file', $event)"
      >
        删除文件
      </n-checkbox>
      <div class="my-2 mb-4 text-[#888]">
        删除容器编排的所有文件，包括配置文件和持久化文件，请谨慎操作！
      </div>
      <div class="mb-2 text-[#d03050]">
        删除操作无法回滚，请输入 <b>"{{ row?.name }}"</b> 删除此编排
      </div>
      <n-input
        :value="deleteConfirmInput"
        placeholder="请输入名称"
        @update:value="emit('update:delete-confirm-input', $event)"
      />
      <div v-if="deleteError" class="mt-2 text-[#d03050]">{{ deleteError }}</div>
    </template>
    <template #action>
      <n-button @click="emit('update:show', false)">取消</n-button>
      <n-button
        type="error"
        :disabled="deleteConfirmInput !== row?.name"
        @click="emit('confirm')"
      >
        确认
      </n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import type { RowData } from "./composeTypes"

defineProps<{
  show: boolean
  row: RowData | null
  deleteWithFile: boolean
  deleteConfirmInput: string
  deleteError: string
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "update:delete-with-file", value: boolean): void
  (e: "update:delete-confirm-input", value: string): void
  (e: "confirm"): void
}>()
</script>
