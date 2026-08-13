<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getAIProjects } from "@/api/modules/code"
import type { AIProject } from "@/api/interface/code"
import type { PipelineFormModel } from "./pipelineForm"
import { normalizePipelineKey } from "./pipelineForm"
import { pipelineSourceMessages } from "./pipelineSourceMessages"

defineProps<{
  formModel: PipelineFormModel
  authOptions: Array<{ label: string; value: string }>
}>()

const { t } = useI18n({ messages: pipelineSourceMessages })
const message = useMessage()
const codeProjects = ref<AIProject[]>([])
const codeProjectsLoading = ref(false)
const codeProjectsError = ref(false)
const codeProjectOptions = computed(() => codeProjects.value.map(project => ({ label: project.name, value: project.id })))

const emit = defineEmits<{
  (e: "mark-pipeline-key-touched"): void
}>()

const handlePipelineKeyInput = () => {
  emit("mark-pipeline-key-touched")
}

const loadCodeProjects = async () => {
  codeProjectsLoading.value = true
  codeProjectsError.value = false
  try {
    const response = await getAIProjects({ page: 1, limit: 100 })
    codeProjects.value = response.data.items || []
  } catch (error: any) {
    codeProjects.value = []
    codeProjectsError.value = true
    message.error(error?.message || t("pipelineSource.codeProjectsLoadFailed"))
  } finally {
    codeProjectsLoading.value = false
  }
}

onMounted(() => {
  void loadCodeProjects()
})
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
    label="唯一标识"
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
        用于生成流水线固定目录：`安装目录/pipelines/唯一标识/`。仅支持字母、数字和中划线；失焦时会自动规范化。创建或更新时会检查该目录是否已存在；若已被其他流水线占用，会提示你更换标识。
      </div>
    </div>
  </n-form-item>
  <n-form-item
    label="版本号"
    path="version"
  >
    <n-input
      v-model:value="formModel.version"
      placeholder="1.0.0"
    />
  </n-form-item>

  <n-form-item
    :label="t('pipelineSource.sourceType')"
    path="sourceType"
  >
    <n-radio-group v-model:value="formModel.sourceType">
      <n-space>
        <n-radio value="git">{{ t('pipelineSource.sourceGit') }}</n-radio>
        <n-radio value="code">{{ t('pipelineSource.sourceCode') }}</n-radio>
      </n-space>
    </n-radio-group>
  </n-form-item>
  <template v-if="formModel.sourceType === 'code'">
    <n-form-item :label="t('pipelineSource.codeProject')" path="codeProjectId">
      <div class="w-full">
        <n-select
          v-model:value="formModel.codeProjectId"
          :options="codeProjectOptions"
          :loading="codeProjectsLoading"
          :disabled="codeProjectsError"
          :placeholder="codeProjectsError ? t('pipelineSource.codeProjectsLoadFailed') : t('pipelineSource.codeProjectPlaceholder')"
          clearable
        />
        <n-alert v-if="codeProjectsError" class="mt-2" type="error" :show-icon="false">
          {{ t('pipelineSource.codeProjectsLoadFailed') }}
          <n-button class="ml-2" size="tiny" text type="primary" @click="loadCodeProjects">
            {{ t('pipelineSource.retry') }}
          </n-button>
        </n-alert>
        <div v-else-if="!codeProjectsLoading && codeProjectOptions.length === 0" class="mt-2 text-xs text-slate-500">
          {{ t('pipelineSource.codeProjectsEmpty') }}
        </div>
        <div v-else class="mt-2 text-xs text-slate-500">
          {{ t('pipelineSource.codeProjectHelper') }}
        </div>
      </div>
    </n-form-item>
  </template>
  <template v-else>
    <n-form-item :label="t('pipelineSource.repoUrl')" path="repoUrl">
      <n-input v-model:value="formModel.repoUrl" placeholder="https://github.com/..." />
    </n-form-item>
    <n-form-item :label="t('pipeline.branch')" path="branch">
      <n-input v-model:value="formModel.branch" placeholder="main" />
    </n-form-item>
    <n-form-item :label="t('pipelineSource.authType')" path="authType">
      <n-select v-model:value="formModel.authType" :options="authOptions" />
    </n-form-item>
    <n-form-item v-if="formModel.authType !== 'none'" :label="t('pipelineSource.authData')" path="authData">
      <n-input v-model:value="formModel.authData" :placeholder="t('pipelineSource.authDataPlaceholder')" type="password" show-password-on="click" />
    </n-form-item>
  </template>

 <n-form-item label="构建模式">
    <n-radio-group v-model:value="formModel.pipelineMode">
      <n-space vertical>
        <n-radio value="runner">简单模式 (代码产物部署)</n-radio>
        <div class="ml-6 text-xs text-slate-500">流水线会把代码产物解压到版本目录，再交给运行时基础镜像执行</div>
        <n-radio value="script">高级模式 (纯脚本)</n-radio>
        <div class="ml-6 text-xs text-slate-500">适合熟练用户，BuildScript 完全自管，交付结果可能是归档包或目录或运行中的服务</div>
      </n-space>
    </n-radio-group>
  </n-form-item>

</template>
