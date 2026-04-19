<script setup lang="ts">
import { ref } from "vue"
import CommonPage from "@/components/page/Common.vue"
import PipelineList from "./src/PipelineList.vue"
import CreatePipelineModal from "./src/CreatePipelineModal.vue"
import TemplateWizardModal from "./src/TemplateWizardModal.vue"
import { NButton, NDropdown, NIcon } from "naive-ui"
import { useAuthStore } from "@/store/auth"
import { computed } from "vue"
import { renderIcon } from "@/utils"

const authStore = useAuthStore()
const isSubAdmin = computed(() => authStore.user?.role === 'SUB_ADMIN')

const createPipelineModalShow = ref(false)
const templateWizardShow = ref(false)
const currentTemplateType = ref('')
const pipelineListRef = ref()
const currentEditPipeline = ref(null)
const initialTemplate = ref<any>(null)

const handleSuccess = () => {
  if (pipelineListRef.value) {
    pipelineListRef.value.refresh()
  }
}

const handleEdit = (row: any) => {
  currentEditPipeline.value = row
  initialTemplate.value = null
  createPipelineModalShow.value = true
}

const handleCreate = () => {
  currentEditPipeline.value = null
  initialTemplate.value = null
  createPipelineModalShow.value = true
}

const handleTemplateCreate = (key: string) => {
  currentEditPipeline.value = null
  
  if (key === 'php-8') {
    currentTemplateType.value = 'php'
    templateWizardShow.value = true
  } else if (key === 'node-18') {
    currentTemplateType.value = 'node'
    templateWizardShow.value = true
  } else if (key === 'go-1.21') {
    currentTemplateType.value = 'go'
    templateWizardShow.value = true
  } else if (key === 'java-17') {
    currentTemplateType.value = 'java'
    templateWizardShow.value = true
  }
}

const onTemplateGenerate = (config: any) => {
  initialTemplate.value = config
  createPipelineModalShow.value = true
}

const templateOptions = [
  {
    label: "Node.js (Vue/React) 环境",
    key: "node-18",
    icon: renderIcon("mdi:nodejs")
  },
  {
    label: "PHP (Composer) 环境",
    key: "php-8",
    icon: renderIcon("mdi:language-php")
  },
  {
    label: "Go (编译) 环境",
    key: "go-1.21",
    icon: renderIcon("mdi:language-go")
  },
  {
    label: "Java (Maven) 环境",
    key: "java-17",
    icon: renderIcon("mdi:language-java")
  }
]
</script>

<template>
  <div class="mt-4">
    <common-page
      show-header
      show-footer
    >
      <template #header>
        <div class="space-y-8 px-4">
          <div class="flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
            <div class="space-y-3">
              <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
                CI/CD Pipeline
              </div>
              <div class="text-2xl font-semibold fg-base-100">{{ $t('pipeline.codePipeline') }}</div>
              <div class="text-sm leading-7 text-slate-500">
                {{ $t('pipeline.codePipelineHelper') }}
              </div>
            </div>
            <div class="flex flex-col gap-4 xl:min-w-[420px] xl:items-end">
              <div
                class="flex w-full items-center justify-between gap-4 rounded-[22px] border border-slate-200 bg-base-100  px-5 py-4"
                v-if="!isSubAdmin"
              >
                <div>
                  <div class="text-sm font-semibold fg-base-100">{{ $t('pipeline.createPipeline') }}</div>
                  <div class="mt-1 text-sm text-slate-500">{{ $t('pipeline.createPipelineHelper') }}</div>
                </div>
                <div class="flex gap-2">
                  <n-dropdown
                    trigger="hover"
                    :options="templateOptions"
                    @select="handleTemplateCreate"
                    placement="bottom-end"
                  >
                    <n-button
                      type="primary"
                      ghost
                      size="large"
                      class="!rounded-[18px] px-4"
                    >
                      <template #icon>
                        <n-icon :component="renderIcon('mdi:chevron-down')" />
                      </template>
                      环境模板
                    </n-button>
                  </n-dropdown>
                  <n-button
                    type="primary"
                    size="large"
                    class="!rounded-[18px] px-6 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
                    @click="handleCreate"
                  >
                    {{ $t('pipeline.addPipeline') }}
                  </n-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
      <template #tabbar></template>
      <n-flex
        vertical
        class="rounded-[28px] border border-slate-100 bg-slate-50/70 p-4 sm:p-6"
      >
        <PipelineList
          ref="pipelineListRef"
          @edit="handleEdit"
        />
      </n-flex>
    </common-page>
    <CreatePipelineModal
      v-model:show="createPipelineModalShow"
      :edit-data="currentEditPipeline"
      :initial-template="initialTemplate"
      @success="handleSuccess"
    />
    <TemplateWizardModal
      v-model:show="templateWizardShow"
      :template-type="currentTemplateType"
      @generate="onTemplateGenerate"
    />
  </div>
</template>
