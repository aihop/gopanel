<template>
	<n-drawer
		v-model:show="drawerVisible"
		:width="'50%'"
		:close-on-esc="false"
		:mask-closable="false"
		:trap-focus="false"
		:block-scroll="false"
		@update:show="
			val => {
				if (!val) handleClose()
			}
		"
	>
		<n-drawer-content :title="$t('commons.button.upgrade')" closable>
			<template #header>
				<DrawerHeader
					:header="$t('commons.button.upgrade')"
					:resource="form.containerName"
					:back="handleClose"
				/>
			</template>
			<n-spin :show="loading">
				<n-form ref="formRef" :model="form" label-placement="top">
					<n-form-item :label="$t('container.oldImage')" path="oldImageName">
						<n-tooltip v-if="form.oldImageName.length > 50" placement="top-start">
							<template #trigger>
								<n-tag>{{ form.oldImageName.substring(0, 50) }}...</n-tag>
							</template>
							{{ form.oldImageName }}
						</n-tooltip>
						<n-tag v-else>{{ form.oldImageName }}</n-tag>
					</n-form-item>
					<n-form-item :label="$t('container.targetImage')" path="newImageName" :rule="Rules.imageName">
						<n-input v-model:value="form.newImageName" />
						<span class="input-help">{{ $t("container.upgradeHelper") }}</span>
						<span v-if="!form.hasName" class="input-help" style="color: #f00">
							{{ " (" + $t("container.imageLoadErr") + ")" }}
						</span>
					</n-form-item>
					<n-form-item path="forcePull">
						<n-checkbox v-model:checked="form.forcePull">
							{{ $t("container.forcePull") }}
						</n-checkbox>
						<span class="input-help">{{ $t("container.forcePullHelper") }}</span>
					</n-form-item>
					<n-form-item v-if="form.fromApp">
						<n-alert type="warning" :show-icon="true">
							{{ $t("container.appHelper") }}
						</n-alert>
					</n-form-item>
				</n-form>
			</n-spin>
			<template #footer>
				<span class="dialog-footer" style="display: flex; justify-content: flex-end">
					<n-button :disabled="loading" @click="drawerVisible = false" style="margin-right: 8px">
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
import { upgradeContainer } from "@/api/modules/container"
import { Rules } from "@/global/form-rules"
import { i18n } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { ref, reactive } from "vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { useDialog } from "naive-ui"

const { t } = useI18n()

const loading = ref(false)
const dialog = useDialog()

const form = reactive({
	containerName: "",
	oldImageName: "",
	newImageName: "",
	hasName: true,
	fromApp: false,
	forcePull: false
})

const formRef = ref()
const drawerVisible = ref<boolean>(false)

interface DialogProps {
	container: string
	image: string
	fromApp: boolean
}
const acceptParams = (props: DialogProps): void => {
	form.containerName = props.container
	form.oldImageName = props.image
	form.fromApp = props.fromApp
	form.hasName = props.image.indexOf("sha256:") === -1
	if (form.hasName) {
		form.newImageName = props.image
	} else {
		form.newImageName = ""
	}
	drawerVisible.value = true
}
const emit = defineEmits<{ (e: "search"): void }>()

const onSubmit = async (formEl: any) => {
	if (!formEl) return
	await formEl.validate(async (errors: any) => {
		if (errors) return
		dialog.warning({
			title: t("commons.button.upgrade"),
			content: t("container.upgradeWarning2"),
			positiveText: t("commons.button.confirm"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: async () => {
				loading.value = true
				try {
					await upgradeContainer(form.containerName, form.newImageName, form.forcePull)
					loading.value = false
					emit("search")
					drawerVisible.value = false
					MsgSuccess(t("commons.msg.operationSuccess"))
				} catch {
					loading.value = false
				}
			}
		})
	})
}
const handleClose = async () => {
	drawerVisible.value = false
	emit("search")
}

defineExpose({
	acceptParams
})
</script>

<style scoped>
.input-help {
	font-size: 12px;
	color: #888;
	margin-left: 8px;
}
</style>
