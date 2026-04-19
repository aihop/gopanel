<template>
	<n-drawer v-model:show="open" :closable="true" width="40%" :mask-closable="false">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('file.deCompress')" :resource="name" :back="handleClose" />
			</template>

			<n-spin :show="loading">
				<div class="p-4">
					<n-form ref="fileForm" :model="form" :rules="rules" label-placement="top">
						<div class="mb-4">
							<n-form-item :label="$t('commons.table.name')">
								<n-input v-model:value="name" disabled />
							</n-form-item>
						</div>

						<div class="mb-4">
							<n-form-item :label="$t('file.deCompressDst')" path="dst">
								<n-input v-model:value="form.dst">
									<template #prefix></template>
								</n-input>
							</n-form-item>
						</div>

						<div v-if="name.includes('tar.gz')" class="mb-4">
							<n-form-item :label="$t('setting.compressPassword')" path="secret">
								<n-input v-model:value="form.secret" />
							</n-form-item>
						</div>
					</n-form>
				</div>
			</n-spin>
			<template #footer>
				<n-space>
					<n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="loading" @click="submit">
						{{ $t("commons.button.confirm") }}
					</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import type { File } from "@/api/interface/file"
import type { FormInst } from "naive-ui"
import { DeCompressFile } from "@/api/modules/file"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { Mimetypes } from "@/global/mimetype"
import { MsgSuccess } from "@/utils/message"
import { reactive, ref } from "vue"

import { useI18n } from "vue-i18n"

const emit = defineEmits(["close"])

const { t } = useI18n()

// 表单模型使用 reactive，便于直接绑定模板
const form = reactive<File.FileDeCompress>({
	type: "zip",
	dst: "",
	path: "",
	secret: ""
})

const fileForm = ref<FormInst | null>(null)
const loading = ref(false)
const open = ref(false)
const name = ref("")

const rules = {
	dst: {
		required: true,
		message: String(t("commons.validate.required") || "必填"),
		trigger: ["blur", "change"]
	}
}

function handleClose() {
	fileForm.value?.restoreValidation?.()
	open.value = false
	emit("close", open.value)
}

function getFileType(mime: string): string {
	if (Mimetypes.get(mime) !== undefined) {
		return String(Mimetypes.get(mime))
	} else {
		return ""
	}
}

function getLinkPath(path: string) {
	form.dst = path
}

async function submit() {
	if (!fileForm.value) return
	try {
		await fileForm.value.validate()
	} catch {
		return
	}
	loading.value = true
	DeCompressFile(form as any)
		.then(() => {
			MsgSuccess(t("file.deCompressSuccess"))
			handleClose()
		})
		.finally(() => {
			loading.value = false
		})
}

interface CompressProps {
	files: Array<any>
	dst: string
	name: string
	path: string
	mimeType: string
}

function acceptParams(props: CompressProps) {
	form.type = getFileType(props.mimeType)
	form.dst = props.dst
	form.path = props.path
	name.value = props.name
	open.value = true
}

defineExpose({ acceptParams })
</script>

<style scoped>
.dialog-footer {
	display: flex;
	gap: 12px;
	justify-content: flex-end;
	align-items: center;
}
</style>
