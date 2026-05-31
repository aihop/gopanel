<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { NButton, useMessage } from "naive-ui"
import { createPipeline, updatePipeline, detectPipelineRunnerPreset } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import { containerValidateAPI } from "@/api/modules/container"
import FullModal from "@/components/FullModal.vue"
import { buildRuntimeDetailText, getRuntimeKindLabel, getRuntimeModeLabel } from "@/utils/runtime"
import PipelineBasicFields from "./PipelineBasicFields.vue"
import PipelineRunnerFields from "./PipelineRunnerFields.vue"
import PipelineScriptFields from "./PipelineScriptFields.vue"
import PipelineActionFields from "./PipelineActionFields.vue"
import {
  applyRunnerPresetToForm,
  buildRunnerDirectorySummary,
  authOptions,
  createDefaultPipelineFormModel,
  createPipelineFormFromEdit,
  createPipelineFormFromTemplate,
  defaultRunnerWorkingDir,
  getRunnerBuildCommandHint,
  getRunnerBuildCommandPlaceholder,
  hasForbiddenRunnerPersistentPath,
  normalizePipelineKey,
  parseEnvText,
  parseExtraNetworksText,
  parsePersistentPathsText,
  runnerPresetOptions
} from "./pipelineForm"

const props = defineProps<{
  show: boolean
  editData?: Pipeline.ResPipeline | null
  initialTemplate?: any | null
}>()

const emit = defineEmits(["update:show", "success"])

const message = useMessage()
const formRef = ref()
const loading = ref(false)
const pipelineKeyTouched = ref(false)
const runtimeValidate = ref<any>(null)
const runtimeValidateLoading = ref(false)
const runnerPreset = ref("nuxt")
const runnerPresetAutoLabel = ref("")
const runnerPresetHits = ref<string[]>([])
const runnerPresetManualTouched = ref(false)
let runnerDetectTimer: number | null = null

const formModel = reactive(createDefaultPipelineFormModel())
const isEdit = computed(() => !!props.editData)

const currentRuntimeKindLabel = computed(() => {
  return getRuntimeKindLabel(runtimeValidate.value, {
    kindFallback: "容器运行时"
  })
})

const currentRuntimeModeLabel = computed(() => {
  const runtimeInfo = runtimeValidate.value?.runtime || {}
  const mode = !!runtimeValidate.value?.rootlessHost || !!runtimeInfo?.rootless
    ? "rootless"
    : runtimeValidate.value?.runtimeKind
      ? "rootful"
      : ""
  return getRuntimeModeLabel({ runtimeMode: mode }, {
    defaultModeLabel: "default"
  })
})

const runnerRuntimeHint = computed(() => {
  if (!runtimeValidate.value?.runtimeKind) {
    return "Runner 会跟随当前面板已选中的容器运行时。`runnerUser` 只影响容器内进程用户，不会改变宿主机是 rootless 还是 rootful。"
  }
  const host = String(runtimeValidate.value?.runtimeHost || "").trim()
  const hostText = host ? `；当前 Host：${host}` : ""
  return `Runner 当前会跟随 ${currentRuntimeKindLabel.value} / ${currentRuntimeModeLabel.value}${hostText}。非 root 安装通常应落到 rootless 运行时；这里的“容器内运行用户”仅控制进程身份，不改变宿主机运行时模式。`
})

const existingRuntimeHint = computed(() => {
  if (!props.editData?.runnerMode) return ""
  if (!(props.editData.runtimeKind || props.editData.runtimeMode || props.editData.runUser)) return ""
  return buildRuntimeDetailText(props.editData, {
    prefix: "该流水线当前记录",
    kindFallback: "Runtime",
    userFallback: "镜像默认",
    runtimePrefix: "运行时：",
    runUserPrefix: "用户："
  })
})

const existingRunnerDirectoryHint = computed(() => {
  if (!props.editData?.runnerMode) return ""
  return buildRunnerDirectorySummary(props.editData.runnerConfig)
})

const runnerBuildCommandPlaceholder = computed(() => getRunnerBuildCommandPlaceholder(runnerPreset.value))
const runnerBuildCommandHint = computed(() => getRunnerBuildCommandHint(runnerPreset.value))

const validateOptionalPort = (_rule: any, value: string) => {
  const text = String(value || "").trim()
  if (!text) return true
  if (!/^\d+$/.test(text)) return new Error("端口必须是数字")
  const port = Number(text)
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return new Error("端口范围必须在 1-65535")
  }
  return true
}

const rules = {
  name: { required: true, message: "请输入名称", trigger: "blur" },
  branch: { required: true, message: "请输入分支名称", trigger: "blur" },
  version: { required: true, message: "请输入版本号", trigger: "blur" },
  pipelineKey: {
    validator: (_rule: any, value: string) => {
      const raw = String(value || "").trim()
      if (!raw) return new Error("请输入流水线唯一标识")
      const normalized = normalizePipelineKey(raw)
      if (!normalized) {
        return new Error("流水线唯一标识仅支持字母、数字、中划线，且规范化后不能为空")
      }
      return true
    },
    trigger: ["blur", "input"]
  },
  runnerContainerPort: {
    validator: validateOptionalPort,
    trigger: ["blur", "input"]
  },
  runnerHostPort: {
    validator: validateOptionalPort,
    trigger: ["blur", "input"]
  }
}

const resetAutoDetectState = () => {
  runnerPresetAutoLabel.value = ""
  runnerPresetHits.value = []
}

const loadRuntimeValidate = async () => {
  if (runtimeValidateLoading.value) return
  runtimeValidateLoading.value = true
  try {
    const res: any = await containerValidateAPI()
    if (res?.code === 0) {
      runtimeValidate.value = res.data || null
    }
  } catch (error) {
  } finally {
    runtimeValidateLoading.value = false
  }
}

const resetToDefaultForm = () => {
  Object.assign(formModel, createDefaultPipelineFormModel())
  pipelineKeyTouched.value = false
  runnerPreset.value = "nuxt"
  runnerPresetManualTouched.value = false
  resetAutoDetectState()
}

const applyEditForm = (editData: Pipeline.ResPipeline) => {
  const state = createPipelineFormFromEdit(editData)
  Object.assign(formModel, state.form)
  pipelineKeyTouched.value = state.pipelineKeyTouched
  runnerPreset.value = state.runnerPreset
  runnerPresetManualTouched.value = true
  resetAutoDetectState()
}

const applyTemplateForm = (initialTemplate: any) => {
  Object.assign(formModel, createPipelineFormFromTemplate(initialTemplate))
  pipelineKeyTouched.value = false
  runnerPreset.value = "custom"
  runnerPresetManualTouched.value = false
  resetAutoDetectState()
}

const applyRunnerPreset = (preset: string) => {
  runnerPresetManualTouched.value = true
  runnerPreset.value = preset
  applyRunnerPresetToForm(formModel, preset)
}

const markRunnerPresetCustom = () => {
  runnerPresetManualTouched.value = true
  runnerPreset.value = "custom"
}

const applyRunnerPresetSilently = (preset: string) => {
  runnerPreset.value = preset
  applyRunnerPresetToForm(formModel, preset)
}

const scheduleRunnerPresetDetect = () => {
  if (runnerDetectTimer) {
    window.clearTimeout(runnerDetectTimer)
  }
  runnerDetectTimer = window.setTimeout(async () => {
    if (formModel.pipelineMode !== "runner") return
    if (runnerPresetManualTouched.value) return
    const repoUrl = String(formModel.repoUrl || "").trim()
    const branch = String(formModel.branch || "").trim()
    if (!repoUrl || !branch) return
    try {
      const res: any = await detectPipelineRunnerPreset({
        repoUrl,
        branch,
        authType: formModel.authType,
        authData: formModel.authData
      })
      const data = res?.data || res
      const preset = String(data?.preset || "").trim()
      if (!preset || preset === "custom") {
        runnerPresetAutoLabel.value = ""
        runnerPresetHits.value = Array.isArray(data?.hits) ? data.hits : []
        return
      }
      applyRunnerPresetSilently(preset)
      runnerPresetAutoLabel.value = preset
      runnerPresetHits.value = Array.isArray(data?.hits) ? data.hits : []
    } catch (error) {
      resetAutoDetectState()
    }
  }, 500)
}

const handleClose = () => {
  emit("update:show", false)
}

const handleSubmit = () => {
  formRef.value?.validate(async (errors: any) => {
    if (errors) return
    loading.value = true
    try {
      const payload: any = { ...formModel }
      payload.pipelineKey = normalizePipelineKey(payload.pipelineKey || "")

      if (payload.pipelineMode === "runner") {
        payload.buildImage = ""
        payload.buildScript = ""
        payload.outputImage = ""
        payload.artifactPath = payload.artifactPath || "."
        payload.runnerEnabled = true
      } else {
        if (payload.buildEnv === "host") {
          payload.buildImage = "host"
        }
        payload.runnerEnabled = false
      }

      if (payload.runnerEnabled) {
        const runnerPersistentPaths = parsePersistentPathsText(payload.runnerPersistentPathsText || "")
        if (hasForbiddenRunnerPersistentPath(runnerPersistentPaths)) {
          message.error("持久化目录不要填写 `node_modules`，依赖目录会由 Runner 自动隔离并在容器内重装。")
          return
        }
        const runnerWorkingDir = String(payload.runnerWorkingDir || "").trim()
        payload.runnerContainerPort = String(payload.runnerContainerPort || "").trim()
        payload.runnerHostPort = String(payload.runnerHostPort || "").trim()
        payload.runnerMode = "runner"
        payload.runnerConfig = {
          advanced: !!payload.runnerAdvanced,
          mode: payload.runnerPolicy || "build_run",
          baseImage: payload.runnerBaseImage || "node:20-alpine",
          containerPort: payload.runnerContainerPort || "3000",
          hostPort: payload.runnerHostPort || "",
          runnerUser: payload.runnerUser || "",
          buildCommand: payload.runnerBuildCommand || "",
          startCommand: payload.runnerStartCommand || "node .output/server/index.mjs",
          preStart: payload.runnerPreStart || "",
          env: parseEnvText(payload.runnerEnvText || ""),
          persistentPaths: runnerPersistentPaths,
          extraNetworks: parseExtraNetworksText(payload.runnerExtraNetworksText || "")
        }
        if (runnerWorkingDir && runnerWorkingDir !== defaultRunnerWorkingDir) {
          payload.runnerConfig.workingDir = runnerWorkingDir
        }
      } else {
        payload.runnerMode = ""
        payload.runnerConfig = undefined
      }

      payload.exposePort = Number(payload.exposePort) || 0

      if (isEdit.value && props.editData) {
        await updatePipeline({ id: props.editData.id, ...payload })
        message.success("更新成功")
      } else {
        await createPipeline(payload)
        message.success("创建成功")
      }
      handleClose()
      emit("success")
    } catch (error: any) {
      message.error(error.message || (isEdit.value ? "更新失败" : "创建失败"))
    } finally {
      loading.value = false
    }
  })
}

watch(() => props.show, (val) => {
  if (!val) {
    if (runnerDetectTimer) {
      window.clearTimeout(runnerDetectTimer)
      runnerDetectTimer = null
    }
    return
  }

  loadRuntimeValidate()
  if (props.editData) {
    applyEditForm(props.editData)
    return
  }
  if (props.initialTemplate) {
    applyTemplateForm(props.initialTemplate)
    return
  }
  resetToDefaultForm()
})

watch(() => formModel.name, (val) => {
  if (pipelineKeyTouched.value) return
  formModel.pipelineKey = normalizePipelineKey(val)
})

watch(() => formModel.pipelineMode, (mode) => {
  if (!pipelineKeyTouched.value && formModel.name) {
    formModel.pipelineKey = normalizePipelineKey(formModel.name)
  }
  if (mode !== "runner") {
    resetAutoDetectState()
    return
  }
  scheduleRunnerPresetDetect()
})

watch(() => [formModel.repoUrl, formModel.branch, formModel.authType, formModel.authData], () => {
  scheduleRunnerPresetDetect()
})
</script>

<template>
  <FullModal
    :show="show"
    :title="isEdit ? '编辑流水线' : '新增流水线'"
    width="900px"
    @update:show="handleClose"
  >
    <n-form
      ref="formRef"
      :model="formModel"
      :rules="rules"
      label-placement="left"
      label-width="100"
    >
      <PipelineBasicFields
        :form-model="formModel"
        :auth-options="authOptions"
        @mark-pipeline-key-touched="pipelineKeyTouched = true"
      />
      <PipelineRunnerFields
        v-if="formModel.pipelineMode === 'runner'"
        :form-model="formModel"
        :is-edit="isEdit"
        :existing-runtime-hint="existingRuntimeHint"
        :existing-runner-directory-hint="existingRunnerDirectoryHint"
        :runner-runtime-hint="runnerRuntimeHint"
        :runner-preset="runnerPreset"
        :runner-preset-options="runnerPresetOptions"
        :runner-preset-auto-label="runnerPresetAutoLabel"
        :runner-preset-hits="runnerPresetHits"
        :runner-build-command-placeholder="runnerBuildCommandPlaceholder"
        :runner-build-command-hint="runnerBuildCommandHint"
        @apply-runner-preset="applyRunnerPreset"
        @mark-runner-preset-custom="markRunnerPresetCustom"
      />
      <PipelineScriptFields
        v-else
        :form-model="formModel"
      />
      <PipelineActionFields
        :form-model="formModel"
      />
    </n-form>
    <template #footer>
      <div class="mt-8 flex justify-end gap-3">
        <n-button @click="handleClose">取消</n-button>
        <n-button
          type="primary"
          :loading="loading"
          @click="handleSubmit"
        >{{ isEdit ? '确认更新' : '确认创建' }}</n-button>
      </div>
    </template>
  </FullModal>
</template>
