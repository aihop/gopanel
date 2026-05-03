<template>
  <n-form-item :label="$t('commons.table.port')">
    <n-radio-group
      v-model:value="rowData.publishAllPorts"
      name="publishAllPortsGroup"
      class="ml-4"
    >
      <n-radio
        :value="false"
        :label="$t('container.exposePort')"
      />
      <n-radio
        :value="true"
        :label="$t('container.exposeAll')"
      />
    </n-radio-group>
  </n-form-item>

  <n-form-item v-if="!rowData.publishAllPorts">
    <n-card
      class="w-full"
      :content-style="{ padding: 25 }"
      :header-style="{ padding: '10px' }"
    >
      <n-data-table
        v-if="rowData.exposedPorts.length !== 0"
        :columns="exposedPortsColumns"
        :data="rowData.exposedPorts"
        :bordered="false"
        :single-line="false"
      />
      <n-button
        class="mb-2 ml-3 mt-2"
        @click="emit('add-port')"
      >
        {{ $t("commons.button.add") }}
      </n-button>
    </n-card>
  </n-form-item>

  <n-form-item
    :label="$t('container.network')"
    path="network"
  >
    <n-select
      v-model:value="rowData.network"
      :options="networks"
      label-field="option"
      value-field="option"
    />
  </n-form-item>

  <n-form-item
    label="IPv4"
    path="ipv4"
  >
    <n-input
      v-model:value="rowData.ipv4"
      :placeholder="$t('container.inputIpv4')"
    />
  </n-form-item>
  <n-form-item
    label="IPv6"
    path="ipv6"
  >
    <n-input
      v-model:value="rowData.ipv6"
      :placeholder="$t('container.inputIpv6')"
    />
  </n-form-item>
</template>

<script setup lang="ts">
import type { SelectOption } from "naive-ui"
import type { Container } from "@/api/interface/container"

defineProps<{
  rowData: Container.ContainerHelper
  networks: SelectOption[]
  exposedPortsColumns: any[]
}>()

const emit = defineEmits<{
  (e: "add-port"): void
}>()
</script>
