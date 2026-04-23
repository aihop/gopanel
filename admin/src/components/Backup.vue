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
            @update:page="onPageChange"
            @update:pageSize="onPageSizeChange"
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
      @search="getData"
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
	NPopconfirm,
	useDialog
} from "naive-ui"

import DrawerHeader from "./DrawerHeader.vue"
import OpDialog from "./OpDialog.vue"
import { computeSize, dateFormat, downloadFile } from "../utils/util"
import { formatTime } from "../utils/date"
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

const data = ref<any[]>([])
const paginationConfig = reactive({
	currentPage: 1,
	limit: 10,
	total: 0
})

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

const paginationOptions = computed(() => ({
	page: paginationConfig.currentPage,
	limit: paginationConfig.limit,
	pageCount: Math.max(1, Math.ceil((paginationConfig.total || 0) / paginationConfig.limit)),
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100],
	showQuickJumper: true,
	itemCount: paginationConfig.total
}))

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

const loadSize = async (params: any) => {
	try {
		const res = await backupRecordSizeAPI(params)
		const stats = (res.data as any[]) || []
		if (!stats.length) return
		for (const backup of list.value) {
			for (const item of stats) {
				if (backup.id === item.id) {
					backup.hasLoad = true
					backup.size = item.size
					break
				}
			}
		}
	} catch {
		loading.value = false
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
				rule: "=",
				val: ""
			},
			{
				field: "type",
				rule: "=",
				val: ""
			}
		]
	}
})

const {
	list,
	pages,
	curPage,
	pageSize,
	getList,
	loading,
	getData,
	onPageSizeChange,
	onPageChange,
	pageSizeOptions,
	total
} = useTable(params)

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
	downloadFile(res.data as any)
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

const buttons = [
	{
		label: t("commons.button.delete"),
		click: (row: Backup.RecordInfo) => onBatchDelete(row)
	},
	{
		label: t("commons.button.recover"),
		disabled: (row: any) => row.size === 0,
		click: (row: Backup.RecordInfo) => onRecover(row)
	},
	{
		label: t("commons.button.download"),
		disabled: (row: any) => row.size === 0,
		click: (row: Backup.RecordInfo) => onDownload(row)
	}
]

const columns: any = [
	{
		type: "selection",
		width: 48
	},
	{
		title: t("commons.table.name"),
		key: "fileName",
		ellipsis: true
	},
	{
		title: t("file.size"),
		key: "size",
		width: 100,
		render(row: any) {
			if (row.hasLoad) {
				return row.size ? computeSize(row.size) : "-"
			}
			return h(NButton, { quaternary: true, size: "tiny", loading: true })
		}
	},
	{
		title: t("database.source"),
		key: "backupType",
		width: 150,
		render(row: any) {
			return row.source ? t("setting." + row.source) : ""
		}
	},
	{
		title: t("commons.table.date"),
		key: "createdAt",
		width: 180,
		render(row: any) {
			return formatTime(row.createdAt)
		}
	},
	{
		title: t("commons.table.operate"),
		key: "actions",
		width: 240,
		render(row: any) {
			return h("div", { style: "display:flex; gap:8px;" }, [
				h(
					NButton,
					{ size: "small", onClick: () => onBatchDelete(row) },
					{ default: () => t("commons.button.delete") }
				),
				h(
					NButton,
					{ size: "small", onClick: () => onRecover(row) },
					{ default: () => t("commons.button.recover") }
				),
				h(
					NButton,
					{ size: "small", onClick: () => onDownload(row) },
					{ default: () => t("commons.button.download") }
				)
			])
		}
	}
]

const acceptParams = (param: { type: string; name: string; detailName: string; status: string; detailId: number }) => {
	type.value = param.type
	if (type.value === "app") loadBackupDir()
	name.value = param.name
	detailName.value = param.detailName
	backupVisible.value = true
	status.value = "Running"
	detailId.value = param.detailId || 0
	params.params.wheres[0].val = detailName.value
	params.params.wheres[1].val = param.type
	getData()
	setTimeout(() => {
		if (total.value === 0) return
		loadSize(params)
	}, 300)
}

defineExpose({ acceptParams })

onMounted(() => {
	emitter.on("database:refresh", getData)
})

onUnmounted(() => {
	emitter.off("database:refresh", getData)
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
