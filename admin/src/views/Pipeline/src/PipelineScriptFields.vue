<script setup lang="ts">
import type { PipelineFormModel } from "./pipelineForm"
import FtEditor from "@/components/FtEditor/index.vue"

defineProps<{
  formModel: PipelineFormModel
}>()
</script>

<template>
  <div class="mb-4 mt-6 text-sm font-semibold text-slate-700">高级模式 (纯脚本)</div>
  <n-form-item
    label="构建环境"
    path="buildEnv"
  >
    <n-radio-group v-model:value="formModel.buildEnv">
      <n-radio value="container">容器化构建 (推荐，基于 Docker/Podman)</n-radio>
      <n-radio value="host">宿主机本地构建 (环境依赖复杂，仅限专家)</n-radio>
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
  <n-form-item
    label="产出镜像名"
    path="outputImage"
  >
    <n-input
      v-model:value="formModel.outputImage"
      placeholder="例如: shoply。系统会自动拼成 shoply:<版本号>"
    />
  </n-form-item>
  <n-form-item
    label="产物路径"
    path="artifactPath"
  >
    <n-input
      v-model:value="formModel.artifactPath"
      placeholder="例如: dist/，如果不填则不进行部署和备份"
    />
  </n-form-item>
  <n-form-item
    label="服务端口"
    path="exposePort"
  >
    <div class="w-full">
      <n-input-number
        v-model:value="formModel.exposePort"
        placeholder="纯脚本自管运行时填写，例如: 3001"
        clearable
      />
      <div class="mt-2 text-xs text-slate-500">
        仅纯脚本模式且由脚本自行 `podman run/start` 服务时填写宿主机访问端口。留空表示只构建产物、不由网站直接接管运行；该值不会修改容器内部监听端口。
      </div>
    </div>
  </n-form-item>
</template>
