<template>
	<n-drawer
		v-model:show="drawerVisible"
		:destroy-on-close="true"
		:close-on-click-modal="false"
		:close-on-press-escape="false"
		size="30%"
	>
		<n-drawer-content>
			<template #header>
				<DrawerHeader
					:header="title + $t('container.repo').toLowerCase()"
					:resource="dialogData.rowData?.name"
					:back="handleClose"
				/>
			</template>
			<n-form
				ref="formRef"
				label-position="top"
				v-loading="loading"
				:model="dialogData.rowData"
				:rules="rules"
				label-width="120px"
			>
				<n-row type="flex" justify="center">
					<n-col :span="22">
						<n-form-item :label="$t('commons.table.name')" prop="name">
							<n-input
								clearable
								:disabled="dialogData.title === 'edit'"
								v-model.trim="dialogData.rowData!.name"
							></n-input>
						</n-form-item>
						<n-form-item :label="$t('container.auth')" prop="auth">
							<n-radio-group v-model="dialogData.rowData!.auth">
								<n-radio :value="true">{{ $t("commons.true") }}</n-radio>
								<n-radio :value="false">{{ $t("commons.false") }}</n-radio>
							</n-radio-group>
						</n-form-item>
						<n-form-item
							v-if="dialogData.rowData!.auth"
							:label="$t('commons.login.username')"
							prop="username"
						>
							<n-input clearable v-model.trim="dialogData.rowData!.username"></n-input>
						</n-form-item>
						<n-form-item
							v-if="dialogData.rowData!.auth"
							:label="$t('commons.login.password')"
							prop="password"
						>
							<n-input
								clearable
								type="password"
								show-password
								v-model.trim="dialogData.rowData!.password"
							></n-input>
						</n-form-item>
						<n-form-item :label="$t('container.downloadUrl')" prop="downloadUrl">
							<n-input
								clearable
								v-model.trim="dialogData.rowData!.downloadUrl"
								:placeholder="'172.16.10.10:8081'"
							></n-input>
							<span v-if="dialogData.rowData!.downloadUrl" class="input-help">
								docker pull {{ dialogData.rowData!.downloadUrl }}/nginx
							</span>
						</n-form-item>
						<n-form-item :label="$t('commons.table.protocol')" prop="protocol">
							<n-radio-group v-model="dialogData.rowData!.protocol">
								<n-radio label="http">http</n-radio>
								<n-radio label="https">https</n-radio>
							</n-radio-group>
							<span v-if="dialogData.rowData!.protocol === 'http'" class="input-help">
								{{ $t("container.httpRepo") }}
							</span>
						</n-form-item>
					</n-col>
				</n-row>
			</n-form>

			<template #footer>
				<span class="dialog-footer">
					<n-button :disabled="loading" @click="drawerVisible = false">
						{{ $t("commons.button.cancel") }}
					</n-button>
					<n-button :disabled="loading" type="primary" @click="onSubmit(formRef)">
						{{ $t("commons.button.confirm") }}
					</n-button>
				</span>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue"
import { Rules } from "@/global/form-rules"
import { Container } from "@/api/interface/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { createImageRepo, updateImageRepo } from "@/api/modules/container"
import { MsgSuccess } from "@/utils/message"
import { t } from "@/i18n"

const loading = ref(false)

interface DialogProps {
	title: string
	rowData?: Container.RepoInfo
	getTableList?: () => Promise<any>
}
const title = ref<string>("")
const drawerVisible = ref(false)
const dialogData = ref<DialogProps>({
	title: ""
})
const acceptParams = (params: DialogProps): void => {
	dialogData.value = params
	title.value = t("commons.button." + dialogData.value.title)
	drawerVisible.value = true
}
const emit = defineEmits<{ (e: "search"): void }>()

const handleClose = () => {
	drawerVisible.value = false
}
const rules = reactive({
	name: [Rules.requiredInput, Rules.name],
	downloadUrl: [{ validator: validateDownloadUrl, trigger: "blur" }, Rules.illegal],
	protocol: [Rules.requiredSelect],
	username: [Rules.illegal],
	password: [Rules.illegal],
	auth: [Rules.requiredSelect]
})

const formRef = ref<any>()

function validateDownloadUrl(rule: any, value: any, callback: any) {
	if (value === "") {
		callback()
	}
	const pattern = /^(http:\/\/|https:\/\/)/i
	if (pattern.test(value)) {
		return callback(new Error(t("container.urlWarning")))
	}
	callback()
}

const onSubmit = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		loading.value = true
		if (dialogData.value.title === "add") {
			await createImageRepo(dialogData.value.rowData!)
				.then(() => {
					loading.value = false
					MsgSuccess(t("commons.msg.operationSuccess"))
					emit("search")
					drawerVisible.value = false
				})
				.catch(() => {
					loading.value = false
				})
			return
		}
		await updateImageRepo(dialogData.value.rowData!)
			.then(() => {
				loading.value = false
				MsgSuccess(t("commons.msg.operationSuccess"))
				emit("search")
				drawerVisible.value = false
			})
			.catch(() => {
				loading.value = false
			})
	})
}

defineExpose({
	acceptParams
})
</script>
