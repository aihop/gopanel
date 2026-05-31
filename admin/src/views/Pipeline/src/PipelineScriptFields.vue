<script setup lang="ts">
import type { PipelineFormModel } from "./pipelineForm"
import FtEditor from "@/components/FtEditor/index.vue"

defineProps<{
  formModel: PipelineFormModel
}>()
</script>

<template>
 <n-form-item
    label="构建环境"
    path="buildEnv"
  >
    <n-radio-group v-model:value="formModel.buildEnv">
      <n-radio value="container">容器化构建</n-radio>
      <n-radio value="host">宿主机本地构建 (仅限专家)</n-radio>
    </n-radio-group>
  </n-form-item>
  <n-form-item
    v-if="formModel.buildEnv === 'container' || formModel.buildEnv === 'docker'"
    label="构建镜像"
    path="buildImage"
  >
    <n-input
      v-model:value="formModel.buildImage"
      placeholder="node:20-alpine"
    />
  </n-form-item>
  <n-form-item
    label="构建脚本"
    path="buildScript"
  >
    <FtEditor
      v-model="formModel.buildScript"
      language="bash"
      height="350px"
      placeholder="npm install && npm run build"
    />
  </n-form-item>
</template>
