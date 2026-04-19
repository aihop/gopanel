<template>
	<n-drawer v-model:show="open" width="40%" :mask-closable="false" :closable="false">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="t('file.rename')" :resource="oldName" :back="handleClose" />
			</template>

			<div style="padding: 16px">
				<n-form ref="fileForm" :model="addForm" :rules="rules" label-placement="top" :show-feedback="true">
					<n-form-item :label="t('file.path')" path="path">
						<n-input v-model:value="addForm.path" disabled />
					</n-form-item>

					<n-form-item :label="t('commons.table.name')" path="newName">
						<n-input v-model:value="addForm.newName" />
					</n-form-item>
				</n-form>
			</div>

			<template #footer>
				<div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px 16px">
					<n-button @click="handleClose">{{ t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="loading" @click="submit">
						{{ t("commons.button.confirm") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue"
import { useI18n } from "vue-i18n"
import type { FormInst } from "naive-ui"
import { RenameRile } from "@/api/modules/file"
import type { File } from "@/api/interface/file"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { MsgSuccess } from "@/utils/message"

const { t } = useI18n()

interface RenameProps {
	path: string
	oldName: string
}

const fileForm = ref<FormInst | null>(null)
const loading = ref(false)
const open = ref(false)
const oldName = ref("")

const addForm = reactive({
	newName: "",
	path: ""
})

// Naive UI 验证规则：newName 必填
const rules = reactive({
	newName: [
		{
			validator: (_rule: any, value: string) => {
				if (value && value.trim()) return Promise.resolve()
				return Promise.reject(new Error(t("commons.msg.required") as string))
			},
			trigger: ["blur", "input"]
		}
	]
})

const em = defineEmits(["close"])
const handleClose = () => {
	open.value = false
	// 手动重置表单数据
	addForm.newName = ""
	addForm.path = ""
	em("close", false)
}

const getPath = (path: string, name: string) => {
	if (!path) return name
	return path.endsWith("/") ? `${path}${name}` : `${path}/${name}`
}

const submit = async () => {
	if (!fileForm.value) return
	try {
		await fileForm.value.validate()
	} catch {
		// 验证失败
		return
	}

	const payload: File.FileRename = {
		oldName: getPath(addForm.path, oldName.value),
		newName: getPath(addForm.path, addForm.newName)
	}

	loading.value = true
	try {
		await RenameRile(payload)
		MsgSuccess(t("commons.msg.updateSuccess") as string)
		handleClose()
	} finally {
		loading.value = false
	}
}

const acceptParams = (props: RenameProps) => {
	oldName.value = props.oldName
	addForm.newName = props.oldName
	addForm.path = props.path
	open.value = true
}

defineExpose({ acceptParams })
</script>
