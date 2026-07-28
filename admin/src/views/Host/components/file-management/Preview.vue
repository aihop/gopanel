<template>
	<n-modal v-model:show="open" :style="{ width: isFullscreen ? '100%' : '75%' }">
		<template #header>
			<div class="flex w-full items-center justify-between">
				<span>{{ `${$t("commons.button.preview")} - ${filePath}` }}</span>
				<n-space align="center" class="dialog-header-icon">
					<n-tooltip v-if="fileType !== 'excel'" placement="top">
						<template #trigger>
							<n-button size="small" circle :aria-label="loadTooltip()" @click="toggleFullscreen">
								<Icon :size="20" name="fluent:full-screen-maximize-24-regular" />
							</n-button>
						</template>
						<template #default>{{ loadTooltip() }}</template>
					</n-tooltip>

					<n-button size="small" circle :aria-label="$t('commons.button.close')" @click="handleClose">
						<Icon :size="20" name="carbon:close" />
					</n-button>
				</n-space>
			</div>
		</template>

		<n-spin :show="loading">
			<div :style="contentStyle" class="preview-body">
				<div class="flex h-full w-full items-center justify-center">
					<n-image
						v-if="fileType === 'image'"
						:src="fileUrl"
						:style="mediaStyle"
						:preview-src-list="[fileUrl]"
						fit="contain"
						@load="renderedHandler"
						@error="errorHandler"
					/>

					<video
						v-else-if="fileType === 'video'"
						:src="fileUrl"
						controls
						autoplay
						:style="mediaStyle"
						@loadeddata="renderedHandler"
						@error="errorHandler"
					></video>

					<audio
						v-else-if="fileType === 'audio'"
						:src="fileUrl"
						controls
						@loadeddata="renderedHandler"
						@error="errorHandler"
					></audio>

					<div v-else class="unsupported">
						{{ $t("commons.msg.unSupportType") }}
					</div>
				</div>
			</div>
		</n-spin>
	</n-modal>
</template>

<script setup lang="ts">
import { DownloadFile } from "@/api/modules/file"
import { MsgError } from "@/utils/message"
import { computed, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"

const emit = defineEmits(["close"])

const { t } = useI18n()

interface EditProps {
	fileType: string
	path: string
	name: string
	extension: string
}

const open = ref(false)
const loading = ref(false)
const filePath = ref("")
const fileName = ref("")
const fileType = ref("")
const fileUrl = ref("")
const fileExtension = ref("")
const isFullscreen = ref(false)

function revokeFileUrl() {
	if (fileUrl.value) {
		URL.revokeObjectURL(fileUrl.value)
		fileUrl.value = ""
	}
}

function handleClose() {
	revokeFileUrl()
	open.value = false
	emit("close", open.value)
}

function renderedHandler() {
	loading.value = false
}
function errorHandler() {
	loading.value = false
	open.value = false
	MsgError(t("commons.msg.unSupportType"))
}

function loadTooltip() {
	return t(`commons.button.${isFullscreen.value ? "quitFullscreen" : "fullscreen"}`)
}

function toggleFullscreen() {
	isFullscreen.value = !isFullscreen.value
}

async function acceptParams(props: EditProps) {
	fileExtension.value = props.extension
	fileName.value = props.name
	filePath.value = props.path
	fileType.value = props.fileType

	loading.value = true
	open.value = true
	revokeFileUrl()
	try {
		const data = await DownloadFile({ path: props.path })
		fileUrl.value = URL.createObjectURL(new Blob([data]))
	} catch {
		errorHandler()
	}
}

defineExpose({ acceptParams })
onBeforeUnmount(revokeFileUrl)

// 布局样式辅助
const contentStyle = computed(() => {
	return isFullscreen.value ? { height: "90vh" } : { height: "80vh" }
})
const mediaStyle = computed(() => {
	return isFullscreen.value ? { height: "90vh", maxWidth: "100%" } : { height: "80vh", maxWidth: "100%" }
})
</script>

<style scoped lang="scss">
.preview-body {
	padding: 0;
}

.dialog-header-icon {
	color: var(--n-info-500);
}

/* 居中显示占位文案 */
.unsupported {
	color: var(--n-text-3);
	font-size: 14px;
	padding: 12px;
}
</style>
