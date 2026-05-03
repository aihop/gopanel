<script setup lang="ts">
const props = defineProps<{
  selectedCount: number
  moveOpen: boolean
  pasteCount: number
  createOptions: Array<{ label: string; key: string }>
}>()

const emit = defineEmits<{
  (e: "create", key: string): void
  (e: "upload"): void
  (e: "move", type: string): void
  (e: "compress"): void
  (e: "batch-role"): void
  (e: "delete"): void
  (e: "paste"): void
  (e: "cancel-move"): void
}>()

const hasSelection = () => props.selectedCount > 0
</script>

<template>
  <n-card>
    <div class="flex justify-between">
      <n-space>
        <n-dropdown
          placement="bottom-start"
          trigger="click"
          :options="createOptions"
          @select="(key) => emit('create', String(key))"
        >
          <n-button type="primary">
            {{ $t("commons.button.create") }}
          </n-button>
        </n-dropdown>
        <n-button
          type="default"
          @click="emit('upload')"
        >{{ $t("file.upload") }}</n-button>

        <n-button
          plain
          :disabled="!hasSelection()"
          @click="emit('move', 'copy')"
        >
          {{ $t("file.copy") }}
        </n-button>
        <n-button
          plain
          :disabled="!hasSelection()"
          @click="emit('move', 'cut')"
        >
          {{ $t("file.move") }}
        </n-button>
        <n-button
          plain
          :disabled="!hasSelection()"
          @click="emit('compress')"
        >
          {{ $t("file.compress") }}
        </n-button>
        <n-button
          plain
          :disabled="!hasSelection()"
          @click="emit('batch-role')"
        >
          {{ $t("file.editPermissions") }}
        </n-button>
        <n-button
          plain
          :disabled="!hasSelection()"
          @click="emit('delete')"
        >
          {{ $t("commons.button.delete") }}
        </n-button>
      </n-space>
      <div v-if="moveOpen">
        <n-space>
          <n-tooltip
            trigger="hover"
            placement="bottom"
          >
            <template #trigger>
              <n-button
                plain
                @click="emit('paste')"
              >{{ $t("file.paste") }} ({{ pasteCount }})</n-button>
            </template>
            {{ $t("file.paste") }}
          </n-tooltip>
          <n-tooltip
            trigger="hover"
            placement="bottom"
          >
            <template #trigger>
              <n-button
                plain
                class="close"
                @click="emit('cancel-move')"
              >×</n-button>
            </template>
            {{ $t("file.cancel") }}
          </n-tooltip>
        </n-space>
      </div>
    </div>
  </n-card>
</template>
