<template>
	<n-modal
		v-model:show="dialogVisible"
		preset="dialog"
		:title="$t('file.existFileTitle')"
		:style="{ width: '600px' }"
		:mask-closable="false"
	>
		<!-- 警告 -->
		<n-alert type="warning" :show-icon="true" :closable="false" class="mb-4">
			<span class="whitespace-break-spaces">{{ $t("file.existFileHelper") }}</span>
		</n-alert>

		<!-- 表格 -->
		<n-data-table :columns="columns" :data="existFiles" :max-height="350" :single-line="false" size="small" />

		<!-- 底部按钮 -->
		<template #action>
			<n-space justify="end">
				<n-button @click="handleSkip">{{ $t("commons.button.skip") }}</n-button>
				<n-button type="primary" @click="handleOverwrite">
					{{ $t("commons.button.cover") }}
				</n-button>
			</n-space>
		</template>
	</n-modal>
</template>

<script lang="ts" setup>
import { ref, h } from "vue"
import { NModal, NAlert, NButton, NSpace, NDataTable } from "naive-ui"
import { computeSize } from "@/utils/util"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

/* -------------------- 类型 -------------------- */
interface DialogProps {
	name: string
	path: string
	size: number
	uploadSize: number
	modTime: string
}

/* -------------------- 状态 -------------------- */
const dialogVisible = ref(false)
const existFiles = ref<DialogProps[]>([])
let onConfirmCallback: ((action: "skip" | "overwrite", skipped?: string[]) => void) | null = null

/* -------------------- 表格列 -------------------- */
const columns = [
	{
		type: "index",
		title: () => h("span", {}, t("commons.table.serialNumber")),
		width: 55
	},
	{
		key: "path",
		title: () => h("span", {}, t("commons.table.name")),
		minWidth: 200
	},
	{
		key: "size",
		title: () => h("span", {}, t("file.existFileSize")),
		width: 230,
		render: (row: DialogProps) => `${computeSize(row.uploadSize)} → ${computeSize(row.size)}`
	}
] as any

/* -------------------- 方法 -------------------- */
const getFileSize = (size: number) => computeSize(size)

const handleSkip = () => {
	dialogVisible.value = false
	onConfirmCallback?.(
		"skip",
		existFiles.value.map(f => f.path)
	)
}

const handleOverwrite = () => {
	dialogVisible.value = false
	onConfirmCallback?.("overwrite")
}

const acceptParams = async ({
	paths,
	onConfirm
}: {
	paths: DialogProps[]
	onConfirm: (action: "skip" | "overwrite", skipped?: string[]) => void
}) => {
	existFiles.value = paths
	onConfirmCallback = onConfirm
	dialogVisible.value = true
}

defineExpose({ acceptParams })
</script>

<style scoped>
.whitespace-break-spaces {
	white-space: pre-wrap;
	word-break: break-word;
}
</style>
