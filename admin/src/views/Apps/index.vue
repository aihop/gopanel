<template>
  <div class="apps-page space-y-8">
    <div class="apps-hero">
      <div class="space-y-3">
        <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">App Explore</div>
        <div class="text-3xl font-semibold fg-base-100">{{ $t('app.containerApp') }}</div>
        <div class="max-w-2xl text-sm leading-7 text-slate-500">{{ $t('app.containerAppHelper') }}</div>
      </div>
      <div class="apps-toolbar">
        <div class="toolbar-tabs">
          <n-tabs
            :value="tabValue"
            type="line"
            animated
            @update:value="handleTabChange"
          >
            <n-tab name="all">{{ $t('app.all') }}</n-tab>
            <n-tab name="installed">{{ $t('app.installed') }}</n-tab>
            <n-tab
              name="upgrade"
              disabled
            >{{ $t('app.upgrade') }}</n-tab>
          </n-tabs>
        </div>
        <div class="toolbar-actions">
          <n-input
            :value="searchName"
            size="large"
            clearable
            class="search-input"
            @update:value="value => (searchName = value)"
            @keydown.enter="handleSearch"
          />
          <n-button
            type="primary"
            size="large"
            class="!rounded-[18px] px-6 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
            @click="handleSearch"
          >
            {{ $t('app.search') }}
          </n-button>
          <n-button
            type="default"
            size="large"
            class="!rounded-[18px] px-6"
            :loading="syncing"
            @click="handleSyncApps"
          >
            {{ $t('app.sync') }}
          </n-button>
        </div>
      </div>
    </div>
    <div class="apps-summary">
      <div class="apps-summary__title">{{ currentTabLabel }}</div>
      <div class="apps-summary__meta text-gray-500">{{$t('app.summary',[total])}}</div>
    </div>
  </div>

  <div class="mt-6">
    <AppsAll
      v-if="tabValue === 'all'"
      :search-name="searchName"
      :page="page"
      :limit="limit"
      :refresh-key="refreshKey"
      @update:total="handleTotalUpdate"
    />
    <AppsInstalled
      v-else-if="tabValue === 'installed'"
      :search-name="searchName"
      :page="page"
      :limit="limit"
      :refresh-key="refreshKey"
      @update:total="handleTotalUpdate"
    />
  </div>

  <div class="mt-6 flex flex-col gap-4 border-t border-slate-100 pt-6 sm:flex-row sm:items-center sm:justify-between">
    <div class="text-sm text-slate-500">
      已展示
      <span class="mx-1 font-semibold text-slate-700">{{ currentTabLabel }}</span>
      的分页结果，共
      <span class="mx-1 font-semibold fg-base-100">{{ total }}</span>
      条记录
    </div>
    <n-pagination
      :page="page"
      :default-page-size="limit"
      :item-count="total"
      show-size-picker
      :page-sizes="[10, 20, 50]"
      class="apps-pagination"
      @update:page="value => (page = value)"
      @update:page-size="handlePageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import AppsAll from "./components/AppsAll.vue"
import AppsInstalled from "./components/AppsInstalled.vue"
import { appsSyncAPI } from "@/api/modules/apps"
import { useMessage } from "naive-ui"
import { t } from "@/i18n"

const message = useMessage()
const tabValue = ref("all")
const searchName = ref("")
const page = ref(1)
const limit = ref(10)
const total = ref(0)
const syncing = ref(false)
const refreshKey = ref(0)

const handleSyncApps = async () => {
	syncing.value = true
	try {
		const res = await appsSyncAPI()
		if (res.code == 0) {
			message.success(t('app.syncSuccess'))
			handleSearch()
		} else {
		}
	} catch (error) {
	} finally {
		syncing.value = false
	}
}

const currentTabLabel = computed(() => {
	if (tabValue.value === "all") return t('app.allApp')
	if (tabValue.value === "installed") return t('app.installed')
	return ""
})

const handleTabChange = (value: string) => {
	tabValue.value = value
	page.value = 1
}

const handleTotalUpdate = (val: number) => {
	total.value = val
}

const handleSearch = () => {
	page.value = 1
	refreshKey.value += 1
}

const handlePageSizeChange = (size: number) => {
	limit.value = size
	page.value = 1
}
</script>

<style scoped>
.apps-page {
	padding: 4px 0 0;
}

.apps-hero {
	display: flex;
	flex-wrap: wrap;
	gap: 24px;
	align-items: stretch;
	justify-content: space-between;
	padding: 28px;
	border-radius: 28px;
	border: 1px solid color-mix(in srgb, var(--border-color) 90%, transparent);
	background:
		radial-gradient(circle at top right, color-mix(in srgb, var(--primary-color) 12%, transparent), transparent 30%),
		linear-gradient(180deg, color-mix(in srgb, var(--bg-default-color) 98%, white), color-mix(in srgb, var(--bg-secondary-color) 92%, transparent));
	box-shadow: 0 18px 40px rgba(15, 23, 42, 0.06);
}

.apps-toolbar {
	display: flex;
	flex-direction: column;
	gap: 16px;
	flex: 1;
	min-width: 320px;
	max-width: 720px;
}

.toolbar-tabs {
	align-self: flex-end;
	padding: 8px 18px;
	border-radius: 20px;
	border: 1px solid color-mix(in srgb, var(--border-color) 92%, transparent);
	background: color-mix(in srgb, var(--bg-secondary-color) 85%, transparent);
}

.toolbar-actions {
	display: flex;
	gap: 12px;
	width: 100%;
}

.apps-summary {
	display: flex;
	align-items: center;
	justify-content: space-between;
	flex-wrap: wrap;
	gap: 10px 18px;
	padding: 0 4px;
}

.apps-summary__title {
	font-size: 1.05rem;
	font-weight: 700;
	color: var(--n-text-color);
}

.apps-summary__meta {
	font-size: 0.92rem;
}

 
.search-input :deep(.n-input) {
	--n-height: 48px;
	--n-border-radius: 18px;
	--n-padding-left: 18px;
	--n-padding-right: 18px;
}

.toolbar-tabs :deep(.n-tabs-nav) {
	margin-bottom: 0;
}

.apps-pagination :deep(.n-pagination-item),
.apps-pagination :deep(.n-pagination-item--button) {
	border-radius: 14px;
}

@media (max-width: 1280px) {
	.toolbar-tabs {
		align-self: flex-start;
	}
}

@media (max-width: 640px) {
	.apps-hero {
		padding: 22px 18px;
		border-radius: 24px;
	}

	.toolbar-actions {
		flex-direction: column;
	}

 

	.apps-pagination {
		justify-content: flex-start;
	}
}
</style>
