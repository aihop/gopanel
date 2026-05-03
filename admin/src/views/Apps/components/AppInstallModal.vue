<script setup lang="ts">
import { ref } from "vue"

defineProps<{
  show: boolean
  currentApp: any
  installLoading: boolean
  formModel: Record<string, any>
  versionOptions: any[]
  formFields: any[]
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "version-change", value: string): void
  (e: "submit"): void
}>()

const formRef = ref<any>(null)

const rules = {
  name: { required: true, message: "请输入应用名称", trigger: "blur" },
  version: { required: true, message: "请选择版本", trigger: "change" }
}

const handleSubmit = () => {
  formRef.value?.validate((errors: any) => {
    if (!errors) {
      emit("submit")
    }
  })
}
</script>

<template>
  <n-modal
    :show="show"
    preset="dialog"
    :title="`安装 ${currentApp?.name}`"
    style="width: 600px"
    @update:show="emit('update:show', $event)"
  >
    <n-spin :show="installLoading">
      <n-form
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item
          label="名称"
          path="name"
        >
          <n-input
            v-model:value="formModel.name"
            placeholder="请输入应用名称"
          />
        </n-form-item>
        <n-form-item
          label="版本"
          path="version"
        >
          <n-select
            v-model:value="formModel.version"
            :options="versionOptions"
            placeholder="请选择版本"
            @update:value="emit('version-change', $event)"
          />
        </n-form-item>

        <template v-if="formFields && formFields.length > 0">
          <n-divider>参数配置</n-divider>
          <n-form-item
            v-for="field in formFields"
            :key="field.envKey"
            :label="field.labelZh || field.label?.zh || field.labelEn || field.envKey"
            :path="'params.' + field.envKey"
          >
            <template v-if="field.type === 'number'">
              <n-input-number
                v-model:value="formModel.params[field.envKey]"
                :min="field.min"
                :max="field.max"
                style="width: 100%"
              />
            </template>
            <template v-else-if="field.type === 'password'">
              <n-input
                v-model:value="formModel.params[field.envKey]"
                type="password"
                show-password-on="click"
              />
            </template>
            <template v-else-if="field.type === 'select'">
              <n-select
                v-model:value="formModel.params[field.envKey]"
                :options="(field.values || []).map((v: any) => ({ label: v.label, value: v.value }))"
              />
            </template>
            <template v-else>
              <n-input v-model:value="formModel.params[field.envKey]" />
            </template>

            <template
              v-if="field.description"
              #feedback
            >
              <div style="font-size: 12px; color: #999; margin-top: 4px;">
                {{ field.description.zh || field.description.en || field.description }}
              </div>
            </template>
          </n-form-item>
        </template>

        <n-collapse>
          <n-collapse-item
            title="高级配置"
            name="advanced"
          >
            <n-form-item label="允许端口外部访问">
              <n-switch v-model:value="formModel.allowPort" />
            </n-form-item>
            <n-form-item label="容器名称">
              <n-input
                v-model:value="formModel.containerName"
                placeholder="默认自动生成"
              />
            </n-form-item>
            <n-form-item label="CPU 限制(核)">
              <n-input-number
                v-model:value="formModel.cpuQuota"
                :min="0"
                :step="0.1"
                style="width: 100%"
              />
            </n-form-item>
            <n-form-item label="内存限制">
              <n-input-group>
                <n-input-number
                  v-model:value="formModel.memoryLimit"
                  :min="0"
                  style="width: 70%"
                />
                <n-select
                  v-model:value="formModel.memoryUnit"
                  :options="[
                    { label: 'MB', value: 'm' },
                    { label: 'GB', value: 'g' }
                  ]"
                  style="width: 30%"
                />
              </n-input-group>
            </n-form-item>
          </n-collapse-item>
        </n-collapse>
      </n-form>
    </n-spin>
    <template #action>
      <n-button @click="emit('update:show', false)">取消</n-button>
      <n-button
        type="primary"
        :loading="installLoading"
        @click="handleSubmit"
      >确认安装</n-button>
    </template>
  </n-modal>
</template>
