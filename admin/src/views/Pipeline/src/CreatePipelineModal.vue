<script setup lang="ts">
import { ref, reactive, watch, computed } from "vue"
import { NForm, NFormItem, NInput, NSelect, NButton, NRadioGroup, NRadio, useMessage, NInputNumber, NSwitch, NSpace } from "naive-ui"
import { createPipeline, updatePipeline } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
import { containerPrecheck } from "@/api/modules/container"
import FullModal from "@/components/FullModal.vue"
import FtEditor from "@/components/FtEditor/index.vue"

const props = defineProps<{ 
  show: boolean,
  editData?: Pipeline.ResPipeline | null,
  initialTemplate?: any | null
}>()
const emit = defineEmits(["update:show", "success"])

const message = useMessage()
const formRef = ref()
const loading = ref(false)
const runnerKeyTouched = ref(false)
const runtimePrecheck = ref<any>(null)
const runtimePrecheckLoading = ref(false)

const isEdit = computed(() => !!props.editData)

const currentRuntimeKindLabel = computed(() => {
  const kind = String(runtimePrecheck.value?.runtimeKind || "").toLowerCase()
  if (kind === "podman") return "Podman"
  if (kind === "docker") return "Docker"
  return "容器运行时"
})

const currentRuntimeModeLabel = computed(() => {
  const runtimeInfo = runtimePrecheck.value?.runtime || {}
  const isRootless = !!runtimePrecheck.value?.rootlessHost || !!runtimeInfo?.rootless
  if (isRootless) return "rootless"
  if (runtimePrecheck.value?.runtimeKind) return "rootful"
  return "default"
})

const runnerRuntimeHint = computed(() => {
  if (!runtimePrecheck.value?.runtimeKind) {
    return "简单模式 Runner 会跟随当前面板已选中的容器运行时。`runnerUser` 只影响容器内进程用户，不会改变宿主机是 rootless 还是 rootful。"
  }
  const host = String(runtimePrecheck.value?.runtimeHost || "").trim()
  const hostText = host ? `；当前 Host：${host}` : ""
  return `简单模式 Runner 当前会跟随 ${currentRuntimeKindLabel.value} / ${currentRuntimeModeLabel.value}${hostText}。非 root 安装通常应落到 rootless 运行时；这里的“容器内运行用户”仅控制进程身份，不改变宿主机运行时模式。`
})

const loadRuntimePrecheck = async () => {
  if (runtimePrecheckLoading.value) return
  runtimePrecheckLoading.value = true
  try {
    const res: any = await containerPrecheck()
    if (res?.code === 0) {
      runtimePrecheck.value = res.data || null
    }
  } catch (e) {
  } finally {
    runtimePrecheckLoading.value = false
  }
}

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

const formModel = reactive({
  name: "",
  description: "",
  repoUrl: "",
  branch: "main",
  version: "1.0.0", // 默认初始版本号
  authType: "none",
  authData: "",
  pipelineMode: "runner", // runner | script
  buildEnv: "container", // 新增前端状态：host | container
  buildImage: "node:20-alpine",
  buildScript: "npm install && npm run build",
  outputImage: "",
  artifactPath: ".",
  exposePort: 80, // 默认访问端口建议值（单个）
  runnerKey: "",

  runnerEnabled: false,
  runnerPolicy: "build_run",
  runnerAdvanced: false,
  runnerBaseImage: "node:20-alpine",
  runnerWorkingDir: "/var/www/app",
  runnerContainerPort: "3000",
  runnerHostPort: "",
  runnerUser: "",
  runnerInstallCommand: "",
  runnerStartCommand: "node .output/server/index.mjs",
  runnerPreStart: "",
  runnerEnvText: "",
  runnerPersistentPathsText: "",
  runnerExtraNetworksText: ""
})

const rules = {
  name: { required: true, message: "请输入名称", trigger: "blur" },
  branch: { required: true, message: "请输入分支名称", trigger: "blur" },
  version: { required: true, message: "请输入初始版本号", trigger: "blur" },
  runnerKey: {
    validator: (_rule: any, value: string) => {
      if (formModel.pipelineMode !== "runner") return true
      if (!String(value || "").trim()) return new Error("请输入流水线标识")
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

const authOptions = [
  { label: "公开仓库 (无需凭证)", value: "none" },
  { label: "Token 凭证 (推荐)", value: "token" },
  { label: "账号密码", value: "password" }
]

const runnerPresetOptions = [
  { label: "Nuxt (推荐)", value: "nuxt" },
  { label: "Next.js", value: "next" },
  { label: "Node 通用", value: "node" },
  { label: "自定义", value: "custom" }
]

const runnerPreset = ref("nuxt")
const installCommandOptions = [
  { label: "自动检测 lockfile (推荐)", value: "" },
  { label: "npm ci", value: "npm ci" },
  { label: "pnpm install --frozen-lockfile", value: "pnpm install --frozen-lockfile" },
  { label: "yarn install --frozen-lockfile", value: "yarn install --frozen-lockfile" }
]

const parseEnvText = (text: string) => {
  const env: Record<string, string> = {}
  const lines = (text || "").split("\n")
  for (const lineRaw of lines) {
    const line = String(lineRaw || "").trim()
    if (!line) continue
    const idx = line.indexOf("=")
    if (idx <= 0) continue
    const k = line.slice(0, idx).trim()
    const v = line.slice(idx + 1)
    if (!k) continue
    env[k] = v
  }
  return env
}

const parsePersistentPathsText = (text: string) => {
  const items: string[] = []
  const seen = new Set<string>()
  const lines = (text || "").split("\n")
  for (const lineRaw of lines) {
    const line = String(lineRaw || "").trim()
    if (!line) continue
    if (seen.has(line)) continue
    seen.add(line)
    items.push(line)
  }
  return items
}

const parseExtraNetworksText = (text: string) => {
  const items: string[] = []
  const seen = new Set<string>()
  const lines = (text || "").split("\n")
  for (const lineRaw of lines) {
    const line = String(lineRaw || "").trim()
    if (!line) continue
    if (seen.has(line)) continue
    seen.add(line)
    items.push(line)
  }
  return items
}

const normalizeRunnerKey = (text: string) => {
  const raw = String(text || "").trim().toLowerCase()
  if (!raw) return ""
  return raw
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^[-_]+|[-_]+$/g, "")
}

const inferRunnerAdvanced = (runnerConfig: any) => {
  if (!runnerConfig || typeof runnerConfig !== "object") return false
  if (typeof runnerConfig.advanced === "boolean") return runnerConfig.advanced

  const baseImage = String(runnerConfig.baseImage || "").trim()
  const workingDir = String(runnerConfig.workingDir || "").trim()
  const containerPort = String(runnerConfig.containerPort || "").trim()
  const hostPort = String(runnerConfig.hostPort || "").trim()
  const runnerUser = String(runnerConfig.runnerUser || "").trim()
  const startCommand = String(runnerConfig.startCommand || "").trim()
  const preStart = String(runnerConfig.preStart || "").trim()
  const buildCommand = String(runnerConfig.buildCommand || "").trim()
  const env = runnerConfig.env && typeof runnerConfig.env === "object" ? runnerConfig.env : {}
  const persistentPaths = Array.isArray(runnerConfig.persistentPaths) ? runnerConfig.persistentPaths : []
  const extraNetworks = Array.isArray(runnerConfig.extraNetworks) ? runnerConfig.extraNetworks : []

  return Boolean(
    preStart
    || buildCommand
    || Object.keys(env).length > 0
    || persistentPaths.length > 0
    || extraNetworks.length > 0
    || (baseImage && baseImage !== "node:20-alpine")
    || (workingDir && workingDir !== "/var/www/app")
    || (containerPort && containerPort !== "3000")
    || !!hostPort
    || !!runnerUser
    || (startCommand && startCommand !== "node .output/server/index.mjs")
  )
}

const applyRunnerPreset = (preset: string) => {
  runnerPreset.value = preset
  if (preset === "nuxt") {
    formModel.runnerContainerPort = "3000"
    formModel.runnerStartCommand = "node .output/server/index.mjs"
    if (!formModel.runnerWorkingDir) formModel.runnerWorkingDir = "/var/www/app"
  } else if (preset === "next") {
    formModel.runnerContainerPort = "3000"
    formModel.runnerStartCommand = "npm run start"
    if (!formModel.runnerWorkingDir) formModel.runnerWorkingDir = "/var/www/app"
  } else if (preset === "node") {
    formModel.runnerContainerPort = "3000"
    formModel.runnerStartCommand = "node server.js"
    if (!formModel.runnerWorkingDir) formModel.runnerWorkingDir = "/var/www/app"
  }
}

const detectRunnerPreset = (startCommand: string) => {
  const cmd = String(startCommand || "").trim()
  if (!cmd || cmd === "node .output/server/index.mjs") return "nuxt"
  if (cmd === "npm run start") return "next"
  if (cmd === "node server.js") return "node"
  return "custom"
}

const handleClose = () => {
  emit("update:show", false)
}

const handleSubmit = () => {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      loading.value = true
      try {
        const payload: any = { ...formModel }
        payload.runnerKey = normalizeRunnerKey(payload.runnerKey || payload.name || "")
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
          payload.runnerContainerPort = String(payload.runnerContainerPort || "").trim()
          payload.runnerHostPort = String(payload.runnerHostPort || "").trim()
          payload.runnerMode = "runner"
          payload.runnerConfig = {
            advanced: !!payload.runnerAdvanced,
            mode: payload.runnerPolicy || "build_run",
            baseImage: payload.runnerBaseImage || "node:20-alpine",
            workingDir: payload.runnerWorkingDir || "/var/www/app",
            containerPort: payload.runnerContainerPort || "3000",
            hostPort: payload.runnerHostPort || "",
            runnerUser: payload.runnerUser || "",
            buildCommand: payload.runnerInstallCommand || "",
            startCommand: payload.runnerStartCommand || "node .output/server/index.mjs",
            preStart: payload.runnerPreStart || "",
            env: parseEnvText(payload.runnerEnvText || ""),
            persistentPaths: parsePersistentPathsText(payload.runnerPersistentPathsText || ""),
            extraNetworks: parseExtraNetworksText(payload.runnerExtraNetworksText || "")
          }
        } else {
          payload.runnerMode = ""
          payload.runnerKey = ""
          payload.runnerConfig = undefined
        }
        
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
    }
  })
}

watch(() => props.show, (val) => {
  if (val) {
    loadRuntimePrecheck()
    if (props.editData) {
      const isHost = props.editData.buildImage === "host" || props.editData.buildImage === ""
      let runnerConfig: any = props.editData.runnerConfig || {}
      if (typeof runnerConfig === "string") {
        try {
          runnerConfig = JSON.parse(runnerConfig)
        } catch (e) {
          runnerConfig = {}
        }
      }
      const runnerEnabled = props.editData.runnerMode === "runner"
      const envObj = runnerConfig.env || {}
      const runnerEnvText = Object.keys(envObj).map((k: string) => `${k}=${envObj[k]}`).join("\n")
      const runnerPersistentPathsText = Array.isArray(runnerConfig.persistentPaths) ? runnerConfig.persistentPaths.join("\n") : ""
      const runnerExtraNetworksText = Array.isArray(runnerConfig.extraNetworks) ? runnerConfig.extraNetworks.join("\n") : ""
      const runnerAdvanced = inferRunnerAdvanced(runnerConfig)
      Object.assign(formModel, {
        name: props.editData.name || "",
        description: props.editData.description || "",
        repoUrl: props.editData.repoUrl || "",
        branch: props.editData.branch || "main",
        version: props.editData.version || "1.0.0",
        authType: props.editData.authType || "none",
        authData: props.editData.authData || "",
        pipelineMode: runnerEnabled ? "runner" : "script",
        buildEnv: isHost ? "host" : "container",
        buildImage: isHost ? "node:20-alpine" : props.editData.buildImage,
        buildScript: props.editData.buildScript || "",
        outputImage: props.editData.outputImage || "",
        artifactPath: props.editData.artifactPath || ".",
        exposePort: props.editData.exposePort || 80,
        runnerKey: props.editData.runnerKey || "",

        runnerEnabled,
        runnerPolicy: runnerConfig.mode || "build_run",
        runnerAdvanced,
        runnerBaseImage: runnerConfig.baseImage || "node:20-alpine",
        runnerWorkingDir: runnerConfig.workingDir || "/var/www/app",
        runnerContainerPort: String(runnerConfig.containerPort || "3000"),
        runnerHostPort: String(runnerConfig.hostPort || ""),
        runnerUser: runnerConfig.runnerUser || "",
        runnerInstallCommand: runnerConfig.buildCommand || "",
        runnerStartCommand: runnerConfig.startCommand || "node .output/server/index.mjs",
        runnerPreStart: runnerConfig.preStart || "",
        runnerEnvText,
        runnerPersistentPathsText,
        runnerExtraNetworksText
      })
      runnerKeyTouched.value = !!props.editData.runnerKey
      runnerPreset.value = detectRunnerPreset(formModel.runnerStartCommand)
    } else if (props.initialTemplate) {
      Object.assign(formModel, {
        name: props.initialTemplate.name || "",
        description: props.initialTemplate.description || "",
        repoUrl: "",
        branch: "main",
        version: "1.0.0",
        authType: "none",
        authData: "",
        pipelineMode: "script",
        buildEnv: props.initialTemplate.buildEnv || "container",
        buildImage: props.initialTemplate.buildImage || "node:20-alpine",
        buildScript: props.initialTemplate.buildScript || "",
        outputImage: "",
        artifactPath: props.initialTemplate.artifactPath || ".",
        exposePort: 80,
        runnerKey: "",

        runnerEnabled: false,
        runnerPolicy: "build_run",
        runnerAdvanced: false,
        runnerBaseImage: "node:20-alpine",
        runnerWorkingDir: "/var/www/app",
        runnerContainerPort: "3000",
        runnerHostPort: "",
        runnerUser: "",
        runnerInstallCommand: "",
        runnerStartCommand: "node .output/server/index.mjs",
        runnerPreStart: "",
        runnerEnvText: "",
        runnerPersistentPathsText: "",
        runnerExtraNetworksText: ""
      })
      runnerKeyTouched.value = false
      runnerPreset.value = "custom"
    } else {
      Object.assign(formModel, {
        name: "",
        description: "",
        repoUrl: "",
        branch: "main",
        version: "1.0.0",
        authType: "none",
        authData: "",
        pipelineMode: "runner",
        buildEnv: "container",
        buildImage: "node:20-alpine",
        buildScript: "npm install && npm run build",
        outputImage: "",
        artifactPath: ".",
        exposePort: 80,
        runnerKey: "",

        runnerEnabled: false,
        runnerPolicy: "build_run",
        runnerAdvanced: false,
        runnerBaseImage: "node:20-alpine",
        runnerWorkingDir: "/var/www/app",
        runnerContainerPort: "3000",
        runnerHostPort: "",
        runnerUser: "",
        runnerInstallCommand: "",
        runnerStartCommand: "node .output/server/index.mjs",
        runnerPreStart: "",
        runnerEnvText: "",
        runnerPersistentPathsText: "",
        runnerExtraNetworksText: ""
      })
      runnerKeyTouched.value = false
      runnerPreset.value = "nuxt"
    }
  }
})

watch(() => formModel.name, (val) => {
  if (formModel.pipelineMode !== "runner") return
  if (runnerKeyTouched.value) return
  formModel.runnerKey = normalizeRunnerKey(val)
})

watch(() => formModel.pipelineMode, (mode) => {
  if (mode === "runner" && !formModel.runnerKey && formModel.name) {
    formModel.runnerKey = normalizeRunnerKey(formModel.name)
  }
})
</script>

<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <FullModal
    :show="show"
    :title="isEdit ? '编辑流水线' : '新增流水线'"
    @update:show="handleClose"
    width="800px"
  >
    <n-form
      ref="formRef"
      :model="formModel"
      :rules="rules"
      label-placement="left"
      label-width="100"
    >
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
            <div class="ml-6 text-xs text-slate-500">适合大多数项目。流水线会把代码产物解压到版本目录，再交给运行时基础镜像执行；版本锚点来自产物与 Commit，而不是应用镜像 Tag。</div>
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

      <template v-if="formModel.pipelineMode === 'runner'">
        <div class="mb-4 mt-6 text-sm font-semibold text-slate-700">简单模式 (代码产物部署)</div>
        <div class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600">
          {{ runnerRuntimeHint }}
        </div>
        <n-form-item label="项目预设">
          <div class="w-full">
            <n-select
              :value="runnerPreset"
              :options="runnerPresetOptions"
              @update:value="applyRunnerPreset"
            />
            <div class="mt-2 text-xs text-slate-500">
              预设会自动填充常用启动命令与默认端口；选择“自定义”后可手动调整。这里的“版本”指产物版本与 Commit，不是应用镜像版本。
            </div>
          </div>
        </n-form-item>
        <n-form-item label="运行策略">
          <n-radio-group v-model:value="formModel.runnerPolicy">
            <n-space>
              <n-radio value="run">直接运行</n-radio>
              <n-radio value="build_run">打包后运行</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>

        <n-form-item
          label="流水线标识"
          path="runnerKey"
        >
          <div class="w-full">
            <n-input
              v-model:value="formModel.runnerKey"
              placeholder="例如：aipanel-site"
              @update:value="runnerKeyTouched = true"
            />
            <div class="mt-2 text-xs text-slate-500">
              用于生成 Runner 的固定数据目录：`安装目录/apps/流水线标识/`。创建或更新时会检查该目录是否已存在；若已被其他流水线占用，会提示你更换标识。
            </div>
          </div>
        </n-form-item>

        <n-form-item
          v-if="formModel.runnerPolicy === 'build_run'"
          label="安装命令"
        >
          <div class="w-full">
            <n-select
              v-model:value="formModel.runnerInstallCommand"
              :options="installCommandOptions"
            />
            <div class="mt-2 text-xs text-slate-500">
              默认自动检测 `package-lock.json` / `pnpm-lock.yaml` / `yarn.lock`。如果你的项目固定使用某个包管理器，也可以在这里指定。
            </div>
          </div>
        </n-form-item>

        <n-form-item label="产物来源">
          <div class="w-full">
            <n-input
              v-model:value="formModel.artifactPath"
              placeholder="默认：. （归档整个代码目录）"
            />
            <div class="mt-2 text-xs text-slate-500">
              代码产物部署使用策略 A：会先把产物归档为 ZIP，再解压到 releaseDir 后启动。默认使用 `.` 归档整个项目目录。
            </div>
          </div>
        </n-form-item>

        <n-form-item label="高级配置">

          <n-switch
            v-model:value="formModel.runnerAdvanced"
            class="mr-3"
          />

          <div class="text-xs text-slate-500">默认极简；开启后可自定义运行时基础镜像、工作目录、端口、启动命令、启动前脚本与环境变量。</div>
        </n-form-item>

        <template v-if="formModel.runnerAdvanced">
          <n-form-item label="运行时基础镜像">
            <div class="w-full">
              <n-input
                v-model:value="formModel.runnerBaseImage"
                placeholder="默认：node:20-alpine（建议填写固定版本标签，避免环境漂移）"
              />
              <div class="mt-2 text-xs text-slate-500">
                这里是运行时代码产物的基础镜像，不是你的应用版本镜像。生产环境建议使用固定版本标签，而不是长期依赖浮动 Tag。
              </div>
            </div>
          </n-form-item>

          <n-form-item label="工作目录">
            <n-input
              v-model:value="formModel.runnerWorkingDir"
              placeholder="默认：/var/www/app"
            />
          </n-form-item>

          <n-form-item
            label="容器端口"
            path="runnerContainerPort"
          >
            <n-input
              v-model:value="formModel.runnerContainerPort"
              placeholder="默认：3000"
            />
          </n-form-item>

          <n-form-item
            label="固定发布端口"
            path="runnerHostPort"
          >
            <div class="w-full">
              <n-input
                v-model:value="formModel.runnerHostPort"
                placeholder="留空则自动分配，例如：3101"
              />
              <div class="mt-2 text-xs text-slate-500">
                这是宿主机稳定入口端口，不是容器内部监听端口。留空时每次自动分配并同步网站代理；填写后会固定绑定到 `127.0.0.1:该端口`，后续发布无需再变更网站端口。
              </div>
            </div>
          </n-form-item>

          <n-form-item label="容器内运行用户">
            <div class="w-full">
              <n-input
                v-model:value="formModel.runnerUser"
                placeholder="留空则使用镜像默认用户，例如：node / 1000 / 1000:1000"
              />
              <div class="mt-2 text-xs text-slate-500">
                这里只控制容器内进程用户，不影响宿主机容器运行时。适合避免 `node:20-alpine` 这类镜像默认以 `root` 启动时写入 root 权限产物；是否 rootless / rootful 取决于当前面板命中的运行时。
              </div>
            </div>
          </n-form-item>

          <n-form-item label="启动命令">
            <n-input
              v-model:value="formModel.runnerStartCommand"
              placeholder="默认：node .output/server/index.mjs"
              @update:value="runnerPreset = 'custom'"
            />
          </n-form-item>

          <n-form-item label="启动前脚本 (可选)">
            <n-input
              type="textarea"
              v-model:value="formModel.runnerPreStart"
              placeholder="在容器内启动前执行，例如：生成配置、迁移脚本等（不要写 docker/compose）"
            />
          </n-form-item>

          <n-form-item label="环境变量 (可选)">
            <n-input
              type="textarea"
              v-model:value="formModel.runnerEnvText"
              placeholder="一行一个：KEY=VALUE"
            />
          </n-form-item>

          <n-form-item label="持久化目录 (可选)">
            <div class="w-full">
              <n-input
                type="textarea"
                v-model:value="formModel.runnerPersistentPathsText"
                placeholder="一行一个，例如：uploads&#10;.data&#10;storage"
              />
              <div class="mt-2 text-xs text-slate-500">
                只填写容器内需要持久化的子目录即可，例如 `uploads`、`.data`、`storage`。系统会自动映射到 `安装目录/apps/流水线标识/对应子目录`，并保持代码目录与数据目录分离。
              </div>
            </div>
          </n-form-item>

          <n-form-item label="额外网络 (可选)">
            <div class="w-full">
              <n-input
                type="textarea"
                v-model:value="formModel.runnerExtraNetworksText"
                placeholder="一行一个，例如：postgres-app_default"
              />
              <div class="mt-2 text-xs text-slate-500">
                默认会接入 `gopanel-network`。当你需要直连其他容器应用（如 PostgreSQL / MySQL / Redis）时，可在这里补充它所在的现有网络名，一行一个；系统不会自动创建不存在的网络。
              </div>
            </div>
          </n-form-item>
        </template>
      </template>

      <template v-else>
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
          label="默认访问端口"
          path="exposePort"
        >
          <div class="w-full">
            <n-input-number
              v-model:value="formModel.exposePort"
              placeholder="例如: 80"
            />
            <div class="mt-2 text-xs text-slate-500">
              单个输入即可。该值用于访问端口建议，不会修改容器内部监听端口；容器内部端口会按镜像的 EXPOSE 或 PORT 自动识别。
            </div>
          </div>
        </n-form-item>
      </template>
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
