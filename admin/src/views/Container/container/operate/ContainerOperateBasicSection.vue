<template>
  <n-form-item
    class="mt-5"
    :label="$t('commons.table.name')"
    path="name"
  >
    <div class="w-full">
      <n-input
        v-model:value="rowData.name"
        :disabled="isFromAppValue"
        clearable
      />
      <div v-if="title === 'edit' && isFromAppValue">
        <span class="input-help">
          {{ $t("container.containerFromAppHelper1") }}
          <n-button
            class="-ml-1"
            size="small"
            text
            type="primary"
            @click="emit('go-router')"
          >
            {{ $t("firewall.quickJump") }}
          </n-button>
        </span>
      </div>
    </div>
  </n-form-item>

  <n-form-item
    :label="$t('container.image')"
    path="image"
  >
    <div class="flex w-full flex-col">
      <n-checkbox
        v-model:checked="rowData.imageInput"
        class="mb-2"
        :label="$t('container.input')"
      />
      <n-select
        v-if="!rowData.imageInput"
        v-model:value="rowData.image"
        filterable
        :options="images"
        label-field="option"
        value-field="option"
      />
      <n-input
        v-else
        v-model:value="rowData.image"
      />
    </div>
  </n-form-item>

  <n-form-item
    path="forcePull"
    :show-label="false"
  >
    <div class="flex w-full flex-col">
      <n-checkbox
        v-model:checked="rowData.forcePull"
        :label="$t('container.forcePull')"
        class="mb-2"
      />
      <span class="input-help">
        {{ $t("container.forcePullHelper") }}
      </span>
    </div>
  </n-form-item>
</template>

<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { Container } from "@/api/interface/container"

defineProps<{
  title: string
  rowData: Container.ContainerHelper
  isFromAppValue: boolean
  images: SelectOption[]
}>()

const emit = defineEmits<{
  (e: "go-router"): void
}>()
</script>

<style scoped lang="scss">
.input-help {
  color: #adb0bc;
}
</style>
