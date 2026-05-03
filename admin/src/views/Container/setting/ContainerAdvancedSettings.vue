<template>
  <div class="p-4">
    <FtEditor
      v-model="dockerConfProxy"
      language="json"
      height="calc(100vh - 450px)"
      class="mt-[10px]"
    />
    <n-button
      :disabled="daemonLoading || dockerOnly"
      type="primary"
      class="mt-[5px]"
      @click="emit('save-file')"
    >
      {{ $t("commons.button.save") }}
    </n-button>
    <n-modal
      :show="showRestartConfirm"
      preset="dialog"
      title="保存配置"
      :loading="saveLoading"
      positive-text="确认"
      negative-text="取消"
      @update:show="emit('update:show-restart-confirm', $event)"
      @positive-click="emit('confirm-restart')"
      @negative-click="emit('update:show-restart-confirm', false)"
    >
      <div>保存后将会重启 Docker 服务，是否确认？</div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue"

const FtEditor = defineAsyncComponent(() => import("../../../components/FtEditor/index.vue"))

const props = defineProps<{
  dockerConf: string
  daemonLoading: boolean
  dockerOnly: boolean
  showRestartConfirm: boolean
  saveLoading: boolean
}>()

const emit = defineEmits<{
  (e: "update:docker-conf", value: string): void
  (e: "save-file"): void
  (e: "confirm-restart"): void
  (e: "update:show-restart-confirm", value: boolean): void
}>()

const dockerConfProxy = computed({
  get: () => props.dockerConf,
  set: (value: string) => emit("update:docker-conf", value)
})
</script>
