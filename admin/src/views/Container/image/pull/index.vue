<template>
	<n-drawer v-model:show="drawerVisible" :default-width="800" resizable placement="right" @close="handleClose">
		<n-drawer-content :title="'拉取镜像'" closable>
			<template #header>
				<div class="flex items-center">
					<div class="flex cursor-pointer items-center gap-2 text-gray-500" @click="handleClose">
						<Icon icon="mdi:arrow-left" />
						返回
					</div>
					<n-divider vertical />
					拉取镜像
				</div>
			</template>

			<n-form ref="formRef" :model="form" :rules="rules" label-placement="top" class="p-4">
				<n-form-item label="来源" path="fromRepo">
					<n-checkbox v-model:checked="form.fromRepo">镜像仓库</n-checkbox>
				</n-form-item>

				<n-form-item v-if="form.fromRepo" label="仓库名" path="repoID">
					<n-select
						v-model:value="form.repoID"
						:options="repos"
						filterable
						clearable
						placeholder="请选择仓库名"
						@update:value="handleRepoChange"
					/>
				</n-form-item>

				<n-form-item label="镜像名" path="imageName">
					<n-input
						v-model:value="form.imageName"
						:placeholder="form.fromRepo ? '例如：nginx:latest' : '例如：docker.io/library/nginx:latest'"
						@keyup.enter="handleSubmit"
					>
						<template v-if="form.fromRepo && form.downloadUrl" #prefix>
							<span class="text-gray-400">{{ form.downloadUrl }}/</span>
						</template>
					</n-input>
				</n-form-item>
			</n-form>

			<log-file
				v-if="showLog"
				:config="logConfig"
				:default-button="false"
				@update:is-reading="handleLogReading"
			/>

			<template #footer>
				<n-space>
					<n-button @click="handleClose">取消</n-button>
					<n-button type="primary" :disabled="isReading || !form.imageName" @click="handleSubmit">
						拉取
					</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, onUnmounted, watch } from "vue"
import {
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NCheckbox,
	NSelect,
	NInput,
	NButton,
	NSpace,
	NDivider,
	useMessage,
	type FormInst,
	type FormRules
} from "naive-ui"
import { Icon } from "@iconify/vue"
import { imagePull } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"
import LogFile from "@/components/LogFile.vue"

const message = useMessage()
const drawerVisible = ref(false)
const formRef = ref<FormInst | null>(null)
const showLog = ref(false)
const isReading = ref(false)
const logConfig = reactive({
	id: 0,
	type: "image-pull",
	name: "",
	tail: true
})

const form = reactive({
	fromRepo: false,
	repoID: null as number | null,
	imageName: "",
	downloadUrl: ""
})

const rules: FormRules = {
	repoID: [
		{
			required: true,
			message: "请选择仓库名",
			trigger: ["blur", "change"],
			validator: (rule, value) => {
				if (form.fromRepo && !value) {
					return new Error("请选择仓库名")
				}
				return true
			}
		}
	],
	imageName: [{ required: true, message: "请输入镜像名", trigger: "blur" }]
}

interface DialogProps {
	repos: Array<Container.RepoOptions>
}

interface RepoOption {
	label: string
	value: number
	downloadUrl: string
}

const repos = ref<RepoOption[]>([])

const acceptParams = async (params: DialogProps): Promise<void> => {
	drawerVisible.value = true
	form.fromRepo = false
	form.imageName = ""
	form.downloadUrl = ""
	form.repoID = null
	repos.value = params.repos.map(repo => ({
		label: repo.name,
		value: repo.id,
		downloadUrl: repo.downloadUrl
	}))
	isReading.value = false
	showLog.value = false
}

const handleRepoChange = (value: number) => {
	const selectedRepo = repos.value.find(repo => repo.value === value)
	if (selectedRepo) {
		form.downloadUrl = selectedRepo.downloadUrl
	} else {
		form.downloadUrl = ""
	}
}

watch(
	() => form.fromRepo,
	newValue => {
		if (!newValue) {
			form.repoID = null
			form.downloadUrl = ""
		}
	}
)

const emit = defineEmits<{ (e: "search"): void }>()

const handleClose = () => {
	showLog.value = false
	isReading.value = false
	emit("search")
	drawerVisible.value = false
}

const handleLogReading = (reading: boolean) => {
	isReading.value = reading
	if (!reading) {
		emit("search")
	}
}

const handleSubmit = async () => {
	if (!formRef.value) return

	try {
		await formRef.value.validate()
		isReading.value = true
		const res = await imagePull({
			fromRepo: form.fromRepo,
			repoID: form.repoID || 0,
			imageName: form.imageName
			// imageName: form.fromRepo && form.downloadUrl ? `${form.downloadUrl}/${form.imageName}` : form.imageName
		} as Container.ImagePull)

		showLog.value = true
		logConfig.name = res.data
		message.success("镜像拉取任务已开始")
	} catch (error) {
		console.error("拉取镜像失败:", error)
		message.error("拉取镜像失败")
		isReading.value = false
	}
}

defineExpose({
	acceptParams
})
</script>

<style scoped>
.whitespace-pre-wrap {
	white-space: pre-wrap;
	word-wrap: break-word;
}
</style>
