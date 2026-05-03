<script setup lang="ts">
import type { PipelineFormModel } from "./pipelineForm"
import { normalizePipelineKey } from "./pipelineForm"

defineProps<{
  formModel: PipelineFormModel
  authOptions: Array<{ label: string; value: string }>
}>()

const emit = defineEmits<{
  (e: "mark-pipeline-key-touched"): void
}>()

const handlePipelineKeyInput = () => {
  emit("mark-pipeline-key-touched")
}
</script>

<template>
  <n-form-item
    label="名称"
    path="name"
  >
    <n-input
      v-model:value="formModel.name"
      placeholder="流水线名称"
    />
  </n-form-item>
  <n-form-item
    label="描述"
    path="description"
  >
    <n-input
      v-model:value="formModel.description"
      placeholder="用途说明..."
    />
  </n-form-item>
  <n-form-item
    label="流水线标识"
    path="pipelineKey"
  >
    <div class="w-full">
      <n-input
        v-model:value="formModel.pipelineKey"
        placeholder="例如：aipanel-site"
        @update:value="handlePipelineKeyInput"
        @blur="formModel.pipelineKey = normalizePipelineKey(formModel.pipelineKey)"
      />
      <div class="mt-2 text-xs text-slate-500">
        用于生成流水线固定目录：`安装目录/pipelines/流水线标识/`。仅支持字母、数字和中划线；失焦时会自动规范化。创建或更新时会检查该目录是否已存在；若已被其他流水线占用，会提示你更换标识。
      </div>
    </div>
  </n-form-item>
  <n-form-item
    label="初始版本号"
    path="version"
  >
    <n-input
      v-model:value="formModel.version"
      placeholder="1.0.0"
    />
  </n-form-item>

  <div class="mb-4 mt-6 text-sm font-semibold text-slate-700">交付模式</div>
  <n-form-item label="模式选择">
    <n-radio-group v-model:value="formModel.pipelineMode">
      <n-space vertical>
        <n-radio value="runner">简单模式 (代码产物部署，推荐)</n-radio>
        <div class="ml-6 text-xs text-slate-500">流水线会把代码产物解压到版本目录，再交给运行时基础镜像执行；</div>
        <n-radio value="script">高级模式 (纯脚本)</n-radio>
        <div class="ml-6 text-xs text-slate-500">适合熟练用户。你完全自行控制 BuildScript、镜像构建、产物归档与发布流程。</div>
      </n-space>
    </n-radio-group>
  </n-form-item>

  <div class="mb-4 mt-6 text-sm font-semibold text-slate-700">源码配置 (选填，纯脚本模式可留空)</div>
  <n-form-item
    label="仓库地址"
    path="repoUrl"
  >
    <n-input
      v-model:value="formModel.repoUrl"
      placeholder="https://github.com/..."
    />
  </n-form-item>
  <n-form-item
    label="分支"
    path="branch"
  >
    <n-input
      v-model:value="formModel.branch"
      placeholder="main"
    />
  </n-form-item>
  <n-form-item
    label="认证方式"
    path="authType"
  >
    <n-select
      v-model:value="formModel.authType"
      :options="authOptions"
    />
  </n-form-item>
  <n-form-item
    v-if="formModel.authType !== 'none'"
    label="凭证信息"
    path="authData"
  >
    <n-input
      v-model:value="formModel.authData"
      placeholder="填写 Token 或 Password"
      type="password"
      show-password-on="click"
    />
  </n-form-item>
</template>
