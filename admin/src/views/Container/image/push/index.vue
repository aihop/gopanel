<template>
	<n-drawer
		v-model:show="drawerVisible"
		:destroy-on-close="true"
		@close="onCloseLog"
		:close-on-click-modal="false"
		:close-on-press-escape="false"
		width="400px"
		>
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('container.imagePush')" :back="onCloseLog" />
			</template>
			<n-row type="flex" justify="center">
				<n-col :span="22">
					<n-form ref="formRef" label-position="top" :model="form" label-width="80px" v-if="!logVisible">
						<n-form-item :label="$t('container.tag')" :rules="Rules.requiredSelect" prop="tagName">
							<n-select
								@change="onEdit(true)"
								filterable
								v-model:value="form.tagName"
								:options="form.tags.map(item => ({ label: item, value: item }))"
							/>
						</n-form-item>
						<n-form-item :label="$t('container.repoName')" :rules="Rules.requiredSelect" prop="repoID">
							<n-select
								@change="onEdit()"
								clearable
								style="width: 100%"
								filterable
								v-model:value="form.repoID"
								:options="dialogData.repos.map(item => ({ label: item.name, value: item.id }))"
							/>
						</n-form-item>
						<n-form-item :label="$t('container.image')" :rules="Rules.imageName" prop="name">
							<n-input
								@change="onEdit()"
								v-model:value="form.name"
								:placeholder="`${loadDetailInfo(form.repoID)}/image:tag`"
							/>
						</n-form-item>
					</n-form>

					<LogFile
						ref="logRef"
						:config="logConfig"
						:default-button="false"
						v-model:is-reading="isReading"
						v-if="logVisible"
						:style="'height: calc(100vh - 200px);min-height: 200px'"
						v-model:loading="loading"
					/>
				</n-col>
			</n-row>

			<template #footer>
				<n-space>
					<n-button @click="drawerVisible = false">
						{{ $t("commons.button.cancel") }}
					</n-button>
					<n-button :disabled="isStartReading || isReading" type="primary" @click="onSubmit(formRef)">
						{{ $t("container.push") }}
					</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import { nextTick, reactive, ref } from "vue"
import { Rules } from "@/global/form-rules"
import { t } from "@/i18n"
import { imagePush } from "@/api/modules/container"
import { Container } from "@/api/interface/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { MsgSuccess } from "@/utils/message"

const drawerVisible = ref(false)
const form = reactive({
	tags: [] as Array<string>,
	tagName: "",
	repoID: 1,
	name: ""
})

const logVisible = ref(false)
const loading = ref(false)
const isStartReading = ref(false)
const isReading = ref(false)
const pushVisible = ref(false)

const logRef = ref()
const logConfig = reactive({
	type: "image-push",
	name: ""
})

interface DialogProps {
	repos: Array<Container.RepoOptions>
	tags: Array<string>
}
const dialogData = ref<DialogProps>({
	repos: [] as Array<Container.RepoOptions>,
	tags: [] as Array<string>
})

const acceptParams = async (params: DialogProps): Promise<void> => {
	logVisible.value = false
	loading.value = false
	drawerVisible.value = true
	pushVisible.value = true
	form.tags = params.tags
	form.repoID = 1
	form.tagName = form.tags.length !== 0 ? form.tags[0] : ""
	form.name = form.tags.length !== 0 ? form.tags[0] : ""
	dialogData.value.repos = params.repos
	isStartReading.value = false
}
const emit = defineEmits<{ (e: "search"): void }>()

const formRef = ref<any>()

const onEdit = (isName?: boolean) => {
	if (!isReading.value && isStartReading.value) {
		isStartReading.value = false
	}
	if (isName) {
		form.name = form.tagName
	}
}
const onSubmit = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async (errors: any) => {
		if (errors) return
		const res = await imagePush(form)
		logConfig.name = res.data
		logConfig.tail = true
		logVisible.value = true
		isStartReading.value = true
	})
}

const onCloseLog = async () => {
	emit("search")
	drawerVisible.value = false
}

function loadDetailInfo(id: number) {
	for (const item of dialogData.value.repos) {
		if (item.id === id) {
			return item.downloadUrl
		}
	}
	return ""
}

defineExpose({
	acceptParams
})
</script>
