<script setup lang="ts">
defineProps<{
  ruleType: string
  keyword: string
  strategy: string
  status: string
  refreshSeconds: number
  strategyOptions: Array<{ label: string; value: string }>
  statusOptions: Array<{ label: string; value: string }>
  refreshOptions: Array<{ label: string; value: number }>
  createButtonText: string
  ruleTypeLabel: string
  selectedCount: number
  selectedDisabled: boolean
  columns: any[]
  rules: any[]
  loading: boolean
  pagination: Record<string, any>
  selectedRowKeys: string[]
}>()

const emit = defineEmits<{
  (e: "update:rule-type", value: string): void
  (e: "update:keyword", value: string): void
  (e: "update:strategy", value: string): void
  (e: "update:status", value: string): void
  (e: "update:refresh-seconds", value: number): void
  (e: "search"): void
  (e: "create"): void
  (e: "batch-delete"): void
  (e: "page-change", value: number): void
  (e: "page-size-change", value: number): void
  (e: "checked-row-keys", value: Array<string | number>): void
}>()

const getRuleRowKey = (row: { id: string | number }) => row.id
</script>

<template>
  <div class="mt-8 rounded-[28px] border border-blue-100/80 bg-base-100 p-6 shadow-[0_18px_48px_rgba(15,23,42,0.08)] sm:p-8">
    <div class="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
      <div class="space-y-3">
        <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Rule Workspace</div>
        <div class="text-2xl font-semibold fg-base-100">规则工作台</div>
        <div class="text-sm leading-7 text-slate-500">
          支持端口规则、IP 规则和转发规则切换，按关键词与策略快速过滤，并对选中规则进行批量移除。
        </div>
      </div>
      <div class="flex flex-col gap-4 xl:min-w-[640px] xl:items-end">
        <div class="rounded-lg border border-slate-200 bg-slate-50/90 p-2 px-6">
          <n-tabs
            :value="ruleType"
            animated
            @update:value="emit('update:rule-type', $event)"
          >
            <n-tab name="port">端口规则</n-tab>
            <n-tab name="ip">IP 规则</n-tab>
            <n-tab name="forward">转发规则</n-tab>
          </n-tabs>
        </div>

        <div class="grid w-full gap-3 lg:grid-cols-[1.5fr_0.9fr_0.9fr_0.8fr_auto_auto]">
          <n-input
            :value="keyword"
            clearable
            placeholder="搜索端口、IP、协议或描述"
            class="filter-input"
            @update:value="emit('update:keyword', $event)"
            @keydown.enter="emit('search')"
          >
            <template #suffix>
              <Icon name="material-symbols:search" />
            </template>
          </n-input>
          <n-select
            v-if="ruleType !== 'forward'"
            :value="strategy"
            :options="strategyOptions"
            class="filter-select"
            @update:value="emit('update:strategy', $event)"
          />
          <div v-else></div>
          <n-select
            :value="status"
            :options="statusOptions"
            class="filter-select"
            @update:value="emit('update:status', $event)"
          />
          <n-select
            :value="refreshSeconds"
            :options="refreshOptions"
            class="filter-select"
            @update:value="emit('update:refresh-seconds', $event)"
          />
          <n-button
            class="!rounded-[18px] px-6"
            @click="emit('search')"
          >筛选</n-button>
          <n-button
            type="primary"
            class="!rounded-[18px] px-6 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
            @click="emit('create')"
          >
            {{ createButtonText }}
          </n-button>
        </div>

        <div class="flex w-full flex-wrap items-center justify-between gap-3 rounded-[22px] border border-slate-200 bg-slate-50/90 px-5 py-4">
          <div class="text-sm leading-7 text-slate-500">
            当前视图：
            <span class="font-medium text-slate-700">{{ ruleTypeLabel }}</span>
            <span class="mx-2 text-slate-300">·</span>
            已选择
            <span class="font-semibold fg-base-100">{{ selectedCount }}</span>
            条规则
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <n-button
              ghost
              type="error"
              class="!rounded-[18px] px-5"
              :disabled="selectedDisabled"
              @click="emit('batch-delete')"
            >
              批量删除
            </n-button>
            <n-tag
              round
              :bordered="false"
              type="warning"
            >Docker 映射端口可能不受 ufw 完全控制</n-tag>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-8 rounded-[26px] border border-slate-100 bg-slate-50/75 p-4 sm:p-6">
      <div class="mb-5 rounded-[20px] border border-amber-100 bg-amber-50/80 px-4 py-3 text-sm leading-7 text-amber-700">
        Linux 防火墙对 Docker
        端口映射存在天然限制，若端口来自容器编排，请优先在应用或已安装页面控制端口暴露策略。
      </div>
      <n-data-table
        :columns="columns"
        :data="rules"
        :loading="loading"
        :pagination="pagination"
        :remote="true"
        :row-key="getRuleRowKey"
        :checked-row-keys="selectedRowKeys"
        @update:page="emit('page-change', $event)"
        @update:page-size="emit('page-size-change', $event)"
        @update:checked-row-keys="emit('checked-row-keys', $event)"
      />
    </div>
  </div>
</template>
