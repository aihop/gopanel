<template>
	<n-drawer v-model:show="drawerVisible" :width="600" placement="right">
		<n-drawer-content title="创建存储卷" closable>
			<template #header>
				<div class="flex items-center">
					<div class="flex cursor-pointer items-center gap-2 text-gray-500" @click="handleClose">
						<Icon name="mdi:arrow-left" />
						返回
					</div>
					<n-divider vertical />
					创建存储卷
				</div>
			</template>

			<n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100" class="py-4">
				<n-form-item label="名称" path="name">
					<n-input v-model:value="form.name" placeholder="请输入存储卷名称" clearable />
				</n-form-item>

				<n-form-item label="驱动" path="driver">
					<n-tag type="success">local</n-tag>
				</n-form-item>

				<n-form-item label="启用 NFS" path="nfsStatus">
					<n-switch v-model:value="form.nfsStatus" />
				</n-form-item>

				<template v-if="form.nfsStatus">
					<n-form-item label="NFS 地址" path="nfsAddress">
						<n-input v-model:value="form.nfsAddress" placeholder="请输入 NFS 服务器地址" clearable />
					</n-form-item>

					<n-form-item label="NFS 版本" path="nfsVersion">
						<n-radio-group v-model:value="form.nfsVersion">
							<n-radio-button value="v3">NFS</n-radio-button>
							<n-radio-button value="v4">NFS4</n-radio-button>
						</n-radio-group>
					</n-form-item>

					<n-form-item label="挂载点" path="nfsMount">
						<n-input v-model:value="form.nfsMount" placeholder="请输入挂载点" clearable />
					</n-form-item>

					<n-form-item label="NFS 选项" path="nfsOption">
						<n-input v-model:value="form.nfsOption" placeholder="请输入 NFS 选项" clearable />
					</n-form-item>
				</template>

				<n-form-item label="选项" path="optionStr">
					<n-input
						type="textarea"
						v-model:value="form.optionStr"
						placeholder="一行一个选项"
						:autosize="{ minRows: 3, maxRows: 5 }"
					/>
				</n-form-item>

				<n-form-item label="标签" path="labelStr">
					<n-input
						type="textarea"
						v-model:value="form.labelStr"
						placeholder="一行一个标签"
						:autosize="{ minRows: 3, maxRows: 5 }"
					/>
				</n-form-item>
			</n-form>

			<template #footer>
				<n-space>
					<n-button @click="handleClose">取消</n-button>
					<n-button type="primary" :loading="loading" @click="handleSubmit">确认</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue"
import { useMessage } from "naive-ui"
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
		message: "请输入存储卷名称",
		trigger: "blur"
	},
	nfsAddress: {
		required: true,
		message: "请输入 NFS 服务器地址",
		trigger: "blur"
	},
	nfsMount: {
		required: true,
		message: "请输入挂载点",
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
				message.success("创建成功")
				handleClose()
				emit("success")
			} catch (error) {
				console.error("创建失败:", error)
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
