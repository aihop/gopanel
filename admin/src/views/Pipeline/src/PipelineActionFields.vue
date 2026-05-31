<script setup lang="ts">
import type { PipelineFormModel } from "./pipelineForm"
 
defineProps<{
  formModel: PipelineFormModel
}>()
 
</script>

<template>
  <n-form-item label="产物操作">
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
</template>