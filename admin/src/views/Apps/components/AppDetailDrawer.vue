<script setup lang="ts">
import { MdPreview } from "md-editor-v3"
import "md-editor-v3/lib/preview.css"

defineProps<{
  show: boolean
  detailApp: any
  detailLoading: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "install"): void
}>()
</script>

<template>
  <n-drawer
    :show="show"
    :width="700"
    placement="right"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content :title="detailApp?.name + $t('app.detail')">
      <n-spin :show="detailLoading">
        <div
          v-if="detailApp"
          class="app-detail-container"
        >
          <div class="mb-6 flex items-center">
            <img
              v-if="detailApp.icon"
              :src="detailApp.icon"
              class="mr-4 h-16 w-16"
            />
            <div>
              <div class="mb-1 text-xl font-bold">{{ detailApp.name }}</div>
              <div class="text-sm text-gray-500">{{ detailApp.shortDescZh || detailApp.description }}</div>
            </div>
          </div>

          <n-descriptions
            label-placement="left"
            :column="1"
            bordered
            size="small"
            class="mb-6"
          >
            <n-descriptions-item label="分类">{{ detailApp.type }}</n-descriptions-item>
            <n-descriptions-item
              v-if="detailApp.versions && detailApp.versions.length"
              label="可选版本"
            >
              <n-space>
                <n-tag
                  v-for="v in detailApp.versions"
                  :key="v"
                  size="small"
                  type="info"
                >{{ v }}</n-tag>
              </n-space>
            </n-descriptions-item>
            <n-descriptions-item
              v-if="detailApp.website || detailApp.document || detailApp.github"
              label="相关链接"
            >
              <n-space>
                <n-button
                  v-if="detailApp.website"
                  text
                  tag="a"
                  :href="detailApp.website"
                  target="_blank"
                  type="primary"
                >官网</n-button>
                <n-button
                  v-if="detailApp.document"
                  text
                  tag="a"
                  :href="detailApp.document"
                  target="_blank"
                  type="primary"
                >文档</n-button>
                <n-button
                  v-if="detailApp.github"
                  text
                  tag="a"
                  :href="detailApp.github"
                  target="_blank"
                  type="primary"
                >GitHub</n-button>
              </n-space>
            </n-descriptions-item>
          </n-descriptions>

          <div v-if="detailApp.readMe">
            <div class="mb-4 text-lg font-bold">应用介绍</div>
            <MdPreview
              editor-id="app-readme"
              :model-value="detailApp.readMe"
            />
          </div>
        </div>
      </n-spin>
      <template #footer>
        <n-space>
          <n-button @click="emit('update:show', false)">关闭</n-button>
          <n-button
            v-if="detailApp?.installed"
            type="info"
            secondary
            disabled
          >已安装</n-button>
          <n-button
            v-else
            type="primary"
            :disabled="detailApp?.installing"
            @click="emit('install')"
          >{{ detailApp?.installing ? '安装中' : '去安装' }}</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>
