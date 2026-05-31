<script setup lang="ts">
import { computed } from "vue"
import type { PipelineFormModel } from "./pipelineForm"
 
const props = defineProps<{
  formModel: PipelineFormModel
}>()

const actionParamsObj = computed({
  get() {
    try {
      return props.formModel.actionParams ? JSON.parse(props.formModel.actionParams) : {}
    } catch (e) {
      return {}
    }
  },
  set(val) {
    props.formModel.actionParams = JSON.stringify(val)
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
  <n-form-item label="产物操作" path="actionType">
    <n-radio-group v-model:value="formModel.actionType">
      <n-space vertical>
        <n-radio value="none">仅构建，不执行后续操作</n-radio>
        <div class="ml-6 text-xs text-slate-500">只完成打包归档，不部署到任何位置</div>
        <n-radio value="deploy">部署到关联网站</n-radio>
        <div class="ml-6 text-xs text-slate-500">构建产物自动推送到绑定的网站运行</div>
        <n-radio value="build_image">构建 Docker 镜像</n-radio>
        <div class="ml-6 text-xs text-slate-500">从 release 目录打包为 Docker 镜像</div>
      </n-space>
    </n-radio-group>
  </n-form-item>

  <n-form-item
    v-if="formModel.actionType === 'deploy'"
    label="产物路径"
    path="artifactPath"
  >
    <n-input
      v-model:value="formModel.artifactPath"
      placeholder="例如: dist/，如果不填则不进行部署和备份"
    />
  </n-form-item>

  <n-form-item
    v-if="formModel.actionType === 'build_image'"
    label="产出镜像名"
  >
    <div class="w-full">
      <n-input
        v-model:value="imageName"
        placeholder="选填，例如: aihop/web。系统会自动打上 tag"
      />
      <div class="mt-2 text-xs text-slate-500">
        用于指定系统自动构建 Docker 镜像时的目标名称。留空则默认使用流水线标识。
      </div>
    </div>
  </n-form-item>
</template>