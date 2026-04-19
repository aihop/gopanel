<template>
	<n-drawer v-model:show="open" width="40%" :mask-closable="false" :closable="false">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('file.changeOwner')" :resource="name" :back="handleClose" />
			</template>

			<div style="padding: 16px">
				<n-alert type="info" :title="$t('file.ownerHelper')" :closable="false" class="common-prompt" />

				<n-form ref="fileForm" :model="addForm" :rules="rules" label-placement="top" :show-feedback="true">
					<n-form-item :label="$t('commons.table.user')" path="user">
						<n-input v-model:value="addForm.user" />
					</n-form-item>

					<n-form-item :label="$t('file.group')" path="group">
						<n-input v-model:value="addForm.group" />
					</n-form-item>

					<n-form-item v-if="isDir">
						<n-checkbox :checked="addForm.sub" @update:checked="value => (addForm.sub = value)">
							{{ $t("file.containSub") }}
						</n-checkbox>
					</n-form-item>
				</n-form>
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

<script lang="ts" setup>
import { ChangeOwner } from "@/api/modules/file"
import { reactive, ref } from "vue"
import { MsgSuccess } from "@/utils/message"
import DrawerHeader from "@/components/DrawerHeader.vue"
import type { FormInst } from "naive-ui"
import { useI18n } from "vue-i18n"
const { t } = useI18n()

interface OwnerProps {
	path: string
	user: string
	group: string
	isDir: boolean
	name: string
}

const fileForm = ref<FormInst>()
const loading = ref(false)
const open = ref(false)
const isDir = ref(false)
const name = ref("")

const addForm = reactive({
	path: "",
	user: "",
	group: "",
	sub: false
})

// Naive UI 验证器：返回 Promise，失败时抛出 Error
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
	if (fileForm.value) {
		// Naive UI 没有 resetFields 方法，手动重置表单数据
		addForm.user = ""
		addForm.group = ""
		addForm.sub = false
	}
	em("close", false)
}

const submit = async () => {
	if (!fileForm.value) return
	try {
		await fileForm.value.validate()
	} catch (e) {
		// 验证失败，直接返回
		return
	}
	loading.value = true
	try {
		await ChangeOwner(addForm)
		MsgSuccess(t("commons.msg.updateSuccess") as string)
		handleClose()
	} finally {
		loading.value = false
	}
}

const acceptParams = (props: OwnerProps) => {
	addForm.user = props.user
	addForm.path = props.path
	addForm.group = props.group
	isDir.value = props.isDir
	name.value = props.name
	open.value = true
}

defineExpose({ acceptParams })
</script>
