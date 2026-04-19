<template>
	<n-drawer v-model:show="open" width="50%" :mask-closable="false" :closable="false">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('file.editPermissions')" :back="handleClose" />
			</template>

			<div style="padding: 16px">
				<n-spin :show="loading">
					<FileRole :mode="mode" @get-mode="getMode" :key="open.toString()" />

					<n-form
						ref="fileForm"
						:model="addForm"
						:rules="rules"
						label-placement="left"
						label-width="100px"
						:show-feedback="true"
					>
						<n-form-item :label="$t('commons.table.user')" path="user">
							<n-input v-model:value="addForm.user" />
						</n-form-item>

						<n-form-item :label="$t('file.group')" path="group">
							<n-input v-model:value="addForm.group" />
						</n-form-item>

						<n-form-item>
							<n-checkbox :checked="addForm.sub" @update:checked="value => (addForm.sub = value)">
								{{ $t("file.containSub") }}
							</n-checkbox>
						</n-form-item>
					</n-form>
				</n-spin>
			</div>

			<template #footer>
				<div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px 16px">
					<n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="loading" @click="submit">
						{{ $t("commons.button.confirm") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import DrawerHeader from "@/components/DrawerHeader.vue"
import { reactive, ref } from "vue"
import type { File } from "@/api/interface/file"
import { BatchChangeRole } from "@/api/modules/file"
import FileRole from "@/components/FileRole.vue"
import { MsgSuccess } from "@/utils/message"
import type { FormInst } from "naive-ui"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

interface BatchRoleProps {
	files: File.File[]
}

const open = ref(false)
const loading = ref(false)
const mode = ref("0755")
const files = ref<File.File[]>([])

const fileForm = ref<FormInst | null>(null)

const addForm = reactive({
	paths: [] as string[],
	mode: 755,
	user: "",
	group: "",
	sub: false
})

// Naive UI 验证规则
const rules = reactive({
	user: [
		{
			validator: (_rule: any, value: string) => {
				if (value && value.trim()) return Promise.resolve()
				return Promise.reject(new Error(t("commons.msg.required") as string))
			},
			trigger: ["blur", "input"]
		}
	],
	group: [
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
	// 手动重置表单字段
	addForm.paths = []
	addForm.mode = 755
	addForm.user = ""
	addForm.group = ""
	addForm.sub = false
	em("close", false)
}

const acceptParams = (props: BatchRoleProps) => {
	addForm.paths = []
	files.value = props.files
	files.value.forEach(file => {
		addForm.paths.push(file.path)
	})
	addForm.mode = Number.parseInt(String(props.files[0].mode), 8)
	addForm.group = String(props.files[0].group || props.files[0].gid)
	addForm.user = String(props.files[0].user || props.files[0].uid)
	addForm.sub = true

	mode.value = String(props.files[0].mode)
	open.value = true
}

const getMode = (val: number) => {
	addForm.mode = val
}

const submit = async () => {
	if (!fileForm.value) return
	try {
		await fileForm.value.validate()
	} catch (e) {
		return
	}

	const regFilePermission = /^[0-7]{3,4}$/
	if (!regFilePermission.test(addForm.mode.toString(8))) {
		return
	}

	loading.value = true
	try {
		await BatchChangeRole(addForm)
		MsgSuccess(t("commons.msg.updateSuccess") as string)
		handleClose()
	} finally {
		loading.value = false
	}
}

defineExpose({ acceptParams })
</script>
