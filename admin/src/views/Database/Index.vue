<script setup lang="ts">
import CreateDatabaseModal from "./src/CreateDatabaseModal.vue"
import CreateServerModal from "./src/CreateServerModal.vue"
import ImportSqliteModal from "./src/ImportSqliteModal.vue"
import CreateUserModal from "./src/CreateUserModal.vue"
import DatabaseList from "./src/DatabaseList.vue"
import ServerList from "./src/ServerList.vue"
import UserList from "./src/UserList.vue"
import DatabaseManager from "./src/DatabaseManager.vue"
import { NButton, NSelect } from "naive-ui"
import { computed, ref, onMounted } from "vue"
import CommonPage from "@/components/page/Common.vue"
import { databaseServerListAPI } from "@/api/modules/database"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"

const { t } = useI18n()
const message = useMessage()
const currentTab = ref("manager")

const createDatabaseModalShow = ref(false)
const createUserModalShow = ref(false)
const createServerModalShow = ref(false)
const importSqliteModalShow = ref(false)

const globalSelectedServerId = ref<number | null>(null)
const serverOptions = ref<{label: string, value: number}[]>([])

const fetchServers = async () => {
	try {
		const res = await databaseServerListAPI({ page: 1, limit: 100 })
		const data = res.data as any
		if (data) {
			const items = Array.isArray(data) ? data : (data.items || [])
			serverOptions.value = [
				{ label: t("commons.button.all", "全部服务器"), value: null as any },
				...items.map((s: any) => ({
					label: `${s.name} (${s.type})`,
					value: s.id
				}))
			]
		}
	} catch (error) {
		console.error(error)
	}
}

import { provide } from "vue"

onMounted(() => {
	fetchServers()
})

provide("globalSelectedServerId", globalSelectedServerId)

const summaryCards = computed(() => [
	{
		label: "View",
		value: currentTab.value === "manager" ? "工作台" : currentTab.value === "database" ? "数据库" : currentTab.value === "user" ? "用户" : "服务器",
		desc: "当前正在管理的数据库资源视图"
	},
	{
		label: "Action",
		value: currentTab.value === "manager" ? "执行 SQL" : currentTab.value === "database" ? "创建库" : currentTab.value === "user" ? "创建用户" : "添加服务器",
		desc: "主操作会随当前标签页自动切换"
	},
	{
		label: "Focus",
		value: "统一管理",
		desc: "把数据库、用户和服务器放在一个清晰入口"
	}
])

const currentActionLabel = computed(() => {
	if (currentTab.value === "manager") return t('database.dataPanel')
	if (currentTab.value === "database") return t('database.createDatabase')
	if (currentTab.value === "user") return t('database.createUser')
	return t('database.addServer')
})

const openCreateModal = () => {
	if (currentTab.value === "manager") return
	if (currentTab.value === "database") {
		createDatabaseModalShow.value = true
		return
	}
	if (currentTab.value === "user") {
		createUserModalShow.value = true
		return
	}
	createServerModalShow.value = true
}

const openImportSqliteModal = () => {
	importSqliteModalShow.value = true
}

onMounted(() => {
	fetchServers()
})

provide("globalSelectedServerId", globalSelectedServerId)
</script>
<template>
  <div class="mt-4">
    <common-page
      show-header
      show-footer
    >
      <template #header>
        <div class="space-y-8 px-4">
          <div class="flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
            <div class="space-y-3">
              <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
                Database Explore
              </div>
              <div class="text-2xl font-semibold fg-base-100">{{ $t('database.workspace') }}</div>
              <div class="text-sm leading-7 text-slate-500">
                {{ $t('database.workspaceDesc') }}
              </div>
            </div>
            <div class="flex flex-col gap-4 xl:min-w-[520px] xl:items-end">
              <div class="rounded-lg border border-slate-200 bg-base-100 p-4 px-6">
                <n-tabs
                  :value="currentTab"
                  animated
                  @update:value="value => (currentTab = value)"
                >
                  <n-tab
                    name="manager"
                    :tab="$t('database.workspaceTitle')"
                  />
                  <n-tab
                    name="database"
                    :tab="$t('database.database')"
                  />
                  <n-tab
                    name="user"
                    :tab="$t('database.user')"
                  />
                  <n-tab
                    name="server"
                    :tab="$t('database.server')"
                  />
                </n-tabs>
              </div>
              <div class="flex w-full items-center justify-end gap-4  px-5 ">
                <div class="flex items-center gap-4 relative z-10">
                  <n-select
                    v-if="currentTab === 'database' || currentTab === 'user'"
                    :value="globalSelectedServerId"
                    :options="serverOptions"
                    class="w-48"
                    size="large"
                    placeholder="请选择服务器过滤"
                    :consistent-menu-width="false"
                    @update:value="(v) => globalSelectedServerId = v"
                  />
                  <n-button
                    v-if="currentTab === 'database'"
                    type="default"
                    size="large"
                    class="!rounded-[18px] px-6"
                    @click="openImportSqliteModal"
                  >
                    导入 SQLite 数据库
                  </n-button>
                  <n-button
                    v-if="currentTab !== 'manager'"
                    type="primary"
                    size="large"
                    class="!rounded-[18px] px-6 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
                    @click="openCreateModal"
                  >
                    {{ currentActionLabel }}
                  </n-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
      <template #tabbar></template>

      <database-manager v-if="currentTab === 'manager'" />
      <database-list v-if="currentTab === 'database'" />
      <user-list v-if="currentTab === 'user'" />
      <server-list v-if="currentTab === 'server'" />

    </common-page>
    <create-database-modal
      :show="createDatabaseModalShow"
      @update:show="value => (createDatabaseModalShow = value)"
    />
    <create-user-modal
      :show="createUserModalShow"
      @update:show="value => (createUserModalShow = value)"
    />
    <create-server-modal
      :show="createServerModalShow"
      @update:show="value => (createServerModalShow = value)"
    />
    <import-sqlite-modal
      :show="importSqliteModalShow"
      @update:show="value => (importSqliteModalShow = value)"
    />
  </div>
</template>
