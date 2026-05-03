<template>
  <n-drawer
    :show="show"
    :width="800"
    :mask-closable="false"
    placement="right"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content
      title="创建编排"
      closable
    >
      <n-form :model="composeForm">
        <n-form-item
          :label="$t('database.source')"
          path="source"
        >
          <n-radio-group
            v-model:value="composeForm.source"
            name="composeSource"
          >
            <n-radio-button value="editor" label="编辑" />
            <n-radio-button value="path" label="路径选择" />
          </n-radio-group>
        </n-form-item>

        <div v-if="composeForm.source === 'editor'">
          <n-form-item label="文件夹" path="projectName">
            <div class="flex w-full flex-col gap-1">
              <n-input v-model:value="composeForm.projectName" placeholder="文件夹" />
              <div class="mt-1 text-xs text-[#adb0bc]">
                配置文件保存路径: {{ baseDir }}{{ composeForm.projectName ? composeForm.projectName + "/" : "" }}
              </div>
            </div>
          </n-form-item>

          <n-tabs
            :value="activeTab"
            type="line"
            default-value="compose-definition"
            @update:value="emit('update:active-tab', $event)"
          >
            <n-tab-pane name="compose-definition" tab="编辑">
              <FtEditor
                v-model="composeForm.composeContent"
                language="yaml"
                height="350px"
              />
            </n-tab-pane>
            <n-tab-pane name="compose-logs" tab="日志">
              <div ref="logBoxRef" class="compose-log-box">
                {{ logContent }}
              </div>
            </n-tab-pane>
          </n-tabs>

          <div class="mt-6">
            <n-text strong>环境变量</n-text>
            <n-input
              v-model:value="composeForm.envContent"
              class="mt-2"
              type="textarea"
              :placeholder="envPlaceholder"
              :autosize="{ minRows: 4, maxRows: 8 }"
            />
            <div class="mt-3 text-[13px] leading-6">
              <n-text depth="3">
                注意: 设置的环境变量会写入 <code>.env</code> 文件 (位于项目目录下)。 默认会自动引用。
              </n-text>
            </div>
          </div>
        </div>

        <div v-if="composeForm.source === 'path'">
          <n-form-item label="编排文件路径" path="pathValue">
            <n-input v-model:value="composeForm.pathValue" placeholder="例: /tmp/docker-compose.yml" />
          </n-form-item>
          <n-alert
            title="提示"
            type="info"
            :show-icon="false"
            class="mb-6 mt-[10px]"
          >
            将从指定路径读取编排文件内容。环境变量文件 (如 .env ) 应位于同一目录下。
          </n-alert>

          <div class="mt-6">
            <n-text strong>环境变量</n-text>
            <n-input
              v-model:value="composeForm.envContent"
              class="mt-2"
              type="textarea"
              :placeholder="envPlaceholder"
              :autosize="{ minRows: 4, maxRows: 8 }"
            />
            <div class="mt-3 text-[13px] leading-6">
              <n-text depth="3">
                注意: 设置的环境变量会写入 <code>.env</code> 文件 (位于项目目录下)。 默认会自动引用。
              </n-text>
            </div>
          </div>
        </div>
      </n-form>
      <template #footer>
        <n-button class="mr-2" @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" @click="emit('confirm')">确认</n-button>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import FtEditor from "@/components/FtEditor/index.vue"
import { nextTick, ref, watch } from "vue"

const props = defineProps<{
  show: boolean
  baseDir: string
  composeForm: {
    source: string
    projectName: string
    composeContent: string
    envContent: string
    pathValue: string
    selectedTemplateId: string | null
  }
  activeTab: string
  logContent: string
  envPlaceholder: string
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "update:active-tab", value: string): void
  (e: "confirm"): void
}>()

const logBoxRef = ref<HTMLElement | null>(null)

watch(
  () => props.logContent,
  () => {
    nextTick(() => {
      if (logBoxRef.value) {
        logBoxRef.value.scrollTop = logBoxRef.value.scrollHeight
      }
    })
  }
)
</script>

<style scoped>
.compose-log-box {
  background: #181818;
  color: #e0e0e0;
  padding: 12px;
  border-radius: 4px;
  min-height: 350px;
  max-height: 400px;
  overflow: auto;
  font-family: monospace;
  font-size: 13px;
  white-space: pre-line;
}
</style>
