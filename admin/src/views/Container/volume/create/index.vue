<template>
	<n-drawer v-model:show="drawerVisible" :width="600" placement="right">
		<n-drawer-content :title="t('container.createVolume')" closable>
			<template #header>
				<div class="flex items-center">
					<div class="flex cursor-pointer items-center gap-2 text-gray-500" @click="handleClose">
						<Icon name="mdi:arrow-left" />
						{{ t('commons.button.back') }}
					</div>
					<n-divider vertical />
					{{ t('container.createVolume') }}
				</div>
			</template>

			<n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100" class="py-4">
				<n-form-item :label="t('container.volumeName')" path="name">
					<n-input v-model:value="form.name" :placeholder="t('container.createVolumeNamePlaceholder')" clearable />
				</n-form-item>

				<n-form-item :label="t('container.driver')" path="driver">
					<n-tag type="success">local</n-tag>
				</n-form-item>

				<n-form-item :label="t('container.nfsEnable')" path="nfsStatus">
					<n-switch v-model:value="form.nfsStatus" />
				</n-form-item>

				<template v-if="form.nfsStatus">
					<n-form-item :label="t('container.nfsAddress')" path="nfsAddress">
						<n-input v-model:value="form.nfsAddress" :placeholder="t('container.createNfsAddressPlaceholder')" clearable />
					</n-form-item>

					<n-form-item :label="t('container.nfsVersion')" path="nfsVersion">
						<n-radio-group v-model:value="form.nfsVersion">
							<n-radio-button value="v3">NFS</n-radio-button>
							<n-radio-button value="v4">NFS4</n-radio-button>
						</n-radio-group>
					</n-form-item>

					<n-form-item :label="t('container.mountpoint')" path="nfsMount">
						<n-input v-model:value="form.nfsMount" :placeholder="t('container.createMountpointPlaceholder')" clearable />
					</n-form-item>

					<n-form-item :label="t('container.nfsOptions')" path="nfsOption">
						<n-input v-model:value="form.nfsOption" :placeholder="t('container.createNfsOptionsPlaceholder')" clearable />
					</n-form-item>
				</template>

				<n-form-item :label="t('container.options')" path="optionStr">
					<n-input
						type="textarea"
						v-model:value="form.optionStr"
						:placeholder="t('container.onePerLine')"
						:autosize="{ minRows: 3, maxRows: 5 }"
					/>
				</n-form-item>

				<n-form-item :label="t('container.label')" path="labelStr">
					<n-input
						type="textarea"
						v-model:value="form.labelStr"
						:placeholder="t('container.onePerLine')"
						:autosize="{ minRows: 3, maxRows: 5 }"
					/>
				</n-form-item>
			</n-form>

			<template #footer>
				<n-space>
					<n-button @click="handleClose">{{ t('commons.button.cancel') }}</n-button>
					<n-button type="primary" :loading="loading" @click="handleSubmit">{{ t('commons.button.confirm') }}</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import type { FormInst, FormRules } from "naive-ui"
import { createVolume } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"

const props = defineProps<{
	show: boolean
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "success"): void
}>()

const message = useMessage()
const { t } = useI18n()
const loading = ref(false)
const formRef = ref<FormInst | null>(null)

const drawerVisible = computed({
	get: () => props.show,
	set: value => emit("update:show", value)
})

const form = reactive({
	name: "",
	driver: "local",
	nfsStatus: false,
	nfsAddress: "",
	nfsVersion: "v4",
	nfsMount: "",
	nfsOption: "rw,noatime,rsize=8192,wsize=8192,tcp,timeo=14",
	optionStr: "",
	labelStr: ""
})

const rules: FormRules = {
	name: {
		required: true,
		message: t("container.createVolumeNameRequired"),
		trigger: "blur"
	},
	nfsAddress: {
		required: true,
		message: t("container.createNfsAddressRequired"),
		trigger: "blur"
	},
	nfsMount: {
		required: true,
		message: t("container.createMountpointRequired"),
		trigger: "blur"
	}
}

const handleClose = () => {
	drawerVisible.value = false
}

const handleSubmit = () => {
	formRef.value?.validate(async errors => {
		if (!errors) {
			try {
				loading.value = true
				const params: Container.VolumeCreate = {
					name: form.name,
					driver: form.driver,
					options: form.optionStr ? form.optionStr.split("\n").filter(Boolean) : [],
					labels: form.labelStr ? form.labelStr.split("\n").filter(Boolean) : []
				}

				if (form.nfsStatus) {
					const typeOption = form.nfsVersion === "v4" ? "nfs4" : "nfs"
					params.options.push(`type=${typeOption}`)
					params.options.push(`o=addr=${form.nfsAddress},${form.nfsOption}`)
					const mount = form.nfsMount.startsWith(":") ? form.nfsMount : `:${form.nfsMount}`
					params.options.push(`device=${mount}`)
				}

				await createVolume(params)
				message.success(t("commons.operate.createSuccess"))
				handleClose()
				emit("success")
			} catch (error) {
				console.error(t("commons.operate.createFailed"), error)
				message.error(t("commons.operate.createFailed"))
			} finally {
				loading.value = false
			}
		}
	})
}
</script>

<style scoped>
.n-form-item {
	margin-bottom: 24px;
}
</style>
