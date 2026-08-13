<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Pipeline } from "@/api/interface/pipeline"
import { t } from "@/i18n"

const props = defineProps<{
  show: boolean
  pipeline: Pipeline.ResPipeline | null
  loading: boolean
}>()

const emit = defineEmits<{
  (event: "update:show", value: boolean): void
  (event: "confirm", confirmName: string): void
}>()

const confirmName = ref("")
const acknowledged = ref(false)
const canDelete = computed(() => acknowledged.value && confirmName.value === props.pipeline?.name)

watch(
  () => props.show,
  (show) => {
    if (show) {
      confirmName.value = ""
      acknowledged.value = false
    }
  }
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('pipeline.forceDeleteTitle')"
    style="width: 620px"
    @update:show="emit('update:show', $event)"
  >
    <n-alert type="error" :title="t('pipeline.forceDeleteDangerTitle')" class="mb-4">
      {{ t("pipeline.forceDeleteDangerDescription") }}
    </n-alert>

    <div class="mb-4 rounded-xl bg-red-50 p-4 text-sm text-red-900 dark:bg-red-950/30 dark:text-red-100">
      <div class="mb-2 font-semibold">{{ t("pipeline.forceDeleteScopeTitle") }}</div>
      <ul class="list-disc space-y-1 pl-5">
        <li>{{ t("pipeline.forceDeleteScopeRecords") }}</li>
        <li>{{ t("pipeline.forceDeleteScopeReleases") }}</li>
        <li>{{ t("pipeline.forceDeleteScopeRuntime") }}</li>
        <li>{{ t("pipeline.forceDeleteImagesRemain") }}</li>
      </ul>
    </div>

    <n-checkbox v-model:checked="acknowledged" class="mb-4">
      {{ t("pipeline.forceDeleteAcknowledge") }}
    </n-checkbox>

    <div class="mb-2 text-sm text-slate-600 dark:text-slate-300">
      {{ t("pipeline.forceDeleteConfirmHint", { name: pipeline?.name || "" }) }}
    </div>
    <n-input
      v-model:value="confirmName"
      :placeholder="t('pipeline.forceDeleteConfirmPlaceholder')"
      :disabled="loading"
    />

    <template #footer>
      <div class="flex justify-end gap-3">
        <n-button :disabled="loading" @click="emit('update:show', false)">
          {{ t("commons.button.cancel") }}
        </n-button>
        <n-button
          type="error"
          :loading="loading"
          :disabled="!canDelete"
          @click="emit('confirm', confirmName)"
        >
          {{ t("pipeline.forceDeleteAction") }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>
