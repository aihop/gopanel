<template>
  <div>
    <n-drawer
      :show="backupVisible"
      @update:show="v => (backupVisible = v)"
      width="50%"
      :mask-closable="false"
    >
      <n-drawer-content>
        <template #header>
          <div class="flex items-center justify-between">
            <DrawerHeader
              v-if="detailName"
              :header="t('commons.button.backup')"
              :resource="name + ' (' + detailName + ')'"
              :back="handleClose"
            />
            <DrawerHeader
              v-else
              :header="t('commons.button.backup')"
              :resource="name"
              :back="handleClose"
            />
            <!--加一个关闭-->
            <n-icon
              class="ml-2 cursor-pointer"
              :component="renderIcon('mdi:close')"
              @click="handleClose"
            />
          </div>
        </template>

        <div style="padding: 16px">
          <n-alert
            v-if="type === 'app'"
            type="warning"
            :title="t('setting.backupJump')"
            show-icon
          >
            <template #default>
              <div class="mt-2 text-xs">
                <span>{{ t("setting.backupJump") }}</span>
                <span
                  class="jump"
                  @click="goFile"
                >
                  <n-icon
                    class="ml-2"
                    :component="renderIcon('mdi:map-marker-outline')"
                  />
                  {{ t("firewall.quickJump") }}
                </span>
              </div>
            </template>
          </n-alert>

          <div style="margin: 12px 0; display: flex; gap: 12px; align-items: center">
            <n-button
              type="primary"
              :disabled="status && status !== 'Running'"
              @click="onBackup"
            >
              {{ t("commons.button.backup") }}
            </n-button>
            <n-button
              type="default"
              :disabled="selects.length === 0"
              @click="onBatchDelete(null)"
            >
              {{ t("commons.button.delete") }}
            </n-button>
          </div>

          <n-data-table
            :columns="columns"
            :data="list"
            :row-key="rowKey"
            :loading="loading"
            :pagination="paginationOptions"
            remote
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
            @update:checked-row-keys="handleCheckAll"
            scroll-x="1000"
          />
        </div>

        <template #footer>
          <div style="padding: 12px 16px; display: flex; justify-content: flex-end"></div>
        </template>
      </n-drawer-content>
    </n-drawer>

    <n-modal
      :show="open"
      @update:show="v => (open = v)"
      :title="modalTitle"
      :mask-closable="false"
      preset="card"
    >
      <n-form
        ref="backupForm"
        :model="{ secret }"
        label-placement="left"
      >
        <n-form-item
          v-if="type === 'app' || type === 'website'"
          :label="t('setting.compressPassword')"
        >
          <n-input
            :value="secret"
            @update:value="v => (secret = v)"
            :placeholder="t('setting.backupRecoverMessage')"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px 0">
          <n-button
            @click="handleBackupClose"
            :disabled="loading"
          >{{ t("commons.button.cancel") }}</n-button>
          <n-button
            type="primary"
            @click="onSubmit"
            :disabled="loading"
          >
            {{ t("commons.button.confirm") }}
          </n-button>
        </div>
      </template>
    </n-modal>

    <OpDialog
      ref="opRef"
      @search="refreshBackupRecords"
    />
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, h, computed, onMounted, onUnmounted } from "vue"
import {
	NDrawer,
	NAlert,
	NButton,
	NDataTable,
	NModal,
	NForm,
	NFormItem,
	NInput,
	useDialog
} from "naive-ui"

import DrawerHeader from "./DrawerHeader.vue"
import OpDialog from "./OpDialog.vue"
import { downloadAuthenticatedFile } from "../utils/fileDownload"
import { createBackupColumns, createBackupPagination } from "./backupColumns"
import { renderIcon } from "../utils"
import { NIcon } from "naive-ui"
import {
	backupRecordListAPI,
	backupHandleAPI,
	backupRecordSizeAPI,
	backupRecordDeletesAPI,
	backupRecoverAPI,
	backupRecordDownloadAPI,
	backupListAPI
} from "../api/modules/backup"
import { useRouter } from "vue-router"
import { Backup } from "../api/interface/backup"
import { MsgError, MsgSuccess } from "../utils/message"
import emitter from "../utils/emitter"
import { useTable } from "../composables/useTable"
import { useAuthStore } from "../store/auth"

import { useI18n } from "vue-i18n"
const { t } = useI18n()
const dialog = useDialog()
const router = useRouter()

const selects = ref<any[]>([])
const opRef = ref<any>(null)

const backupVisible = ref(false)
const type = ref<string>("")
const name = ref<string>("")
const detailName = ref<string>("")
const detailId = ref<number>(0)
const backupPath = ref<string>("")
const status = ref<string>("")
const secret = ref<string>("")

const open = ref(false)
const isBackup = ref(true)
const currentRow = ref<any>(null)

const rowKey = (row: any) => row.id

const modalTitle = computed(() =>
	isBackup.value ? t("commons.button.backup") : `${t("commons.button.recover")} - ${name.value}`
)

const paginationOptions = computed(() => createBackupPagination(curPage.value, pageSize.value, total.value))

const loadBackupDir = async () => {
	try {
		const res = await backupListAPI({})
		let backupList = (res.data as any[]) || []
		for (const bac of backupList) {
			if (bac.type !== "LOCAL") continue
			if (bac.id !== 0) {
				bac.varsJson = JSON.parse(bac.vars)
			}
			backupPath.value = bac.varsJson?.dir || backupPath.value
			break
		}
	} catch (e) {
		console.error(e)
	}
}

const goFile = async () => {
	router.push({ name: "File", query: { path: `${backupPath.value}/app/${name.value}/${detailName.value}` } })
}

const loadSize = async () => {
	try {
		// 复用 useTable 的查询参数构造，保证 size 查询和列表查询命中同一批记录
		const res = await backupRecordSizeAPI({
			page: curPage.value,
			limit: pageSize.value,
			...(getParams() || {})
		})
		const stats = (res.data as any[]) || []
		const statsMap = new Map(stats.map((item: any) => [item.id, item.size]))
		for (const backup of list.value) {
			backup.hasLoad = true
			// 后端没返回的记录保持 size 为 undefined（显示 "-" 且不禁用恢复/下载），
			// 不能写 0：size===0 会被当作空备份禁用恢复和下载按钮
			if (statsMap.has(backup.id)) {
				backup.size = statsMap.get(backup.id)
			}
		}
	} catch {
		for (const backup of list.value) {
			backup.hasLoad = true
		}
	}
}

const onBackup = async () => {
	isBackup.value = true
	if (type.value !== "app" && type.value !== "website") {
		await dialog.info({
			title: t("commons.button.backup"),
			content: () =>
				h("div", null, t("commons.msg.backupHelper", [name.value + " ( " + detailName.value + " )"])),
			positiveText: t("commons.button.confirm"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: () => {
				onSubmit()
			}
		})
		return
	}
	open.value = true
}

const onSubmit = async () => {
	if (isBackup.value) {
		const params = {
			type: type.value,
			name: name.value,
			detailName: detailName.value,
			detailId: detailId.value,
			secret: secret.value
		}
		loading.value = true
		try {
			const res = await backupHandleAPI(params)
			loading.value = false
			handleBackupClose()
			if (res.code !== 0) {
				MsgError(res.msg || "备份提交失败")
				return
			}
			const key = (res.data as any)?.key
			if (!key) {
				MsgError("备份提交失败")
				return
			}
			const apiUrl = (window as any).__VITE_API_URL__ || "/api"
			const authStore = useAuthStore()
			const safeToken = encodeURIComponent(authStore.auth || "")
			opRef.value.acceptParams({
				title: t("commons.button.backup"),
				msg: "备份已开始，正在输出实时日志...",
				names: [],
				sseUrl: `${apiUrl}/backup/logs?key=${encodeURIComponent(key)}&token=${safeToken}`
			})
		} catch {
			loading.value = false
		}
		return
	}

	const params = {
		source: currentRow.value.source,
		type: type.value,
		name: name.value,
		detailName: detailName.value,
		file: `${currentRow.value.fileDir}/${currentRow.value.fileName}`,
		secret: secret.value,
		detailId: detailId.value
	}
	loading.value = true
	try {
		const res = await backupRecoverAPI(params)
		loading.value = false
		handleClose()
		handleBackupClose()
		if (res.code !== 0) {
			MsgError(res.msg || "恢复提交失败")
			return
		}
		const key = (res.data as any)?.key
		if (!key) {
			MsgError("恢复提交失败")
			return
		}
		const apiUrl = (window as any).__VITE_API_URL__ || "/api"
		const authStore = useAuthStore()
		const safeToken = encodeURIComponent(authStore.auth || "")
		opRef.value.acceptParams({
			title: t("commons.button.recover"),
			msg: "恢复已开始，正在输出实时日志...",
			names: [],
			sseUrl: `${apiUrl}/backup/logs?key=${encodeURIComponent(key)}&token=${safeToken}`
		})
	} catch {
		loading.value = false
	}
}

const params = reactive({
	listAPI: backupRecordListAPI,
	params: {
		wheres: [
			{
				field: "detailName",
				rule: "eq",
				val: ""
			},
			{
				field: "type",
				rule: "eq",
				val: ""
			}
		]
	}
})

const { list, curPage, pageSize, loading, getData, getParams, total } = useTable(params)

const refreshBackupRecords = async () => {
	await getData()
	if (!list.value.length) return
	await loadSize()
}

const handlePageChange = async (page: number) => {
	curPage.value = page
	await refreshBackupRecords()
}

const handlePageSizeChange = async (size: number) => {
	pageSize.value = size
	curPage.value = 1
	await refreshBackupRecords()
}

const onRecover = async (row: Backup.RecordInfo) => {
	isBackup.value = false
	currentRow.value = row
	if (type.value !== "app" && type.value !== "website") {
		await dialog.warning({
			title: t("commons.button.recover"),
			content: () =>
				h("div", null, t("commons.msg.recoverHelper", [name.value + " ( " + detailName.value + " )"])),
			positiveText: t("commons.button.confirm"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: () => {
				onSubmit()
			}
		})
		return
	}
	open.value = true
}

const onDownload = async (row: Backup.RecordInfo) => {
	const params = {
		source: row.source,
		fileDir: row.fileDir,
		fileName: row.fileName
	}
	const res = await backupRecordDownloadAPI(params)
	await downloadAuthenticatedFile(res.data as any)
}

const onBatchDelete = async (row: Backup.RecordInfo | null) => {
	const ids: number[] = []
	const names: string[] = []

	console.log(row, selects.value)

	if (row) {
		ids.push(row.id)
		names.push(row.fileName)
	} else {
		selects.value.forEach((item: Backup.RecordInfo) => {
			ids.push(item.id)
			names.push(item.fileName)
		})
	}
	opRef.value.acceptParams({
		names,
		title: t("commons.button.delete"),
		api: backupRecordDeletesAPI,
		msg: t("commons.msg.operatorHelper", [t("commons.button.backup"), t("commons.button.delete")]),
		params: { ids }
	})
}

const handleCheckAll = (keys: Array<number | string>) => {
	if (!Array.isArray(keys) || keys.length === 0) {
		selects.value = []
		return
	}
	const keySet = new Set(keys.map(k => String(k)))
	selects.value = list.value.filter((item: any) => keySet.has(String(item.id)))
}

const handleClose = () => {
	backupVisible.value = false
}
const handleBackupClose = () => {
	open.value = false
}

const columns = createBackupColumns(t, onBatchDelete, onRecover, onDownload)

const acceptParams = (param: { type: string; name: string; detailName: string; status: string; detailId: number }) => {
	type.value = param.type
	if (type.value === "app") loadBackupDir()
	name.value = param.name
	detailName.value = param.detailName
	backupVisible.value = true
	status.value = "Running"
	detailId.value = param.detailId || 0
	selects.value = []
	curPage.value = 1
	params.params.wheres[0].val = detailName.value
	params.params.wheres[1].val = param.type
	void refreshBackupRecords()
}

defineExpose({ acceptParams })

onMounted(() => {
	emitter.on("database:refresh", refreshBackupRecords)
})

onUnmounted(() => {
	emitter.off("database:refresh", refreshBackupRecords)
})
</script>

<style lang="scss" scoped>
.jump {
	color: var(--primary-color);
	cursor: pointer;
	&:hover {
		color: #74a4f3;
	}
}
</style>
