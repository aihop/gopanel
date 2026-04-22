<script setup lang="ts">
import { ref, reactive, watch, computed } from "vue"
import { NModal, NForm, NFormItem, NInput, NSelect, NButton, NRadioGroup, NRadio, useMessage } from "naive-ui"
import { createPipeline, updatePipeline } from "@/api/modules/pipeline"
import { Pipeline } from "@/api/interface/pipeline"
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

const isEdit = computed(() => !!props.editData)

const formModel = reactive({
  name: "",
  description: "",
  repoUrl: "",
  branch: "main",
  version: "1.0.0", // 默认初始版本号
  authType: "none",
  authData: "",
  buildEnv: "container", // 新增前端状态：host | container
  buildImage: "node:20-alpine",
  buildScript: "npm install && npm run build",
  outputImage: "",
  artifactPath: "dist/",
  exposePort: 80 // 默认访问端口建议值（单个）
})

const rules = {
  name: { required: true, message: "请输入名称", trigger: "blur" },
  branch: { required: true, message: "请输入分支名称", trigger: "blur" },
  version: { required: true, message: "请输入初始版本号", trigger: "blur" }
}

const authOptions = [
  { label: "公开仓库 (无需凭证)", value: "none" },
  { label: "Token 凭证 (推荐)", value: "token" },
  { label: "账号密码", value: "password" }
]

const handleClose = () => {
  emit("update:show", false)
}

const handleSubmit = () => {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      loading.value = true
      try {
        const payload = { ...formModel }
        if (payload.buildEnv === "host") {
          payload.buildImage = "host"
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
    if (props.editData) {
      const isHost = props.editData.buildImage === "host" || props.editData.buildImage === ""
      Object.assign(formModel, {
        name: props.editData.name || "",
        description: props.editData.description || "",
        repoUrl: props.editData.repoUrl || "",
        branch: props.editData.branch || "main",
        version: props.editData.version || "1.0.0",
        authType: props.editData.authType || "none",
        authData: props.editData.authData || "",
        buildEnv: isHost ? "host" : "container",
        buildImage: isHost ? "node:20-alpine" : props.editData.buildImage,
        buildScript: props.editData.buildScript || "",
        outputImage: props.editData.outputImage || "",
        artifactPath: props.editData.artifactPath || "",
        exposePort: props.editData.exposePort || 80
      })
    } else if (props.initialTemplate) {
      Object.assign(formModel, {
        name: props.initialTemplate.name || "",
        description: props.initialTemplate.description || "",
        repoUrl: "",
        branch: "main",
        version: "1.0.0",
        authType: "none",
        authData: "",
        buildEnv: props.initialTemplate.buildEnv || "container",
        buildImage: props.initialTemplate.buildImage || "node:20-alpine",
        buildScript: props.initialTemplate.buildScript || "",
        outputImage: "",
        artifactPath: props.initialTemplate.artifactPath || "dist/",
        exposePort: 80
      })
    } else {
      Object.assign(formModel, {
        name: "",
        description: "",
        repoUrl: "",
        branch: "main",
        version: "1.0.0",
        authType: "none",
        authData: "",
        buildEnv: "container",
        buildImage: "node:20-alpine",
        buildScript: "npm install && npm run build",
        outputImage: "",
        artifactPath: "dist/",
        exposePort: 80
      })
    }
  }
})
</script>

<template>
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

      <div class="mb-4 mt-6 text-sm font-semibold text-slate-700">构建与部署</div>
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
