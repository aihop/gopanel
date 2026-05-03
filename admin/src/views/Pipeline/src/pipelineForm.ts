import type { Pipeline } from "@/api/interface/pipeline"

export interface PipelineFormModel {
  name: string
  description: string
  repoUrl: string
  branch: string
  version: string
  authType: string
  authData: string
  pipelineMode: string
  buildEnv: string
  buildImage: string
  buildScript: string
  outputImage: string
  artifactPath: string
  exposePort: number | null
  pipelineKey: string
  runnerEnabled: boolean
  runnerPolicy: "run" | "build_run"
  runnerAdvanced: boolean
  runnerBaseImage: string
  runnerWorkingDir: string
  runnerContainerPort: string
  runnerHostPort: string
  runnerUser: string
  runnerBuildCommand: string
  runnerStartCommand: string
  runnerPreStart: string
  runnerEnvText: string
  runnerPersistentPathsText: string
  runnerExtraNetworksText: string
}

type RunnerPresetConfig = {
  policy: "run" | "build_run"
  baseImage: string
  containerPort: string
  startCommand: string
  buildCommand: string
}

export const authOptions = [
  { label: "公开仓库 (无需凭证)", value: "none" },
  { label: "Token 凭证 (推荐)", value: "token" },
  { label: "账号密码", value: "password" }
]

export const runnerPresetOptions = [
  { label: "Nuxt (推荐)", value: "nuxt" },
  { label: "Next.js", value: "next" },
  { label: "Node 通用", value: "node" },
  { label: "Go 通用", value: "go" },
  { label: "Python 通用", value: "python" },
  { label: "PHP 通用", value: "php" },
  { label: "自定义", value: "custom" }
]

export const runnerPresetDefaults: Record<string, RunnerPresetConfig> = {
  nuxt: {
    policy: "build_run",
    baseImage: "node:20-alpine",
    containerPort: "3000",
    startCommand: "node .output/server/index.mjs",
    buildCommand: ""
  },
  next: {
    policy: "build_run",
    baseImage: "node:20-alpine",
    containerPort: "3000",
    startCommand: "npm run start",
    buildCommand: ""
  },
  node: {
    policy: "build_run",
    baseImage: "node:20-alpine",
    containerPort: "3000",
    startCommand: "node server.js",
    buildCommand: ""
  },
  go: {
    policy: "build_run",
    baseImage: "golang:1.25.1-alpine",
    containerPort: "8080",
    startCommand: "./app",
    buildCommand: "go mod download && (CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server 2>/dev/null || CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .)"
  },
  python: {
    policy: "build_run",
    baseImage: "python:3.11-slim",
    containerPort: "8000",
    startCommand: "python app.py",
    buildCommand: "if [ -f requirements.txt ]; then pip install -r requirements.txt; elif [ -f pyproject.toml ]; then pip install .; else echo '未检测到 requirements.txt / pyproject.toml，跳过依赖安装'; fi"
  },
  php: {
    policy: "build_run",
    baseImage: "composer:2",
    containerPort: "8000",
    startCommand: "php -S 0.0.0.0:${PORT:-8000} -t public",
    buildCommand: "if [ -f composer.json ]; then composer install --no-dev --optimize-autoloader; else echo '未检测到 composer.json，跳过 Composer 安装'; fi"
  }
}

export const createDefaultPipelineFormModel = (): PipelineFormModel => ({
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
  exposePort: null,
  pipelineKey: "",
  runnerEnabled: false,
  runnerPolicy: "build_run",
  runnerAdvanced: false,
  runnerBaseImage: "node:20-alpine",
  runnerWorkingDir: "/var/www/app",
  runnerContainerPort: "3000",
  runnerHostPort: "",
  runnerUser: "",
  runnerBuildCommand: "",
  runnerStartCommand: "node .output/server/index.mjs",
  runnerPreStart: "",
  runnerEnvText: "",
  runnerPersistentPathsText: "",
  runnerExtraNetworksText: ""
})

export const getRunnerBuildCommandPlaceholder = (preset: string) => {
  switch (preset) {
    case "go":
      return "例如：go mod download && go build -o app ."
    case "python":
      return "例如：pip install -r requirements.txt"
    case "php":
      return "例如：composer install --no-dev --optimize-autoloader"
    default:
      return "留空则按默认 Node 规则自动构建；也可填写自定义构建命令"
  }
}

export const getRunnerBuildCommandHint = (preset: string) => {
  switch (preset) {
    case "go":
      return "Go 预设默认会先 `go mod download`，再尝试 `./cmd/server` 和项目根目录两种常见编译入口。"
    case "python":
      return "Python 预设默认优先安装 `requirements.txt`，若不存在则尝试按 `pyproject.toml` 安装。"
    case "php":
      return "PHP 预设默认在存在 `composer.json` 时执行 `composer install`，适合 Laravel / ThinkPHP 等常见项目。"
    default:
      return "留空时仍会按 Node 项目的 `package.json / .output` 规则自动构建；填写后将优先执行你写的命令。"
  }
}

export const parseEnvText = (text: string) => {
  const env: Record<string, string> = {}
  for (const lineRaw of String(text || "").split("\n")) {
    const line = String(lineRaw || "").trim()
    if (!line) continue
    const idx = line.indexOf("=")
    if (idx <= 0) continue
    const key = line.slice(0, idx).trim()
    if (!key) continue
    env[key] = line.slice(idx + 1)
  }
  return env
}

const parseUniqueMultilineText = (text: string) => {
  const items: string[] = []
  const seen = new Set<string>()
  for (const lineRaw of String(text || "").split("\n")) {
    const line = String(lineRaw || "").trim()
    if (!line || seen.has(line)) continue
    seen.add(line)
    items.push(line)
  }
  return items
}

export const parsePersistentPathsText = (text: string) => parseUniqueMultilineText(text)

export const parseExtraNetworksText = (text: string) => parseUniqueMultilineText(text)

export const normalizePipelineKey = (text: string) => {
  const raw = String(text || "").trim().toLowerCase()
  if (!raw) return ""
  return raw
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^[-_]+|[-_]+$/g, "")
}

export const hasForbiddenRunnerPersistentPath = (items: string[]) => {
  return items.some((item) => {
    const normalized = String(item || "")
      .trim()
      .replace(/\\/g, "/")
      .replace(/^\/+/, "")
      .replace(/\/+/g, "/")
    return normalized === "node_modules" || normalized.startsWith("node_modules/")
  })
}

export const inferRunnerAdvanced = (runnerConfig: any) => {
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

export const applyRunnerPresetToForm = (formModel: PipelineFormModel, preset: string) => {
  const presetConfig = runnerPresetDefaults[preset]
  if (!presetConfig) return
  formModel.runnerPolicy = presetConfig.policy
  formModel.runnerBaseImage = presetConfig.baseImage
  formModel.runnerContainerPort = presetConfig.containerPort
  formModel.runnerBuildCommand = presetConfig.buildCommand
  formModel.runnerStartCommand = presetConfig.startCommand
  if (!formModel.runnerWorkingDir) {
    formModel.runnerWorkingDir = "/var/www/app"
  }
}

export const detectRunnerPresetFromConfig = (runnerConfig: any) => {
  const baseImage = String(runnerConfig?.baseImage || "").trim()
  const startCommand = String(runnerConfig?.startCommand || "").trim()
  const buildCommand = String(runnerConfig?.buildCommand || "").trim()

  for (const [preset, config] of Object.entries(runnerPresetDefaults)) {
    if (config.baseImage === baseImage && config.startCommand === startCommand && config.buildCommand === buildCommand) {
      return preset
    }
  }

  if (!startCommand || startCommand === "node .output/server/index.mjs") return "nuxt"
  if (startCommand === "npm run start") return "next"
  if (startCommand === "node server.js") return "node"
  if (startCommand === "./app") return "go"
  if (startCommand === "python app.py") return "python"
  if (startCommand === "php -S 0.0.0.0:${PORT:-8000} -t public") return "php"
  return "custom"
}

const parseRunnerConfig = (runnerConfig: any) => {
  if (typeof runnerConfig === "string") {
    try {
      return JSON.parse(runnerConfig)
    } catch (error) {
      return {}
    }
  }
  if (runnerConfig && typeof runnerConfig === "object") {
    return runnerConfig
  }
  return {}
}

export const createPipelineFormFromEdit = (editData: Pipeline.ResPipeline) => {
  const isHost = editData.buildImage === "host" || editData.buildImage === ""
  const runnerConfig = parseRunnerConfig(editData.runnerConfig)
  const runnerEnabled = editData.runnerMode === "runner"
  const env = runnerConfig.env && typeof runnerConfig.env === "object" ? runnerConfig.env : {}

  return {
    form: {
      name: editData.name || "",
      description: editData.description || "",
      repoUrl: editData.repoUrl || "",
      branch: editData.branch || "main",
      version: editData.version || "1.0.0",
      authType: editData.authType || "none",
      authData: editData.authData || "",
      pipelineMode: runnerEnabled ? "runner" : "script",
      buildEnv: isHost ? "host" : "container",
      buildImage: isHost ? "node:20-alpine" : editData.buildImage,
      buildScript: editData.buildScript || "",
      outputImage: editData.outputImage || "",
      artifactPath: editData.artifactPath || ".",
      exposePort: editData.exposePort || null,
      pipelineKey: editData.pipelineKey || "",
      runnerEnabled,
      runnerPolicy: runnerConfig.mode || "build_run",
      runnerAdvanced: inferRunnerAdvanced(runnerConfig),
      runnerBaseImage: runnerConfig.baseImage || "node:20-alpine",
      runnerWorkingDir: runnerConfig.workingDir || "/var/www/app",
      runnerContainerPort: String(runnerConfig.containerPort || "3000"),
      runnerHostPort: String(runnerConfig.hostPort || ""),
      runnerUser: runnerConfig.runnerUser || "",
      runnerBuildCommand: runnerConfig.buildCommand || "",
      runnerStartCommand: runnerConfig.startCommand || "node .output/server/index.mjs",
      runnerPreStart: runnerConfig.preStart || "",
      runnerEnvText: Object.keys(env).map((key) => `${key}=${env[key]}`).join("\n"),
      runnerPersistentPathsText: Array.isArray(runnerConfig.persistentPaths) ? runnerConfig.persistentPaths.join("\n") : "",
      runnerExtraNetworksText: Array.isArray(runnerConfig.extraNetworks) ? runnerConfig.extraNetworks.join("\n") : ""
    } satisfies PipelineFormModel,
    pipelineKeyTouched: !!editData.pipelineKey,
    runnerPreset: detectRunnerPresetFromConfig(runnerConfig)
  }
}

export const createPipelineFormFromTemplate = (initialTemplate: any): PipelineFormModel => ({
  ...createDefaultPipelineFormModel(),
  name: initialTemplate?.name || "",
  description: initialTemplate?.description || "",
  pipelineMode: "script",
  buildEnv: initialTemplate?.buildEnv || "container",
  buildImage: initialTemplate?.buildImage || "node:20-alpine",
  buildScript: initialTemplate?.buildScript || "",
  artifactPath: initialTemplate?.artifactPath || "."
})
