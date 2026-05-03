<script setup lang="ts">
const props = defineProps<{
  path: string
  search: string
  showHidden: boolean
  expand: boolean
  copiedPath: boolean
  pathSegments: string[]
}>()

const emit = defineEmits<{
  (e: "update:path", value: string): void
  (e: "update:search", value: string): void
  (e: "update:show-hidden", value: boolean): void
  (e: "update:expand", value: boolean): void
  (e: "load"): void
  (e: "search"): void
  (e: "copy-path"): void
  (e: "go-parent"): void
  (e: "go-path", index: number): void
}>()
</script>

<template>
  <n-card class="mt-4">
    <div class="mb-4 flex items-center gap-2">
      <n-button
        size="small"
        :disabled="path === '/'"
        @click="emit('go-parent')"
      >
        返回上级
      </n-button>
      <template
        v-for="(segment, idx) in pathSegments"
        :key="idx"
      >
        <n-button
          text
          type="primary"
          style="padding: 0 4px"
          @click="emit('go-path', idx)"
        >
          {{ segment || "/" }}
        </n-button>
        <n-text
          v-if="idx !== pathSegments.length - 1"
          style="padding: 0 2px"
        >></n-text>
      </template>
    </div>

    <div class="mb-4 flex items-center gap-4">
      <n-input
        :value="path"
        placeholder="路径"
        style="width: 40%"
        @update:value="emit('update:path', $event)"
        @keyup.enter="emit('load')"
      >
        <template #suffix>
          <n-button
            quaternary
            circle
            size="tiny"
            @click="emit('copy-path')"
          >
            <template #icon>
              <n-icon>
                <Icon :name="copiedPath ? 'mdi:check' : 'mdi:content-copy'" />
              </n-icon>
            </template>
          </n-button>
        </template>
      </n-input>
      <n-input
        :value="search"
        placeholder="搜索文件名"
        style="width: 25%"
        @update:value="emit('update:search', $event)"
        @keyup.enter="emit('load')"
      />
      <a-space>
        <n-switch
          :value="showHidden"
          @update:value="emit('update:show-hidden', $event)"
        />
        <n-text class="ml-1">显示隐藏文件</n-text>
      </a-space>
      <a-space>
        <n-switch
          :value="expand"
          @update:value="emit('update:expand', $event)"
        />
        <n-text class="ml-1">展开目录</n-text>
      </a-space>
      <n-button
        type="primary"
        @click="emit('search')"
      >{{ $t("commons.button.search") }}</n-button>
    </div>

    <slot></slot>
  </n-card>
</template>
