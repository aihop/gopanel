<template>
	<n-modal
		v-model:show="open"
		preset="card"
		:title="$t('app.delete')"
		:closable="true"
		:mask-closable="false"
		:style="{ width: '420px' }"
	>
		<n-space vertical>
			<!-- 警告提示 -->
			<n-alert :type="recycleStatus === 'enable' ? 'warning' : 'info'" :show-icon="true">
				<span v-if="recycleStatus === 'enable'">
					{{ $t("file.deleteHelper") }}
				</span>
				<span v-else>
					{{ $t("file.deleteHelper2") }}
				</span>
			</n-alert>

			<!-- 强制删除 -->
			<n-checkbox v-if="recycleStatus === 'enable'" v-model:checked="forceDelete" class="force-delete">
				{{ $t("file.forceDeleteHelper") }}
			</n-checkbox>

			<!-- 文件列表 -->
			<n-scrollbar style="max-height: 280px">
				<n-space vertical size="small">
					<n-space v-for="(row, index) in files" :key="index" align="center" :size="4">
						<Icon :name="row.isDir ? DirIcon : FileIcon" />
						<span class="sle">{{ row.name }}</span>
					</n-space>
				</n-space>
			</n-scrollbar>
		</n-space>

		<!-- 底部按钮 -->
		<template #action>
			<n-space justify="end">
				<n-button :disabled="loading" @click="open = false">
					{{ $t("commons.button.cancel") }}
				</n-button>
				<n-button type="primary" :disabled="loading" :loading="loading" @click="onConfirm">
					{{ $t("commons.button.confirm") }}
				</n-button>
			</n-space>
		</template>
	</n-modal>
</template>

<script lang="ts" setup>
import type { File } from "@/api/interface/file"
import { DeleteFile, FileRecycleStatusAPI } from "@/api/modules/file"
import { settingSystemBaseDirAPI } from "@/api/modules/setting"
import { MsgSuccess, MsgWarning } from "@/utils/message"
import { getIcon } from "@/utils/util"
import { useMessage } from "naive-ui"
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"

const DirIcon = "ion:folder-outline"
const FileIcon = "ion:document-text-outline"

const emit = defineEmits<{
	(e: "close"): void
}>()

const { t } = useI18n()

const open = ref(false)
const files = ref<File.File[]>([])
const loading = ref(false)
const forceDelete = ref(false)
const recycleStatus = ref("enable")

const message = useMessage()

function acceptParams(props: File.File[]) {
	getStatus()
	files.value = props
	open.value = true
	forceDelete.value = true
}

async function getStatus() {
	try {
		const { data } = await FileRecycleStatusAPI()
		recycleStatus.value = data
		if (data === "disable") forceDelete.value = true
	} catch {}
}

async function onConfirm() {
	const tasks: Promise<any>[] = []
	for (const s of files.value) {
		if (s.isDir) {
			if (s.path.includes(".console_clash")) {
				MsgWarning(t("file.clashDeleteAlert"))
				return
			}
			const { data: base } = await settingSystemBaseDirAPI()
			if (s.path === base) {
				MsgWarning(t("file.panelInstallDir"))
				return
			}
		}
		tasks.push(DeleteFile({ path: s.path, isDir: s.isDir, forceDelete: forceDelete.value }))
	}

	loading.value = true
	try {
		await Promise.all(tasks)
		MsgSuccess(t("commons.msg.deleteSuccess"))
		open.value = false
		emit("close")
	} finally {
		loading.value = false
	}
}

const getIconName = (ext: string) => getIcon(ext)

defineExpose({ acceptParams })
</script>

<style scoped>
.table-icon {
	width: 16px;
	height: 16px;
}

.sle {
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.force-delete {
	white-space: pre-line;
	line-height: 24px;
}
</style>
