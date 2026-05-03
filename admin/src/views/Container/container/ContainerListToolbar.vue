<template>
  <div>
    <div class="flex flex-col justify-end gap-4 sm:flex-row">
      <div class="w-[200px]">
        <n-select
          :value="searchState"
          clearable
          :options="stateOptions"
          @update:value="emit('update:search-state', $event)"
        >
          <template #header>{{ $t("commons.table.status") }}</template>
        </n-select>
      </div>
      <TableColumnSelect
        :columns="columnSettings"
        storage-key="containerColumn"
        size="medium"
        button-label="列设置"
        @update:columns="emit('update:columns', $event)"
      />
    </div>

    <div class="flex w-full flex-col gap-4 py-3 md:flex-row md:justify-between">
      <div class="flex flex-wrap gap-4">
        <n-button
          type="primary"
          @click="emit('create')"
        >
          {{ $t("container.create") }}
        </n-button>
        <n-button
          type="primary"
          ghost
          @click="emit('prune')"
        >
          {{ $t("container.containerPrune") }}
        </n-button>

        <n-button-group>
          <n-button
            v-for="item in bulkActions"
            :key="item.key"
            :disabled="item.disabled"
            @click="emit('bulk-operate', item.key)"
          >
            {{ item.label }}
          </n-button>
        </n-button-group>
      </div>

      <div class="flex flex-row gap-2 md:flex-col lg:flex-row">
        <TableSetting
          title="container-refresh"
          @search="emit('refresh')"
        />
        <TableSearch
          :search-name="searchName"
          @update:search-name="emit('update:search-name', $event)"
          @search="emit('search')"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import TableColumnSelect from "@/components/TableColumnSelect.vue"
import TableSearch from "@/components/TableSearch.vue"
import TableSetting from "@/components/TableSetting.vue"

interface ColumnSetting {
  key: string
  title: string
  visible: boolean
  fixed?: boolean
  original?: any
}

interface BulkAction {
  key: string
  label: string
  disabled: boolean
}

defineProps<{
  searchState: string
  searchName?: string
  stateOptions: Array<{ label: string; value: string }>
  columnSettings: ColumnSetting[]
  bulkActions: BulkAction[]
}>()

const emit = defineEmits<{
  (e: "update:search-state", value: string): void
  (e: "update:search-name", value: string): void
  (e: "update:columns", value: ColumnSetting[]): void
  (e: "create"): void
  (e: "prune"): void
  (e: "refresh"): void
  (e: "search"): void
  (e: "bulk-operate", value: string): void
}>()
</script>
