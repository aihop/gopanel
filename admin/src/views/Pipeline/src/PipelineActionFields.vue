<script setup lang="ts">
import { computed } from "vue"
import type { PipelineFormModel } from "./pipelineForm"
 
const props = defineProps<{
  formModel: PipelineFormModel
}>()

const actionParamsObj = computed({
  get() {
    return props.formModel.actionParams || {}
  },
  set(val) {
    props.formModel.actionParams = val
  }
})

const imageName = computed({
  get() {
    return actionParamsObj.value.imageName || ''
  },
  set(val) {
    actionParamsObj.value = { ...actionParamsObj.value, imageName: val }
  }
})

</script>

<template>
  <n-form-item :label="$t('pipeline.artifactAction')" path="actionType">
    <n-radio-group v-model:value="formModel.actionType">
      <n-space vertical>
        <n-radio value="none">{{ $t("pipeline.actionNone") }}</n-radio>
        <div class="ml-6 text-xs text-slate-500">{{ $t("pipeline.actionNoneHelper") }}</div>
        <n-radio value="build_image">{{ $t("pipeline.actionBuildImage") }}</n-radio>
        <div class="ml-6 text-xs text-slate-500">{{ $t("pipeline.actionBuildImageHelper") }}</div>
      </n-space>
    </n-radio-group>
  </n-form-item>

  <n-form-item
    v-if="formModel.actionType === 'build_image'"
    :label="$t('pipeline.imageName')"
  >
    <div class="w-full">
      <n-input
        v-model:value="imageName"
        :placeholder="$t('pipeline.imageNamePlaceholder')"
      />
      <div class="mt-2 text-xs text-slate-500">
        {{ $t("pipeline.imageNameHelper") }}
      </div>
    </div>
  </n-form-item>
</template>
