<template>
	<n-modal
		v-model:show="dialogVisible"
		:mask-closable="false"
		preset="dialog"
		:title="$t('container.imagePrune')"
		:style="{ width: '500px' }"
	>
		<n-form ref="formRef" :model="formValue" :loading="loading">
			<n-form-item>
				<n-radio-group v-model:value="formValue.withTagAll">
					<n-space>
						<n-radio :value="false">{{ $t("container.imagePruneSome") }}</n-radio>
						<n-radio :value="true">{{ $t("container.imagePruneAll") }}</n-radio>
					</n-space>
				</n-radio-group>
			</n-form-item>

			<n-text v-if="formValue.withTagAll">
				{{ unUsedList.length !== 0 ? $t("container.imagePruneAllHelper") : $t("container.imagePruneAllEmpty") }}
			</n-text>
			<n-text v-else class="text-gray-500">
				{{
					unTagList.length !== 0 ? $t("container.imagePruneSomeHelper") : $t("container.imagePruneSomeEmpty")
				}}
			</n-text>

			<n-space vertical v-if="!formValue.withTagAll">
				<n-tag v-for="(item, index) in unTagList" :key="index" :bordered="false" size="small">
					{{ item.tags && item.tags[0] ? item.tags[0] : item.id.replace(/sha256:/g, "").substring(0, 12) }}
				</n-tag>
			</n-space>

			<n-space vertical v-else>
				<n-tag v-for="(item, index) in unUsedList" :key="index" :bordered="false" size="small">
					{{
						item.tags && item.tags[0]
							? item.tags.join(", ")
							: item.id.replace(/sha256:/g, "").substring(0, 12)
					}}
				</n-tag>
			</n-space>
		</n-form>

		<template #action>
			<n-space>
				<n-button @click="dialogVisible = false">
					{{ $t("commons.button.cancel") }}
				</n-button>
				<n-button type="primary" :disabled="buttonDisable() || loading" @click="onClean">
					{{ $t("commons.button.confirm") }}
				</n-button>
			</n-space>
		</template>
	</n-modal>
</template>

<script lang="ts" setup>
import { ref, reactive } from "vue"
import { useMessage } from "naive-ui"
import { containerPrune, listAllImage } from "@/api/modules/container"
import { computeSize } from "@/utils/util"
import type { Container } from "@/api/interface/container"

const message = useMessage()
const dialogVisible = ref(false)
const loading = ref(false)
const unTagList = ref<Container.ImageInfo[]>([])
const unUsedList = ref<Container.ImageInfo[]>([])

const formValue = reactive({
	withTagAll: false
})

const emit = defineEmits<{ (e: "search"): void }>()

const acceptParams = async (): Promise<void> => {
	try {
		const res = await listAllImage()
		const list = res.data || []
		unTagList.value = []
		unUsedList.value = []

		for (const item of list) {
			if (
				!item.tags ||
				item.tags.length === 0 ||
				(item.tags.length === 1 && item.tags[0].indexOf("<none>") !== -1 && !item.isUsed)
			) {
				unTagList.value.push(item)
			}
			if (!item.isUsed) {
				unUsedList.value.push(item)
			}
		}

		dialogVisible.value = true
		formValue.withTagAll = false
	} catch (error: any) {
		// 错误提示由请求拦截器统一处理
	}
}

const buttonDisable = () => {
	return formValue.withTagAll ? unUsedList.value.length === 0 : unTagList.value.length === 0
}

const onClean = async () => {
	loading.value = true
	try {
		const params = {
			pruneType: "image",
			withTagAll: formValue.withTagAll
		}

		const res = await containerPrune(params)
		if (res.code === 0) {
			message.success(
				`清理成功，共删除 ${res.data.deletedNumber} 个镜像，释放空间 ${computeSize(res.data.spaceReclaimed)}`
			)
			dialogVisible.value = false
			emit("search")
		} else {
			message.error(res.msg || "清理镜像失败")
		}
	} catch (error: any) {
		// 错误提示由请求拦截器统一处理
	} finally {
		loading.value = false
	}
}

defineExpose({
	acceptParams
})
</script>
