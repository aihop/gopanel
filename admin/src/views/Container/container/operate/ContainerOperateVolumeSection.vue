<template>
  <n-form-item :label="$t('container.mount')">
    <div class="w-full">
      <div
        v-for="(row, index) in rowData.volumes"
        :key="index"
        class="mb-2 last:mb-0"
      >
        <n-card class="w-full">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <n-radio-group
              v-model:value="row.type"
              name="volumeTypeGroup"
            >
              <n-radio-button
                value="volume"
                :label="$t('container.volumeOption')"
              />
              <n-radio-button
                value="bind"
                :label="$t('container.hostOption')"
              />
            </n-radio-group>
            <n-button
              text
              type="primary"
              @click="emit('delete-volume', index)"
            >
              {{ $t("commons.button.delete") }}
            </n-button>
          </div>
          <n-grid
            class="mt-4"
            :x-gap="8"
            :y-gap="8"
            :cols="24"
          >
            <n-gi :span="10">
              <n-form-item
                v-if="row.type === 'volume'"
                :label="$t('container.volumeOption')"
                :path="`volumes[${index}].sourceDir`"
              >
                <n-select
                  v-model:value="row.sourceDir"
                  filterable
                  :options="volumes"
                  value-field="option"
                >
                  <template #arrow>
                    <div></div>
                  </template>
                  <template #empty>
                    <div v-if="!volumes || volumes.length === 0">
                      {{ $t("commons.noData") }}
                    </div>
                  </template>
                </n-select>
              </n-form-item>
              <n-form-item
                v-else
                :label="$t('container.hostOption')"
                :path="`volumes[${index}].sourceDir`"
              >
                <n-input v-model:value="row.sourceDir" />
              </n-form-item>
            </n-gi>
            <n-gi :span="5">
              <n-form-item
                :label="$t('container.mode')"
                :path="`volumes[${index}].mode`"
              >
                <n-select
                  v-model:value="row.mode"
                  class="w-full"
                  filterable
                  :options="modeOptions"
                />
              </n-form-item>
            </n-gi>
            <n-gi :span="9">
              <n-form-item
                :label="$t('container.containerDir')"
                :path="`volumes[${index}].containerDir`"
              >
                <n-input v-model:value="row.containerDir" />
              </n-form-item>
            </n-gi>
          </n-grid>
        </n-card>
      </div>
      <n-button
        class="mt-3"
        @click="emit('add-volume')"
      >
        {{ $t("commons.button.add") }}
      </n-button>
    </div>
  </n-form-item>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { SelectOption } from "naive-ui"
import type { Container } from "@/api/interface/container"

defineProps<{
  rowData: Container.ContainerHelper
  volumes: SelectOption[]
}>()

const emit = defineEmits<{
  (e: "add-volume"): void
  (e: "delete-volume", index: number): void
}>()

const { t } = useI18n()

const modeOptions = computed(() => [
  { value: "rw", label: t("container.modeRW") },
  { value: "ro", label: t("container.modeR") }
])
</script>
