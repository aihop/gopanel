<template>
	<n-drawer v-model:show="drawerVisible" :width="400" placement="right" :mask-closable="false">
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('container.cutLog')" :back="handleClose" />
			</template>
			<n-alert type="warning" class="mb-4">
				<ul style="margin-left: -20px">
					<li>{{ $t("container.cutLogHelper1") }}</li>
					<li>{{ $t("container.cutLogHelper2") }}</li>
					<li>{{ $t("container.cutLogHelper3") }}</li>
				</ul>
			</n-alert>
			<n-form ref="formRef" :model="form" :rule="rules" label-placement="top">
				<n-form-item :label="$t('container.maxSize')" path="logMaxSize">
					<n-input-group>
						<n-input-number v-model:value="form.logMaxSize" :min="1" :max="1024000" style="width: 180px" />
						<n-select v-model:value="form.sizeUnit" :options="sizeUnitOptions" style="width: 80px" />
					</n-input-group>
				</n-form-item>
				<n-form-item :label="$t('container.maxFile')" path="logMaxFile">
					<n-input-number v-model:value="form.logMaxFile" :min="1" :max="100" style="width: 180px" />
				</n-form-item>
			</n-form>
			<template #footer>
				<n-space justify="end">
					<n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="loading" @click="onSave">
						{{ $t("commons.button.confirm") }}
					</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
	<RebootAlert ref="rebootAlertRef" @confirm="onSubmitSave" />
</template>
<script setup lang="ts">
import { ref, reactive } from "vue"
import {
	useMessage,
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NInputNumber,
	NSelect,
	NButton,
	NAlert,
	NSpace,
	NInputGroup
} from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import RebootAlert from "@/components/RebootAlert.vue"
import { updateLogOption } from "@/api/modules/container"
import { MsgSuccess } from "@/utils/message"

const loading = ref(false)
const drawerVisible = ref(false)
const rebootAlertRef = ref()
const formRef = ref()
const emit = defineEmits(["search"])

const form = reactive({
	logMaxSize: 10,
	logMaxFile: 3,
	sizeUnit: "m"
})
const sizeUnitOptions = [
	{ label: "Byte", value: "b" },
	{ label: "KB", value: "k" },
	{ label: "MB", value: "m" },
	{ label: "GB", value: "g" }
]
const rules = {
	logMaxSize: [{ required: true, type: "number", min: 1, max: 1024000, message: "1-1024000" }],
	logMaxFile: [{ required: true, type: "number", min: 1, max: 100, message: "1-100" }]
}

interface DialogProps {
	logMaxSize: string
	logMaxFile: number
}

const acceptParams = (params: DialogProps): void => {
	form.logMaxFile = params.logMaxFile || 3
	if (params.logMaxSize) {
		form.logMaxSize = loadSize(params.logMaxSize)
	} else {
		form.logMaxSize = 10
		form.sizeUnit = "m"
	}
	drawerVisible.value = true
}

const onSave = async () => {
	formRef.value?.validate((errors: any) => {
		if (!errors) {
			rebootAlertRef.value.open({
				title: "配置修改",
				input: "立即重启",
				msg: "修改配置后需要重启生效。"
			})
		}
	})
}

const onSubmitSave = async () => {
	rebootAlertRef.value.close()
	loading.value = true
	try {
		await updateLogOption(form.logMaxSize + form.sizeUnit, String(form.logMaxFile))
		loading.value = false
		drawerVisible.value = false
		emit("search")
		MsgSuccess("操作成功")
	} catch {
		loading.value = false
	}
}

const loadSize = (value: string) => {
	if (value.includes("k") || value.includes("KB")) {
		form.sizeUnit = "k"
		return Number(value.replace(/k|KB/g, ""))
	}
	if (value.includes("m") || value.includes("MB")) {
		form.sizeUnit = "m"
		return Number(value.replace(/m|MB/g, ""))
	}
	if (value.includes("g") || value.includes("GB")) {
		form.sizeUnit = "g"
		return Number(value.replace(/g|GB/g, ""))
	}
	if (value.includes("b") || value.includes("B")) {
		form.sizeUnit = "b"
		return Number(value.replace(/b|B/g, ""))
	}
	return Number(value)
}

const handleClose = () => {
	emit("search")
	drawerVisible.value = false
}

defineExpose({
	acceptParams
})
</script>
