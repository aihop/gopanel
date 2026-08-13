<script setup lang="ts">
import type { PipelineFormModel } from "./pipelineForm"
import FtEditor from "@/components/FtEditor/index.vue"

defineProps<{
  formModel: PipelineFormModel
  isEdit: boolean
  existingRuntimeHint: string
  existingRunnerDirectoryHint: string
  runnerPreset: string
  runnerPresetOptions: Array<{ label: string; value: string }>
  runnerPresetAutoLabel: string
  runnerPresetHits: string[]
  runnerBuildCommandPlaceholder: string
  runnerBuildCommandHint: string
}>()

const emit = defineEmits<{
  (e: "apply-runner-preset", preset: string): void
  (e: "mark-runner-preset-custom"): void
}>()

const handlePresetChange = (value: string) => {
  emit("apply-runner-preset", value)
}

const markCustomPreset = () => {
  emit("mark-runner-preset-custom")
}
</script>

<template>
  <div
    v-if="isEdit && existingRuntimeHint"
    class="mb-4 rounded-2xl border border-blue-200 bg-blue-50 px-4 py-4 text-sm text-slate-700"
  >
    {{ existingRuntimeHint }}
  </div>
  <div
    v-if="isEdit && existingRunnerDirectoryHint"
    class="mb-4 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-4 text-sm text-slate-700"
  >
    {{ existingRunnerDirectoryHint }}
  </div>
  <n-form-item label="项目预设">
    <div class="w-full">
      <n-select
        :value="runnerPreset"
        :options="runnerPresetOptions"
        @update:value="handlePresetChange"
      />
      <div class="mt-2 text-xs text-slate-500">
        预设会自动填充常用基础镜像、构建命令、启动命令与默认端口；Go / Python / PHP 也可直接走简单模式。这里的“版本”指产物版本与 Commit，不是应用镜像版本。
      </div>
      <div
        v-if="runnerPresetAutoLabel"
        class="mt-2 text-xs text-emerald-600"
      >
        已根据仓库内容自动推荐为 `{{ runnerPresetAutoLabel }}` 预设
        <span v-if="runnerPresetHits.length">，识别依据：{{ runnerPresetHits.join("、") }}</span>
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
    v-if="formModel.runnerPolicy === 'build_run'"
    label="构建命令"
  >
    <div class="w-full">
      <FtEditor
        v-model="formModel.runnerBuildCommand"
        language="bash"
        height="200px"
        :placeholder="runnerBuildCommandPlaceholder"
      />
      <div class="mt-2 text-xs text-slate-500">
        {{ runnerBuildCommandHint }}
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
    <div class="text-xs text-slate-500">默认极简；预设已经会自动填充常用值。开启后可继续细调基础镜像、端口、启动命令、启动前脚本与环境变量。</div>
  </n-form-item>

  <template v-if="formModel.runnerAdvanced">
    <n-form-item label="运行时基础镜像">
      <div class="w-full">
        <n-input
          v-model:value="formModel.runnerBaseImage"
          placeholder="例如：node:20-alpine / golang:1.22-alpine / python:3.11-slim"
          @update:value="markCustomPreset"
        />
        <div class="mt-2 text-xs text-slate-500">
          这里是运行时代码产物的基础镜像，不是你的应用版本镜像。生产环境建议使用固定版本标签，而不是长期依赖浮动 Tag。
        </div>
      </div>
    </n-form-item>

    <n-form-item label="工作目录">
      <div class="w-full">
        <n-input
          v-model:value="formModel.runnerWorkingDir"
          placeholder="默认：/var/www/app"
        />
        <div class="mt-2 text-xs text-slate-500">
          保持默认值时，Runner 会沿用兼容链路：代码源先挂到只读中间目录，再同步到运行目录。填写其他目录时，代码源会直接挂到你指定的工作目录。
        </div>
      </div>
    </n-form-item>

    <n-form-item
      label="容器端口"
      path="runnerContainerPort"
    >
      <n-input
        v-model:value="formModel.runnerContainerPort"
        placeholder="例如：3000 / 8080 / 8000"
        @update:value="markCustomPreset"
      />
    </n-form-item>

    <n-form-item
      label="发布端口"
      path="runnerHostPort"
    >
      <div class="w-full">
        <n-input
          v-model:value="formModel.runnerHostPort"
          placeholder="留空则自动分配，例如：3101"
        />
        <div class="mt-2 text-xs text-slate-500">
          这是容器外映射的默认端口，如不填写将默认获取，无非特别需要，不建议填写
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
          这里只控制容器内进程用户，不影响宿主机容器运行时。若使用非 root 用户，请不要持久化 `node_modules`，否则 npm 容易报权限错
        </div>
      </div>
    </n-form-item>

    <n-form-item label="启动命令">
      <n-input
        v-model:value="formModel.runnerStartCommand"
        placeholder="例如：node server.js / ./app / python app.py"
        @update:value="markCustomPreset"
      />
    </n-form-item>

    <n-form-item label="启动前脚本 (可选)">
      <n-input
        v-model:value="formModel.runnerPreStart"
        type="textarea"
        placeholder="在容器内启动前执行，例如：生成配置、迁移脚本等（不要写 docker/compose）"
      />
    </n-form-item>

    <n-form-item label="环境变量 (可选)">
      <n-input
        v-model:value="formModel.runnerEnvText"
        type="textarea"
        placeholder="一行一个：KEY=VALUE"
      />
    </n-form-item>

    <n-form-item label="持久化目录 (可选)">
      <div class="w-full">
        <n-input
          v-model:value="formModel.runnerPersistentPathsText"
          type="textarea"
          placeholder="一行一个，例如：uploads&#10;.data&#10;storage"
        />
        <div class="mt-2 text-xs text-slate-500">
          只填写容器内需要持久化的子目录即可，例如 uploads、.data、storage 
        </div>
      </div>
    </n-form-item>

    <n-form-item label="额外网络 (可选)">
      <div class="w-full">
        <n-input
          v-model:value="formModel.runnerExtraNetworksText"
          type="textarea"
          placeholder="一行一个，例如：postgres-app_default"
        />
        <div class="mt-2 text-xs text-slate-500">
          默认会接入 `gopanel-network`。当你需要直连其他容器应用（如 PostgreSQL / MySQL / Redis）时，可在这里补充它所在的现有网络名，一行一个；系统不会自动创建不存在的网络。
        </div>
      </div>
    </n-form-item>
  </template>
</template>
