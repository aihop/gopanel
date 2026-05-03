<template>
  <n-form-item
    label="Command"
    path="cmdStr"
  >
    <n-input
      v-model:value="rowData.cmdStr"
      type="textarea"
      :placeholder="$t('container.cmdHelper')"
    />
  </n-form-item>
  <n-form-item
    label="Entrypoint"
    path="entrypointStr"
  >
    <n-input
      v-model:value="rowData.entrypointStr"
      :placeholder="$t('container.entrypointHelper')"
    />
  </n-form-item>
  <n-form-item path="autoRemove">
    <n-checkbox v-model:checked="rowData.autoRemove">
      {{ $t("container.autoRemove") }}
    </n-checkbox>
  </n-form-item>
  <n-form-item>
    <div class="flex w-full flex-col gap-1">
      <n-checkbox v-model:checked="rowData.privileged">
        {{ $t("container.privileged") }}
      </n-checkbox>
      <span class="input-help">{{ $t("container.privilegedHelper") }}</span>
    </div>
  </n-form-item>
  <n-form-item :label="$t('container.console')">
    <n-checkbox
      v-model:checked="rowData.tty"
      :label="$t('container.tty')"
    />
    <n-checkbox
      v-model:checked="rowData.openStdin"
      :label="$t('container.openStdin')"
    />
  </n-form-item>
  <n-form-item
    :label="$t('container.restartPolicy')"
    path="restartPolicy"
  >
    <n-radio-group
      v-model:value="rowData.restartPolicy"
      name="restartPolicyGroup"
    >
      <n-radio value="no" :label="$t('container.no')" />
      <n-radio value="always" :label="$t('container.always')" />
      <n-radio value="on-failure" :label="$t('container.onFailure')" />
      <n-radio value="unless-stopped" :label="$t('container.unlessStopped')" />
    </n-radio-group>
  </n-form-item>
  <n-form-item
    :label="$t('container.cpuShare')"
    path="cpuShares"
  >
    <n-input-number
      v-model:value="rowData.cpuShares"
      class="w-full max-w-[240px]"
      :min="0"
    />
    <span class="input-help">{{ $t("container.cpuShareHelper") }}</span>
  </n-form-item>

  <n-form-item
    :label="$t('container.cpuQuota')"
    path="nanoCPUs"
    :rule="checkFloatNumberRange(0, Number(limits.cpu))"
  >
    <n-input-group class="w-full max-w-[240px]">
      <n-input-number
        v-model:value="rowData.nanoCPUs"
        :step="0.1"
        :min="0"
        :max="Number(limits.cpu)"
      />
      <n-input-group-label class="min-w-[50px] text-center">
        {{ $t("commons.units.core") }}
      </n-input-group-label>
    </n-input-group>

    <span class="input-help">
      {{ $t("container.limitHelper", [limits.cpu]) }}{{ $t("commons.units.core") }}
    </span>
  </n-form-item>
  <n-form-item
    :label="$t('container.memoryLimit')"
    path="memory"
    :rule="checkFloatNumberRange(0, Number(limits.memory))"
  >
    <n-input-group class="w-full max-w-[240px]">
      <n-input-number
        v-model:value="rowData.memory"
        :step="1"
        :min="0"
        :max="Number(limits.memory)"
      />
      <n-input-group-label class="min-w-[35px] text-center">
        MB
      </n-input-group-label>
    </n-input-group>
    <span class="input-help">{{ $t("container.limitHelper", [limits.memory]) }}MB</span>
  </n-form-item>
  <n-form-item
    :label="$t('container.tag')"
    path="labelsStr"
  >
    <n-input
      v-model:value="rowData.labelsStr"
      type="textarea"
      :placeholder="$t('container.tagHelper')"
      :rows="3"
    />
  </n-form-item>
  <n-form-item
    :label="$t('container.env')"
    path="envStr"
  >
    <n-input
      v-model:value="rowData.envStr"
      type="textarea"
      :placeholder="$t('container.tagHelper')"
      :rows="3"
    />
  </n-form-item>
</template>

<script setup lang="ts">
import { checkFloatNumberRange } from "@/global/form-rules"
import type { Container } from "@/api/interface/container"

defineProps<{
  rowData: Container.ContainerHelper
  limits: Container.ResourceLimit
}>()
</script>

<style scoped lang="scss">
.input-help {
  color: #adb0bc;
}
</style>
