<template>
	<n-drawer v-model:show="open" :mask-closable="false" width="50%" :destroy-on-close="true">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="title" :back="handleClose" />
			</template>

			<div style="padding: 16px">
				<n-form ref="fileFormRef" :model="form" label-placement="top">
					<n-form-item :label="t('file.compressType')" path="type">
						<n-select v-model:value="form.type" :options="options" />
					</n-form-item>

					<n-form-item :label="t('commons.table.name')" path="name">
						<div class="flex items-center gap-2">
							<n-input v-model:value="form.name" class="flex-1" />
							<span class="text-sm text-slate-500">{{ extension }}</span>
						</div>
					</n-form-item>

					<n-form-item :label="t('file.compressDst')" path="dst">
						<n-input v-model:value="form.dst">
							<template #prefix></template>
						</n-input>
					</n-form-item>

					<n-form-item v-if="form.type === 'tar.gz'">
						<n-input v-model:value="form.secret" :placeholder="t('setting.compressPassword')" />
					</n-form-item>

					<n-form-item>
						<n-checkbox :checked="form.replace" @update:checked="value => (form.replace = value)">
							{{ t("file.replace") }}
						</n-checkbox>
					</n-form-item>
				</n-form>
			</div>

			<template #footer>
				<div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px">
					<n-button @click="handleClose">{{ t("commons.button.cancel") }}</n-button>
					<n-button type="primary" @click="submit" :loading="loading">
						{{ t("commons.button.confirm") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from "vue"
import { useI18n } from "vue-i18n"
import { NDrawer, NForm, NFormItem, NSelect, NInput, NCheckbox, NButton } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { fileCompressAPI } from "@/api/modules/file"
import { CompressExtension, CompressType } from "@/enums/files"
import { MsgSuccess, MsgError } from "@/utils/message"
import type { File as FileType } from "@/api/interface/file"
import { i18n } from "@/i18n"

interface CompressProps {
	files: Array<any>
	dst: string
	name: string
	operate: string
}

const { t } = useI18n()

const fileFormRef = ref<any>(null)
const loading = ref(false)

const form = reactive<FileType.FileCompress>({
	files: [],
	type: "zip",
	dst: "",
	name: "",
	replace: false,
	secret: ""
})

const options = ref<{ label: string; value: string }[]>([])
const open = ref(false)
const title = ref("")
const operate = ref("compress")
const emit = defineEmits(["close"])

const extension = computed(() => CompressExtension[form.type as keyof typeof CompressExtension] || "")

const handleClose = () => {
	// reset form
	form.files = []
	form.type = "zip"
	form.dst = ""
	form.name = ""
	form.replace = false
	form.secret = ""
	open.value = false
	emit("close", open)
}

const validate = () => {
	if (!form.type) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	if (!form.dst) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	if (!form.name) {
		MsgError(t("commons.msg.invalid"))
		return false
	}
	return true
}

const submit = async () => {
	if (!validate()) return
	const payload = { ...form, name: form.name + extension.value }
	loading.value = true
	try {
		await fileCompressAPI(payload as FileType.FileCompress)
		MsgSuccess(t("file.compressSuccess"))
		handleClose()
	} catch (e) {
		// 可选地展示错误
		console.error(e)
	} finally {
		loading.value = false
	}
}

const acceptParams = (props: CompressProps) => {
	form.files = props.files
	form.dst = props.dst
	form.name = props.name
	operate.value = props.operate

	options.value = Object.keys(CompressType)
		.map(k => CompressType[k as keyof typeof CompressType])
		.filter(Boolean)
		.map(v => ({ label: v, value: v }))

	title.value = t("file." + props.operate)
	open.value = true
}

defineExpose({ acceptParams })
</script>
